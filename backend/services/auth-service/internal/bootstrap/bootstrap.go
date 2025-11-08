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
	"github.com/fathima-sithara/auth-service/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

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

type CleanupFn func(context.Context)

func Init() (*AppContext, CleanupFn, error) {
	cfg, err := config.Load("internal/config/config.yaml")
	if err != nil {
		return nil, nil, err
	}

	var logger *zap.Logger
	if cfg.App.Env == "development" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	sugar := logger.Sugar()

	app := &AppContext{Config: cfg, Logger: logger, Sugar: sugar}
	sugar.Infof("Starting service in %s environment", cfg.App.Env)

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

	app.Twilio = twilio.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.From)
	app.EmailJS = emailJS.NewClient(cfg.EmailJS.PublicKey, cfg.EmailJS.PrivateKey, cfg.EmailJS.ServiceID, cfg.EmailJS.TemplateID)

	jwtMgr := utils.NewJWTManager(
		cfg.App.JWT.PrivateKeyPath,
		cfg.App.JWT.PublicKeyPath,
		cfg.App.JWT.AccessTTLMinutes,
		cfg.App.JWT.RefreshTTLDays,
	)

	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)
	authSvc := services.NewAuthService(userRepo, app.Twilio, app.EmailJS, rdb, jwtMgr, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour, logger)
	app.Handler = handlers.NewHandler(authSvc, logger)

	return app, func(ctx context.Context) {
		if cerr := logger.Sync(); cerr != nil {
			log.Printf("Logger sync error: %v", cerr)
		}

		if cerr := mongoClient.Disconnect(ctx); cerr != nil {
			app.Sugar.Errorf("MongoDB disconnect error: %v", cerr)
		}

		if cerr := rdb.Close(); cerr != nil {
			app.Sugar.Errorf("Redis client close error: %v", cerr)
		}
	}, nil
}
