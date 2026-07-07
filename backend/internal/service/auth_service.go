package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"
	"university-pass/internal/model"
	"university-pass/internal/repository"

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
	Reason    string      `json:"reason"`
	User      *model.User `json:"user,omitempty"`
}

func (s *AuthService) VerifyUser(ctx context.Context, userID int, otpCode, scannerID, direction string, accessPointID int) (*VerifyUserResult, error) {
	device, err := s.userRepo.GetDeviceByUserID(ctx, userID)
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
		_ = s.userRepo.EnqueueAccessLog(ctx, evt)
		return &VerifyUserResult{IsAllowed: false, Reason: "device not found"}, nil
	}

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
		_ = s.userRepo.EnqueueAccessLog(ctx, evt)
		return &VerifyUserResult{IsAllowed: false, Reason: "invalid otp"}, nil
	}

	step := time.Now().UTC().Unix() / 30
	updated, err := s.userRepo.UpdateLastUsedStepIfGreater(ctx, userID, step)
	if err != nil {
		return nil, fmt.Errorf("failed to update last used step: %w", err)
	}
	if !updated {
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
		_ = s.userRepo.EnqueueAccessLog(ctx, evt)
		return &VerifyUserResult{IsAllowed: false, Reason: "replay detected"}, nil
	}

	user, err := s.userRepo.GetByUserID(ctx, userID)
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
	_ = s.userRepo.EnqueueAccessLog(ctx, evt)

	return &VerifyUserResult{
		IsAllowed: true,
		Reason:    "",
		User:      user,
	}, nil
}

type VerifyGuestResult struct {
	IsAllowed bool             `json:"is_allowed"`
	Reason    string           `json:"reason"`
	Guest     *model.GuestPass `json:"guest,omitempty"`
}

func (s *AuthService) VerifyGuest(ctx context.Context, guestID, scannerID, direction string, accessPointID int) (*VerifyGuestResult, error) {
	guest, err := s.guestRepo.GetGuestPassByID(ctx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guest pass: %w", err)
	}
	if guest == nil {
		evt := model.AccessLogEvent{
			Type:          "scan",
			GuestPassID:   &guestID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     false,
			Reason:        "guest pass not found",
			ScannerID:     scannerID,
			LoggedAt:      time.Now().UTC(),
		}
		_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
		return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass not found"}, nil
	}

	now := time.Now().UTC()

	switch direction {
	case "enter":
		if now.Before(guest.ValidFrom) {
			evt := model.AccessLogEvent{
				Type:          "scan",
				GuestPassID:   &guestID,
				AccessPointID: accessPointID,
				Direction:     direction,
				IsAllowed:     false,
				Reason:        "guest pass not active yet",
				ScannerID:     scannerID,
				LoggedAt:      now,
			}
			_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass not active yet"}, nil
		}

		if now.After(guest.ValidTo) {
			evt := model.AccessLogEvent{
				Type:          "scan",
				GuestPassID:   &guestID,
				AccessPointID: accessPointID,
				Direction:     direction,
				IsAllowed:     false,
				Reason:        "guest pass expired",
				ScannerID:     scannerID,
				LoggedAt:      now,
			}
			_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass expired"}, nil
		}

		if guest.IsUsed || guest.IsEntered {
			evt := model.AccessLogEvent{
				Type:          "scan",
				GuestPassID:   &guestID,
				AccessPointID: accessPointID,
				Direction:     direction,
				IsAllowed:     false,
				Reason:        "guest pass already used",
				ScannerID:     scannerID,
				LoggedAt:      now,
			}
			_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass already used"}, nil
		}

		updated, err := s.guestRepo.MarkGuestPassEnteredIfValid(ctx, guestID)
		if err != nil {
			return nil, fmt.Errorf("failed to mark guest pass entered: %w", err)
		}
		if !updated {
			evt := model.AccessLogEvent{
				Type:          "scan",
				GuestPassID:   &guestID,
				AccessPointID: accessPointID,
				Direction:     direction,
				IsAllowed:     false,
				Reason:        "guest pass already used or invalid",
				ScannerID:     scannerID,
				LoggedAt:      now,
			}
			_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest pass already used or invalid"}, nil
		}

		evt := model.AccessLogEvent{
			Type:          "scan",
			GuestPassID:   &guestID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     true,
			Reason:        "",
			ScannerID:     scannerID,
			LoggedAt:      now,
		}
		_ = s.guestRepo.EnqueueAccessLog(ctx, evt)

		guest.IsUsed = true
		guest.IsEntered = true

		return &VerifyGuestResult{
			IsAllowed: true,
			Reason:    "",
			Guest:     guest,
		}, nil

	case "exit":
		if !guest.IsEntered {
			evt := model.AccessLogEvent{
				Type:          "scan",
				GuestPassID:   &guestID,
				AccessPointID: accessPointID,
				Direction:     direction,
				IsAllowed:     false,
				Reason:        "guest is not inside",
				ScannerID:     scannerID,
				LoggedAt:      now,
			}
			_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest is not inside"}, nil
		}

		updated, err := s.guestRepo.MarkGuestPassExited(ctx, guestID)
		if err != nil {
			return nil, fmt.Errorf("failed to mark guest pass exited: %w", err)
		}
		if !updated {
			evt := model.AccessLogEvent{
				Type:          "scan",
				GuestPassID:   &guestID,
				AccessPointID: accessPointID,
				Direction:     direction,
				IsAllowed:     false,
				Reason:        "guest exit not allowed",
				ScannerID:     scannerID,
				LoggedAt:      now,
			}
			_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
			return &VerifyGuestResult{IsAllowed: false, Reason: "guest exit not allowed"}, nil
		}

		evt := model.AccessLogEvent{
			Type:          "scan",
			GuestPassID:   &guestID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     true,
			Reason:        "",
			ScannerID:     scannerID,
			LoggedAt:      now,
		}
		_ = s.guestRepo.EnqueueAccessLog(ctx, evt)

		guest.IsEntered = false

		return &VerifyGuestResult{
			IsAllowed: true,
			Reason:    "",
			Guest:     guest,
		}, nil

	default:
		evt := model.AccessLogEvent{
			Type:          "scan",
			GuestPassID:   &guestID,
			AccessPointID: accessPointID,
			Direction:     direction,
			IsAllowed:     false,
			Reason:        "invalid direction",
			ScannerID:     scannerID,
			LoggedAt:      now,
		}
		_ = s.guestRepo.EnqueueAccessLog(ctx, evt)
		return &VerifyGuestResult{IsAllowed: false, Reason: "invalid direction"}, nil
	}
}
