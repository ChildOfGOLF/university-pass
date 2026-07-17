package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
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
	Email    string `json:"email" example:"admin@uni.com"`
	Password string `json:"password" example:"password123"`
}

// AdminLogin godoc
// @Summary Логин админа
// @Tags admin-auth
// @Accept json
// @Produce json
// @Param request body AdminLoginRequest true "Учетные данные администратора"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/auth/login [post]
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

// CreateUser godoc
// @Summary Создать пользователя
// @Tags admin-users
// @Accept json
// @Produce json
// @Security AdminBearer
// @Param request body model.CreateUserRequest true "Данные пользователя"
// @Success 201 {object} model.User
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/users [post]
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

// ListUsers godoc
// @Summary Список пользователей
// @Tags admin-users
// @Produce json
// @Security AdminBearer
// @Success 200 {array} model.User
// @Failure 401 {object} ErrorResponse
// @Router /admin/users [get]
func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListUsers(r.Context())
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, users)
}

// ListGroups godoc
// @Summary Список групп
// @Tags admin-users
// @Produce json
// @Security AdminBearer
// @Success 200 {array} model.GroupResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/groups [get]
func (h *AdminUserHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.userRepo.ListGroups(r.Context())
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, groups)
}

// UpdateUser godoc
// @Summary Обновить данные пользователя
// @Tags admin-users
// @Accept json
// @Produce json
// @Security AdminBearer
// @Param id path int true "ID пользователя"
// @Param request body model.UpdateUserRequest true "Поля для обновления"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/users/{id} [patch]
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

// DeactivateUser godoc
// @Summary Деактивировать пользователя
// @Tags admin-users
// @Produce json
// @Security AdminBearer
// @Param id path int true "ID пользователя"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/users/{id} [delete]
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

// CreateGuestPass godoc
// @Summary Создать гостевой пропуск
// @Tags admin-guests
// @Accept json
// @Produce json
// @Security AdminBearer
// @Param request body model.CreateGuestPassRequest true "Данные гостя"
// @Success 201 {object} model.GuestPass
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/guests [post]
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

// ListGuestPasses godoc
// @Summary Список гостевых пропусков
// @Tags admin-guests
// @Produce json
// @Security AdminBearer
// @Success 200 {array} model.GuestPass
// @Failure 401 {object} ErrorResponse
// @Router /admin/guests [get]
func (h *AdminGuestHandler) List(w http.ResponseWriter, r *http.Request) {
	guests, err := h.guestRepo.ListGuestPasses(r.Context())
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, guests)
}

// RevokeGuestPass godoc
// @Summary Отозвать гостевой пропуск
// @Tags admin-guests
// @Produce json
// @Security AdminBearer
// @Param id path string true "UUID гостевого пропуска"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/guests/{id}/revoke [post]
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

type AdminLogHandler struct {
	logRepo *repository.LogRepository
}

func NewAdminLogHandler(logRepo *repository.LogRepository) *AdminLogHandler {
	return &AdminLogHandler{logRepo: logRepo}
}

// List godoc
// @Summary Список логов
// @Description Фильтры опциональны. from/to например 2026-07-01T00:00:00Z. direction: enter, exit, unknown.
// @Tags admin-logs
// @Produce json
// @Security AdminBearer
// @Param user_id query int false "ID пользователя"
// @Param guest_pass_id query string false "UUID гостевого пропуска"
// @Param access_point_id query int false "ID точки доступа"
// @Param direction query string false "enter, exit или unknown"
// @Param is_allowed query bool false "true/false"
// @Param from query string false "format"
// @Param to query string false "format"
// @Param limit query int false "по умолчанию 100, максимум 500"
// @Param offset query int false "по умолчанию 0"
// @Success 200 {array} model.AccessLogResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/logs [get]
func (h *AdminLogHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := repository.ListAccessLogsFilter{
		Direction: q.Get("direction"),
	}

	if v := q.Get("user_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			sendError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		filter.UserID = &id
	}

	if v := q.Get("guest_pass_id"); v != "" {
		filter.GuestPassID = &v
	}

	if v := q.Get("access_point_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			sendError(w, http.StatusBadRequest, "invalid access_point_id")
			return
		}
		filter.AccessPointID = &id
	}

	if v := q.Get("is_allowed"); v != "" {
		allowed, err := strconv.ParseBool(v)
		if err != nil {
			sendError(w, http.StatusBadRequest, "invalid is_allowed")
			return
		}
		filter.IsAllowed = &allowed
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			sendError(w, http.StatusBadRequest, "invalid")
			return
		}
		filter.From = &t
	}

	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			sendError(w, http.StatusBadRequest, "invalid")
			return
		}
		filter.To = &t
	}

	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			sendError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		filter.Limit = n
	}

	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			sendError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		filter.Offset = n
	}

	logs, err := h.logRepo.ListAccessLogs(r.Context(), filter)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, logs)
}
