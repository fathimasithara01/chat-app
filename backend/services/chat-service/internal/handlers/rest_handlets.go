package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yourorg/chat-app/services/chat-service/internal/service"
)

type RestHandler struct {
	chatSvc *service.ChatService
}

func NewRestHandler(cs *service.ChatService) *RestHandler {
	return &RestHandler{chatSvc: cs}
}

// GET /conversations/{convId}/messages?limit=50&before=2025-01-02T15:04:05Z
func (h *RestHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	convId := chi.URLParam(r, "convId")
	limitStr := r.URL.Query().Get("limit")
	limit := int64(50)
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = int64(v)
		}
	}
	beforeStr := r.URL.Query().Get("before")
	var before time.Time
	if beforeStr != "" {
		if t, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			before = t
		}
	}
	msgs, err := h.chatSvc.GetHistory(r.Context(), convId, limit, before)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(msgs)
}
