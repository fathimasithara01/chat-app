package main

import (
	"log"
	"net/http"
	"user-service/internal/config"
	"user-service/internal/handler"
	"user-service/internal/pkg/db"
	"user-service/internal/repository"
	"user-service/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.LoadConfig()
	col := db.ConnectMongo(cfg)

	repo := repository.NewUserMongoRepository(col)
	userService := service.NewUserService(repo)
	userHandler := handler.NewUserHandler(userService)

	r := chi.NewRouter()
	r.Post("/users", userHandler.CreateUser)

	log.Println("User Service running on port:", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, r))
}
