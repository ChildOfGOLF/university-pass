package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"university-pass/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

type LoginResponse struct {
	SecretKey string `json:"secret_key"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.DeviceID == "" {
		sendError(w, http.StatusBadRequest, "email, password and device_id are required")
		return
	}

	secretKey, err := h.authService.Login(r.Context(), req.Email, req.Password, req.DeviceID)
	if err != nil {
		// 401 остальные 500
		msg := err.Error()
		if msg == "invalid credentials" || msg == "user not found" || msg == "no password set for user" {
			sendError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, LoginResponse{SecretKey: secretKey})
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("failed to encode response: %v\n", err)
	}
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil {
		fmt.Printf("failed to encode error response: %v\n", err)
	}
}
