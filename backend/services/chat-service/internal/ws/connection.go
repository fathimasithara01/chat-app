package ws

import (
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Connection struct {
	ws     *websocket.Conn
	send   chan interface{}
	chatID string
	hub    *Hub
	userID string
}

func (c *Connection) readPump() {
	defer func() {
		c.hub.Unregister(c.chatID, c)
		_ = c.ws.Close()
	}()

	c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.ws.SetPongHandler(func(string) error {
		_ = c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg map[string]interface{}
		if err := c.ws.ReadJSON(&msg); err != nil {
			log.Println("ws read:", err)
			break
		}
		// client events e.g. typing, reaction forwarded by server to room
		if ev, ok := msg["event"].(string); ok {
			switch ev {
			case "typing":
				c.hub.Broadcast(c.chatID, map[string]interface{}{"event": "typing", "user": c.userID})
			case "read":
				c.hub.Broadcast(c.chatID, map[string]interface{}{"event": "read", "user": c.userID, "msg_id": msg["msg_id"]})
			case "reaction":
				c.hub.Broadcast(c.chatID, map[string]interface{}{"event": "reaction", "user": c.userID, "msg_id": msg["msg_id"], "emoji": msg["emoji"]})
			}
		}
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteJSON(msg); err != nil {
				log.Println("ws write:", err)
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}	
		}
	}
}
