package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"university-pass/internal/model"
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

type VerifyUserRequest struct {
	UserID    int    `json:"user_id"`
	OTP       string `json:"otp"`
	ScannerID string `json:"scanner_id"`
	Direction string `json:"direction"`
}

type VerifyUserResponse struct {
	IsAllowed bool        `json:"is_allowed"`
	Reason    string      `json:"reason"`
	User      *model.User `json:"user,omitempty"` // TODO: подумать над *model
}

type VerifyGuestRequest struct {
	GuestID   string `json:"guest_id"`
	ScannerID string `json:"scanner_id"`
	Direction string `json:"direction"`
}

type VerifyGuestResponse struct {
	IsAllowed bool             `json:"is_allowed"`
	Reason    string           `json:"reason"`
	Guest     *model.GuestPass `json:"guest,omitempty"`
}

func (h *AuthHandler) VerifyUser(w http.ResponseWriter, r *http.Request) {
	var req VerifyUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID <= 0 || req.OTP == "" || req.ScannerID == "" || req.Direction == "" {
		sendError(w, http.StatusBadRequest, "user_id, otp, scanner_id and direction are required")
		return
	}

	// TODO: определить accessPointID по scannerID; пока что 0 или искать в бд
	accessPointID := 0

	result, err := h.authService.VerifyUser(r.Context(), req.UserID, req.OTP, req.ScannerID, req.Direction, accessPointID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, VerifyUserResponse{
		IsAllowed: result.IsAllowed,
		Reason:    result.Reason,
		User:      result.User,
	})
}

func (h *AuthHandler) VerifyGuest(w http.ResponseWriter, r *http.Request) {
	var req VerifyGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GuestID == "" || req.ScannerID == "" || req.Direction == "" {
		sendError(w, http.StatusBadRequest, "guest_id, scanner_id and direction are required")
		return
	}

	// TODO: определить accessPointID по scannerID; пока что 0 или искать в бд
	accessPointID := 0

	result, err := h.authService.VerifyGuest(r.Context(), req.GuestID, req.ScannerID, req.Direction, accessPointID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, VerifyGuestResponse{
		IsAllowed: result.IsAllowed,
		Reason:    result.Reason,
		Guest:     result.Guest,
	})
}
