package main

import (
	"context"
	"time"

	"githhub.com/fathimasithara/user-service/internal/config"
	"githhub.com/fathimasithara/user-service/internal/handler"
	"githhub.com/fathimasithara/user-service/internal/middleware"
	"githhub.com/fathimasithara/user-service/internal/repository"
	"githhub.com/fathimasithara/user-service/internal/routes"
	"githhub.com/fathimasithara/user-service/internal/service"
	"githhub.com/fathimasithara/user-service/internal/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		logger.Fatal("failed to connect to MongoDB", zap.Error(err))
	}
	db := client.Database(cfg.Mongo.Database)

	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)

	jwtManager := utils.NewJWTManager(cfg.App.JWT.Secret)

	userService := service.NewUserService(userRepo, logger, jwtManager, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays)

	userHandler := handler.NewUserHandler(userService, logger)

	app := fiber.New()

	jwtMiddleware := middleware.JWTMiddleware(jwtManager, logger)

	routes.RegisterRoutes(app, userHandler, jwtMiddleware)

	logger.Info("starting server", zap.Int("port", cfg.App.Port))
	if err := app.Listen(":8080"); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}
}
