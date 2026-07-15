package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"
	"university-pass/internal/middleware"
	"university-pass/internal/model"
	"university-pass/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	guestRepo *repository.GuestRepository
}

func NewAuthService(userRepo *repository.UserRepository, guestRepo *repository.GuestRepository) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		guestRepo: guestRepo,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password, deviceID string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return "", fmt.Errorf("user is not active")
	}

	hash, err := s.userRepo.GetPasswordHashByUserID(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get password hash: %w", err)
	}
	if hash == "" {
		return "", fmt.Errorf("no password set for user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	secretKey, err := generateTOTPSecret()
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}

	if err := s.userRepo.UpsertDeviceSecret(ctx, user.ID, deviceID, secretKey); err != nil {
		return "", fmt.Errorf("failed to save device secret: %w", err)
	}

	return secretKey, nil
}

func generateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	_, err := rand.Read(secret)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

type VerifyUserResult struct {
	IsAllowed bool        `json:"is_allowed"`
	Direction string      `json:"direction,omitempty"`
	Reason    string      `json:"reason"`
	User      *model.User `json:"user,omitempty"`
}

func (s *AuthService) VerifyUser(ctx context.Context, deviceID, otpCode, scannerID string, accessPointID int) (*VerifyUserResult, error) {
	logDenied := func(userID *int, direction, reason string) {
		evt := model.AccessLogEvent{
			Type:          "scan",
			UserID:        userID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     false,
			Reason:        reason,
			ScannerID:     scannerID,
			LoggedAt:      time.Now().UTC(),
		}
		_ = s.userRepo.EnqueueAccessLog(ctx, evt)
	}

	device, err := s.userRepo.GetDeviceByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		logDenied(nil, "unknown", "device not found")
		return &VerifyUserResult{IsAllowed: false, Reason: "access denied"}, nil
	}

	ok, _ := totp.ValidateCustom(otpCode, device.SecretKey, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      2,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if !ok {
		logDenied(&device.UserID, "unknown", "invalid otp")
		return &VerifyUserResult{IsAllowed: false, Reason: "access denied"}, nil
	}

	step := time.Now().UTC().Unix() / 30
	updated, err := s.userRepo.UpdateLastUsedStepIfGreater(ctx, device.UserID, step)
	if err != nil {
		return nil, fmt.Errorf("failed to update last used step: %w", err)
	}
	if !updated {
		logDenied(&device.UserID, "unknown", "replay detected")
		return &VerifyUserResult{IsAllowed: false, Reason: "access denied"}, nil
	}

	user, err := s.userRepo.GetByUserID(ctx, device.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || !user.IsActive {
		logDenied(&device.UserID, "unknown", "user not found or inactive")
		return &VerifyUserResult{IsAllowed: false, Reason: "access denied"}, nil
	}

	isInside, err := s.userRepo.ToggleInside(ctx, device.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle inside state: %w", err)
	}
	direction := "exit"
	if isInside {
		direction = "enter"
	}

	successEvt := model.AccessLogEvent{
		Type:          "scan",
		UserID:        &device.UserID,
		AccessPointID: accessPointID,
		Direction:     direction,
		IsAllowed:     true,
		ScannerID:     scannerID,
		LoggedAt:      time.Now().UTC(),
	}
	_ = s.userRepo.EnqueueAccessLog(ctx, successEvt)

	return &VerifyUserResult{
		IsAllowed: true,
		Direction: direction,
		User:      user,
	}, nil
}

type VerifyGuestResult struct {
	Direction string           `json:"direction,omitempty"`
	IsAllowed bool             `json:"is_allowed"`
	Reason    string           `json:"reason"`
	Guest     *model.GuestPass `json:"guest,omitempty"`
}

func (s *AuthService) logGuestEvent(ctx context.Context, guestID, scannerID string, accessPointID int, direction string, allowed bool, reason string) {
	evt := model.AccessLogEvent{
		Type:          "scan",
		GuestPassID:   &guestID,
		AccessPointID: accessPointID,
		Direction:     direction,
		IsAllowed:     allowed,
		Reason:        reason,
		ScannerID:     scannerID,
		LoggedAt:      time.Now().UTC(),
	}
	_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
}

func (s *AuthService) VerifyGuest(ctx context.Context, guestID, scannerID string, accessPointID int) (*VerifyGuestResult, error) {
	guest, err := s.guestRepo.GetGuestPassByID(ctx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guest pass: %w", err)
	}
	if guest == nil {
		s.logGuestEvent(ctx, guestID, scannerID, accessPointID, "unknown", false, "guest pass not found")
		return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass not found"}, nil
	}

	now := time.Now().UTC()
	direction := "enter"
	if guest.IsEntered {
		direction = "exit"
	}

	if direction == "exit" {
		updated, err := s.guestRepo.MarkGuestPassExited(ctx, guestID)
		if err != nil {
			return nil, fmt.Errorf("failed to mark guest pass exited: %w", err)
		}
		if !updated {
			s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, false, "guest exit not allowed")
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest exit not allowed"}, nil
		}
		s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, true, "")
		guest.IsEntered = false
		return &VerifyGuestResult{IsAllowed: true, Direction: direction, Guest: guest}, nil
	}

	if now.Before(guest.ValidFrom) {
		s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, false, "guest pass not active yet")
		return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass not active yet"}, nil
	}
	if now.After(guest.ValidTo) {
		s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, false, "guest pass expired")
		return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass expired"}, nil
	}
	if guest.IsUsed {
		s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, false, "guest pass already used")
		return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass already used"}, nil
	}

	updated, err := s.guestRepo.MarkGuestPassEnteredIfValid(ctx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark guest pass entered: %w", err)
	}
	if !updated {
		s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, false, "guest pass already used or invalid")
		return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass already used or invalid"}, nil
	}

	s.logGuestEvent(ctx, guestID, scannerID, accessPointID, direction, true, "")
	guest.IsUsed = true
	guest.IsEntered = true
	return &VerifyGuestResult{IsAllowed: true, Direction: direction, Guest: guest}, nil
}

func (s *AuthService) AdminLogin(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.Role != "admin" {
		return "", fmt.Errorf("invalid credentials")
	}
	if !user.IsActive {
		return "", fmt.Errorf("user is not active")
	}

	hash, err := s.userRepo.GetPasswordHashByUserID(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get password hash: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	claims := middleware.Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("vBS0K4W5DRo2iTQI1JmnuqIouvnHaBbsyvXxqk1Ibhz")) // TODO: move to env
}
