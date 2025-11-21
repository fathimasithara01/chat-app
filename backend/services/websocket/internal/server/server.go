package server

import (
	"net/http"

	"github.com/fathima-sithara/websocket/internal/auth"
	"github.com/fathima-sithara/websocket/internal/handlers"
	"github.com/fathima-sithara/websocket/internal/ws"
	"github.com/go-chi/chi/v5"
)

func Start(hub *ws.Hub, validator *auth.JWTValidator, port string) {
	r := chi.NewRouter()

	wsHandler := handlers.NewWSHandler(hub)

	r.With(auth.JWTMiddleware(validator)).
		Get("/ws", wsHandler.ServeWS)

	http.ListenAndServe(":"+port, r)
}
