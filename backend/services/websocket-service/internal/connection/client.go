package connection

import (
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Client struct {
	Conn      *websocket.Conn
	UserUUID  string
	ConvID    string
	Send      chan []byte
	Connected time.Time
	mu        sync.Mutex
	closed    bool
}

func NewClient(conn *websocket.Conn, userUUID, convID string) *Client {
	return &Client{
		Conn:      conn,
		UserUUID:  userUUID,
		ConvID:    convID,
		Send:      make(chan []byte, 256),
		Connected: time.Now().UTC(),
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	if !c.closed {
		close(c.Send)
		_ = c.Conn.Close()
		c.closed = true
	}
	c.mu.Unlock()
}
