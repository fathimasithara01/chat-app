package handler

import (
	"encoding/json"
	"net/http"
	"user-service/internal/domain"
	"user-service/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{s}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	json.NewDecoder(r.Body).Decode(&req)

	resp, err := h.service.CreateUser(req)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	json.NewEncoder(w).Encode(resp)
}
