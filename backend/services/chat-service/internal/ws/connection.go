package ws

import (
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Connection struct {
	WS     *websocket.Conn
	Send   chan interface{}
	ChatID string
	Hub    *Hub
	UserID string
}

// ReadPump reads messages from WebSocket
func (c *Connection) ReadPump() {
	defer func() {
		c.Hub.Unregister(c.ChatID, c)
		_ = c.WS.Close()
	}()

	c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.WS.SetPongHandler(func(string) error {
		_ = c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg map[string]interface{}
		if err := c.WS.ReadJSON(&msg); err != nil {
			log.Println("ws read error:", err)
			break
		}

		if ev, ok := msg["event"].(string); ok {
			switch ev {
			case "typing":
				c.Hub.Broadcast(c.ChatID, map[string]interface{}{"event": "typing", "user": c.UserID})
			case "read":
				c.Hub.Broadcast(c.ChatID, map[string]interface{}{"event": "read", "user": c.UserID, "msg_id": msg["msg_id"]})
			case "reaction":
				c.Hub.Broadcast(c.ChatID, map[string]interface{}{"event": "reaction", "user": c.UserID, "msg_id": msg["msg_id"], "emoji": msg["emoji"]})
			}
		}
	}
}

// WritePump writes messages to WebSocket
func (c *Connection) WritePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.WS.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.WS.WriteJSON(msg); err != nil {
				log.Println("ws write error:", err)
				return
			}
		case <-ticker.C:
			_ = c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
