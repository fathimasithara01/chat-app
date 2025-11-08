    package routes

    import (
        "github.com/fathima-sithara/chat-service/config"
        "github.com/fathima-sithara/chat-service/internal/cache"
        "github.com/fathima-sithara/chat-service/internal/handlers"
        "github.com/fathima-sithara/chat-service/internal/kafka"
        "github.com/fathima-sithara/chat-service/internal/repository"
        "github.com/fathima-sithara/chat-service/internal/ws"
        "github.com/gofiber/fiber/v2"
    )

    func Register(app *fiber.App, cfg *config.Config, repo *repository.MongoRepository, kp *kafka.Producer, redisClient *cache.Client, hub *ws.Hub, kc *kafka.Consumer) {
        api := app.Group("/api/v1")

        api.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

        chat := api.Group("/chat")
        h := handlers.NewChatHandler(repo, hub)

        chat.Post("/conversations", h.CreateConversation)
        chat.Post("/messages", h.SendMessage)
        chat.Get("/conversations/:id/messages", h.GetMessages)

        go kc.Run(hub)
    }
