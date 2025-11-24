package ws

import (
	"encoding/json"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Connection struct {
	ws   *websocket.Conn
	send chan interface{}
	room string
	uid  string
	hub  *Hub
}

func NewConnection(conn *websocket.Conn, uid, room string, hub *Hub) *Connection {
	return &Connection{
		ws:   conn,
		send: make(chan interface{}, 256),
		room: room,
		uid:  uid,
		hub:  hub,
	}
}

func (c *Connection) readPump() {
	defer func() {
		c.hub.Unregister(c.room, c)
		_ = c.ws.Close()
	}()
	c.ws.SetReadLimit(1024 * 64)
	c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.ws.SetPongHandler(func(string) error {
		_ = c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}

		msg := map[string]interface{}{
			"type":      payload["type"],
			"from":      c.uid,
			"room":      c.room,
			"payload":   payload,
			"timestamp": time.Now().Unix(),
		}
		c.hub.Broadcast(c.room, msg)
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
				_ = c.ws.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second))
				return
			}
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			w, err := c.ws.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			b, _ := json.Marshal(msg)
			if _, err := w.Write(b); err != nil {
				_ = w.Close()
				return
			}
			_ = w.Close()
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.ws.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second)); err != nil {
				return
			}
		}
	}
}
