package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

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
	IsAllowed bool
	Reason    string
}

func (s *AuthService) VerifyUser(ctx context.Context, userID int, otpCode string) (*VerifyUserResult, error) {
	device, err := s.repo.GetDeviceByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return &VerifyUserResult{
			IsAllowed: false,
			Reason:    "device not found",
		}, nil
	}

	ok, _ := totp.ValidateCustom(otpCode, device.SecretKey, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if !ok {
		return &VerifyUserResult{
			IsAllowed: false,
			Reason:    "invalid otp",
		}, nil
	}

	return &VerifyUserResult{
		IsAllowed: true,
		Reason:    "",
	}, nil
}
