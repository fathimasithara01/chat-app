package bootstrap

import (
	"context"
	"log"

	"github.com/fathima-sithara/auth-service/internal/config"
	"github.com/fathima-sithara/auth-service/internal/database"
	"github.com/fathima-sithara/auth-service/internal/emailJS"
	"github.com/fathima-sithara/auth-service/internal/handlers"
	"github.com/fathima-sithara/auth-service/internal/repository"
	"github.com/fathima-sithara/auth-service/internal/services"
	"github.com/fathima-sithara/auth-service/internal/twilio"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// AppContext holds all initialized application components.
type AppContext struct {
	Config  *config.Config
	Logger  *zap.Logger
	Sugar   *zap.SugaredLogger
	Mongo   *mongo.Client
	Redis   *redis.Client
	Twilio  *twilio.Client
	EmailJS *emailJS.Client
	Handler *handlers.Handler
}

// CleanupFn is a function that cleans up resources.
type CleanupFn func(context.Context)

func Init() (*AppContext, CleanupFn, error) {
	// 1. Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return nil, nil, err
	}

	// 2. Initialize logger
	var logger *zap.Logger
	if cfg.App.Env == "development" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	sugar := logger.Sugar()

	app := &AppContext{Config: cfg, Logger: logger, Sugar: sugar}
	sugar.Infof("Starting service in %s environment", cfg.App.Env)

	// 3. Database connections
	db, mongoClient, err := database.ConnectMongo(cfg.Mongo.URI, cfg.Mongo.Database, sugar)
	if err != nil {
		return nil, nil, err
	}
	app.Mongo = mongoClient

	rdb, err := database.ConnectRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, sugar)
	if err != nil {
		return nil, nil, err
	}
	app.Redis = rdb

	// 4. External clients
	app.Twilio = twilio.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.From)
	app.EmailJS = emailJS.NewClient(cfg.EmailJS.PublicKey, cfg.EmailJS.PrivateKey, cfg.EmailJS.ServiceID, cfg.EmailJS.TemplateID)

	// 5. Application layers
	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)
	authSvc := services.NewAuthService(userRepo, app.Twilio, app.EmailJS, rdb, cfg.App.JWT.Secret, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour, logger)
	app.Handler = handlers.NewHandler(authSvc, logger)

	// Return the application context and a cleanup function
	return app, func(ctx context.Context) {
		// Flush logger
		if cerr := logger.Sync(); cerr != nil {
			log.Printf("Logger sync error: %v", cerr)
		}

		// Disconnect MongoDB
		if cerr := mongoClient.Disconnect(ctx); cerr != nil {
			app.Sugar.Errorf("MongoDB disconnect error: %v", cerr)
		}

		// Close Redis connection
		if cerr := rdb.Close(); cerr != nil {
			app.Sugar.Errorf("Redis client close error: %v", cerr)
		}
	}, nil
}
