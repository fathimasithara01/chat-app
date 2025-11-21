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

// Client represents a single websocket connection.
type Client struct {
	ws      *websocket.Conn
	send    chan []byte
	chat    string // chat room id
	uid     string // user id
	hub     *Hub
	limiter *rate.Limiter
	pending map[string]*Envelope
	closed  int32
}

// NewClient constructs a client. rps = requests per second allowed.
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

// readPump reads messages from ws connection, validates, and routes them.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c.chat, c)
		_ = c.ws.Close()
	}()

	c.ws.SetReadLimit(64 * 1024)
	// optional read deadline handling left out for brevity

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}

		// inbound rate limiting
		if !c.limiter.Allow() {
			// optionally notify client about rate limit
			continue
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			// malformed => ignore
			continue
		}

		// refresh presence TTL
		_ = c.hub.rdb.Set(context.Background(), "presence:"+c.uid, "online", 60*time.Second).Err()

		switch env.Type {
		case "message":
			// ensure msg id
			if env.MsgID == "" {
				env.MsgID = uuid.NewString()
			}
			env.From = c.uid
			// track pending ack
			c.pending[env.MsgID] = &env

			// local delivery + pubsub for other nodes
			_ = c.hub.Broadcast(c.chat, &env)

			// wait for ack and if not received, republish for retry
			go c.waitAck(env.MsgID, 5*time.Second)

		case "ack":
			if env.MsgID != "" {
				delete(c.pending, env.MsgID)
			}

		case "typing_start", "typing_stop":
			_ = c.hub.Broadcast(c.chat, &env)

		default:
			// unknown: ignore or log
		}
	}
}

func (c *Client) waitAck(msgID string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	<-timer.C
	if _, ok := c.pending[msgID]; ok {
		// publish to global channel for other nodes to retry
		env := c.pending[msgID]
		_ = c.hub.publishRedis("ws:global", mustJSON(env))
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// writePump writes messages from send channel to websocket.
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
			// ping to keep connection alive
			if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// close safely closes a client
func (c *Client) close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		close(c.send)
		_ = c.ws.Close()
	}
}
