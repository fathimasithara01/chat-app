package routes

import (
	"context"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/fathima-sithara/chat-service/internal/cache"
	handlers "github.com/fathima-sithara/chat-service/internal/handler"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// routes/routes.go
func Register(
	app *fiber.App,
	cfg *config.Config,
	repo *repository.MongoRepository,
	prod *kafka.Producer,
	redisClient *cache.Client,
	hub *ws.Hub,
	consumer *kafka.Consumer,
	jwtMw fiber.Handler, 
) {

	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// User routes
	users := api.Group("/users")
	users.Use(jwtMw) // protect all user routes
	userHandler := handlers.NewUserHandler(repo, hub, redisClient)
	users.Get("/online", userHandler.GetOnlineUsers)
	users.Get("/search", userHandler.SearchUsers)

	// Chat routes
	chats := api.Group("/chats")
	chats.Use(jwtMw)
	chatHandler := handlers.NewChatHandler(repo, hub, prod, redisClient)
	chats.Post("/create", chatHandler.CreateChat)
	chats.Get("/", chatHandler.GetUserChats)
	chats.Get("/:chat_id", chatHandler.GetChatDetails)
	chats.Delete("/:chat_id", chatHandler.DeleteChat)

	// Messages
	messages := api.Group("/messages")
	messages.Use(jwtMw)
	messagesHandler := handlers.NewMessageHandler(repo, hub, prod)
	messages.Post("/send", messagesHandler.SendMessage)
	messages.Get("/:chat_id", messagesHandler.GetMessages)
	messages.Put("/:message_id/edit", messagesHandler.EditMessage)
	messages.Delete("/:message_id", messagesHandler.DeleteMessage)

	// Typing / Status
	status := api.Group("/status")
	status.Use(jwtMw)
	statusHandler := handlers.NewStatusHandler(hub, repo)
	status.Post("/typing", statusHandler.UpdateTypingStatus)
	status.Post("/read", statusHandler.MarkAsRead)

	// WebSocket endpoint
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		hub.HandleWebsocket(c) // now c is *websocket.Conn
	}))

	// Start Kafka consumer
	msgChan := make(chan map[string]any)
	go consumer.Run(context.Background(), msgChan)
	go func() {
		for msg := range msgChan {
			hub.BroadcastJSON(msg)
		}
	}()
}
