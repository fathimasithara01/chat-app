package ws

import (
	"encoding/json"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Connection struct {
	ws   *websocket.Conn
	send chan interface{}
	chat string
	uid  string
	hub  *Hub
}

func (c *Connection) readPump() {
	defer func() {
		c.hub.Unregister(c.chat, c)
		_ = c.ws.Close()
	}()
	c.ws.SetReadLimit(1024 * 16)
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		// Expect small JSON events like {"type":"typing","state":true}
		var ev map[string]interface{}
		if err := json.Unmarshal(data, &ev); err != nil {
			continue
		}
		// Broadcast to others in same chat
		c.hub.Broadcast(c.chat, map[string]interface{}{
			"from":  c.uid,
			"event": ev,
		})
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// channel closed
				_ = c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			b, _ := json.Marshal(msg)
			if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-ticker.C:
			// ping
			if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
