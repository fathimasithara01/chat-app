package ws

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type Client struct {
	ws      *websocket.Conn
	send    chan []byte
	chat    string 
	uid     string 
	hub     *Hub
	limiter *rate.Limiter
	pending map[string]*Envelope
	closed  int32
}

func NewClient(conn *websocket.Conn, uid, chatID string, hub *Hub, rps int) *Client {
	return &Client{
		ws:      conn,
		send:    make(chan []byte, 256),
		chat:    chatID,
		uid:     uid,
		hub:     hub,
		limiter: rate.NewLimiter(rate.Limit(rps), rps),
		pending: make(map[string]*Envelope),
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c.chat, c)
		_ = c.ws.Close()
	}()
	c.ws.SetReadLimit(64 * 1024)
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		if !c.limiter.Allow() {
			continue
		}
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}

		_ = c.hub.rdb.Set(context.Background(), "presence:"+c.uid, "online", 60*time.Second).Err()

		switch env.Type {
		case "message":
			if env.MsgID == "" {
				env.MsgID = uuid.NewString()
			}
			env.From = c.uid
			c.pending[env.MsgID] = &env
			_ = c.hub.Broadcast(c.chat, &env)
			go c.waitAck(env.MsgID, 5*time.Second)
		case "ack":
			if env.MsgID != "" {
				delete(c.pending, env.MsgID)
			}
		case "typing_start", "typing_stop":
			_ = c.hub.Broadcast(c.chat, &env)
		default:
		}
	}
}

func (c *Client) waitAck(msgID string, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-t.C:
		if _, ok := c.pending[msgID]; ok {
			env := c.pending[msgID]
			_ = c.hub.publishRedis("ws:global", mustJSON(env))
		}
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Client) close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		close(c.send)
		_ = c.ws.Close()
	}
}
