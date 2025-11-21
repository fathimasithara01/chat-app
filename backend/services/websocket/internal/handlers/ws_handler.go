package handlers

import (
	"net/http"

	"github.com/fathima-sithara/websocket/internal/ws"
)

type WSHandler struct {
	Hub *ws.Hub
}

func NewWSHandler(h *ws.Hub) *WSHandler {
	return &WSHandler{Hub: h}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	conn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", 500)
		return
	}

	client := &ws.Client{
		ID:   userID,
		Conn: conn,
		Hub:  h.Hub,
		Send: make(chan []byte, 256),
	}

	h.Hub.Register <- client

	go client.ReadLoop()
	go client.WriteLoop()
}
