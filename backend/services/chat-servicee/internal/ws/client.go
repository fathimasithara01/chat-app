package ws

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func NewWebsocketHandler(hub *Hub) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		hub.HandleWebsocket(c)
	})
}

// Client represents a connected user
type Client struct {
	ID   string          // user ID
	Hub  *Hub            // reference to hub
	Conn *websocket.Conn // websocket connectionMarkUserOnline
	Send chan []byte     // outbound messages
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		c.Hub.Broadcast <- msg
	}
}

// WritePump sends outbound messages to the client
func (c *Client) WritePump() {
	defer c.Conn.Close()

	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}
