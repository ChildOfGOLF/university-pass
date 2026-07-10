package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"university-pass/internal/model"
	"university-pass/internal/repository"
	"university-pass/internal/service"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminAuthHandler struct {
	authService *service.AuthService
}

func NewAdminAuthHandler(authService *service.AuthService) *AdminAuthHandler {
	return &AdminAuthHandler{authService: authService}
}

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		sendError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	token, err := h.authService.AdminLogin(r.Context(), req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			sendError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"token": token})
}

type AdminUserHandler struct {
	userRepo *repository.UserRepository
}

func NewAdminUserHandler(userRepo *repository.UserRepository) *AdminUserHandler {
	return &AdminUserHandler{userRepo: userRepo}
}

func (h *AdminUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.LastName == "" || req.FirstName == "" || req.Role == "" || req.Password == "" {
		sendError(w, http.StatusBadRequest, "email, last_name, first_name, role and password are required")
		return
	}

	roleID, err := h.userRepo.GetRoleIDByName(r.Context(), req.Role)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.userRepo.CreateUser(r.Context(), repository.CreateUserParams{
		Email: req.Email, LastName: req.LastName, FirstName: req.FirstName,
		Patronymic: req.Patronymic, Phone: req.Phone,
		RoleID: roleID, RoleName: req.Role, GroupID: req.GroupID,
		PasswordHash: string(hash),
	})
	if err != nil {
		// TODO: проверить pgconn.PgError с кодом 23505
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusCreated, user)
}

func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListUsers(r.Context())
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, users)
}

func (h *AdminUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.userRepo.UpdateUser(r.Context(), id, req); err != nil {
		if err.Error() == "user not found" {
			sendError(w, http.StatusNotFound, "user not found")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminUserHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "invalid id")
		return
	}

	inactive := false
	if err := h.userRepo.UpdateUser(r.Context(), id, model.UpdateUserRequest{IsActive: &inactive}); err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

type AdminGuestHandler struct {
	guestRepo *repository.GuestRepository
}

func NewAdminGuestHandler(guestRepo *repository.GuestRepository) *AdminGuestHandler {
	return &AdminGuestHandler{guestRepo: guestRepo}
}

func (h *AdminGuestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateGuestPassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.LastName == "" || req.FirstName == "" || req.ValidFrom.IsZero() || req.ValidTo.IsZero() {
		sendError(w, http.StatusBadRequest, "last_name, first_name, valid_from and valid_to are required")
		return
	}

	if !req.ValidTo.After(req.ValidFrom) {
		sendError(w, http.StatusBadRequest, "valid_to must be after valid_from")
		return
	}

	guest := &model.GuestPass{
		LastName: req.LastName, FirstName: req.FirstName, Patronymic: req.Patronymic,
		Purpose: req.Purpose, ValidFrom: req.ValidFrom, ValidTo: req.ValidTo,
	}

	created, err := h.guestRepo.CreateGuestPass(r.Context(), guest)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusCreated, created)
}

func (h *AdminGuestHandler) List(w http.ResponseWriter, r *http.Request) {
	guests, err := h.guestRepo.ListGuestPasses(r.Context())
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, guests)
}

func (h *AdminGuestHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ok, err := h.guestRepo.RevokeGuestPass(r.Context(), id)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !ok {
		sendError(w, http.StatusNotFound, "guest pass already used, entered or not found")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
