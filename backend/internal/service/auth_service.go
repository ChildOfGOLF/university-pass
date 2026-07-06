package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
	"university-pass/internal/model"

	"university-pass/internal/repository"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Login(ctx context.Context, email, password, deviceID string) (string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return "", fmt.Errorf("user is not active")
	}

	hash, err := s.repo.GetPasswordHashByUserID(ctx, user.ID)
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

	if err := s.repo.UpsertDeviceSecret(ctx, user.ID, deviceID, secretKey); err != nil {
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
	Reason    string      `json:"reason"`
	User      *model.User `json:"user,omitempty"`
}

func (s *AuthService) VerifyUser(ctx context.Context, userID int, otpCode, scannerID, direction string, accessPointID int) (*VerifyUserResult, error) {
	device, err := s.repo.GetDeviceByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		evt := model.AccessLogEvent{
			Type:          "scan",
			UserID:        &userID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     false,
			Reason:        "device not found",
			ScannerID:     scannerID,
			LoggedAt:      time.Now().UTC(),
		}
		_ = s.repo.EnqueueAccessLog(ctx, evt)
		return &VerifyUserResult{IsAllowed: false, Reason: "device not found"}, nil
	}

	// --- DEBUG START ---
	fmt.Printf("DEBUG request otp: %q\n", otpCode)
	fmt.Printf("DEBUG device.SecretKey raw: %q (len=%d)\n", device.SecretKey, len(device.SecretKey))

	// trim spaces and uppercase for base32 canonicalization
	trimmed := strings.TrimSpace(device.SecretKey)
	upper := strings.ToUpper(trimmed)
	decoded, decErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(upper)

	fmt.Printf("DEBUG base32 decode err: %v decoded_len=%d\n", decErr, len(decoded))
	if decErr == nil {
		norm := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(decoded)
		fmt.Printf("DEBUG normalized secret (re-encoded): %q (len=%d)\n", norm, len(norm))
	} else {
		fmt.Printf("DEBUG cannot normalize secret\n")
	}

	// server-generated codes for window [-1, 0, +1]
	now := time.Now().UTC()
	for i := -1; i <= 1; i++ {
		t := now.Add(time.Duration(i*30) * time.Second)
		code, genErr := totp.GenerateCode(device.SecretKey, t)
		step := t.Unix() / 30
		fmt.Printf("DEBUG server code offset=%+d code=%q genErr=%v time=%s step=%d\n", i, code, genErr, t.Format(time.RFC3339), step)
	}
	fmt.Printf("DEBUG current server step: %d\n", now.Unix()/30)
	// --- DEBUG END ---

	ok, _ := totp.ValidateCustom(otpCode, device.SecretKey, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      2,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if !ok {
		evt := model.AccessLogEvent{
			Type:          "scan",
			UserID:        &userID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     false,
			Reason:        "invalid otp",
			ScannerID:     scannerID,
			LoggedAt:      time.Now().UTC(),
		}
		_ = s.repo.EnqueueAccessLog(ctx, evt)
		return &VerifyUserResult{IsAllowed: false, Reason: "invalid otp"}, nil
	}

	step := time.Now().UTC().Unix() / 30
	updated, err := s.repo.UpdateLastUsedStepIfGreater(ctx, userID, step)
	if err != nil {
		return nil, fmt.Errorf("failed to update last used step: %w", err)
	}
	if !updated {
		// replay или уже использован
		evt := model.AccessLogEvent{
			Type:          "scan",
			UserID:        &userID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     false,
			Reason:        "replay detected",
			ScannerID:     scannerID,
			LoggedAt:      time.Now().UTC(),
		}
		_ = s.repo.EnqueueAccessLog(ctx, evt)
		return &VerifyUserResult{IsAllowed: false, Reason: "replay detected"}, nil
	}

	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	evt := model.AccessLogEvent{
		Type:          "scan",
		UserID:        &userID,
		AccessPointID: accessPointID,
		Direction:     direction,
		IsAllowed:     true,
		Reason:        "",
		ScannerID:     scannerID,
		LoggedAt:      time.Now().UTC(),
	}
	_ = s.repo.EnqueueAccessLog(ctx, evt)

	return &VerifyUserResult{
		IsAllowed: true,
		Reason:    "",
		User:      user,
	}, nil
}
