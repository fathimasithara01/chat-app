package websocket

import (
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Client struct {
	Conn        *websocket.Conn
	UserUUID    string
	ConvID      string
	Send        chan []byte
	ConnectedAt time.Time
	mu          sync.Mutex
}

func NewClient(conn *websocket.Conn, userUUID, convID string) *Client {
	return &Client{
		Conn:        conn,
		UserUUID:    userUUID,
		ConvID:      convID,
		Send:        make(chan []byte, 256),
		ConnectedAt: time.Now().UTC(),
	}
}
