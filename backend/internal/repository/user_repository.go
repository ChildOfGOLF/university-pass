package repository

import (
	"context"
	"fmt"
	"university-pass/internal/database"
	"university-pass/internal/model"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT u.id, u.email, u.first_name, u.last_name, u.patronymic,
       COALESCE(u.avatar_url, '') AS avatar_url,
       r.name AS role, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1`

	var u model.User

	err := r.db.Pg.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Patronymic, &u.AvatarURL, &u.Role, &u.IsActive, &u.CreatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetDeviceByUserID(ctx context.Context, userID int) (*model.UserDevice, error) {
	query := `
		SELECT user_id, device_id, secret_key, last_used_step, created_at, updated_at
		FROM user_devices
		WHERE user_id = $1
	`

	var d model.UserDevice

	err := r.db.Pg.QueryRow(ctx, query, userID).Scan(
		&d.UserID,
		&d.DeviceID,
		&d.SecretKey,
		&d.LastUsedStep,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *UserRepository) UpsertDeviceSecret(ctx context.Context, userID int, deviceID string, secretKey string) error {
	query := `
	INSERT INTO user_devices (user_id, device_id, secret_key, updated_at)
	VALUES ($1, $2, $3, NOW())
	ON CONFLICT (user_id) DO UPDATE
	SET device_id = $2, secret_key = $3, updated_at = NOW()
	`

	_, err := r.db.Pg.Exec(ctx, query, userID, deviceID, secretKey)
	if err != nil {
		return fmt.Errorf("failed to upsert device secret: %w", err)
	}

	return nil
}

func (r *UserRepository) GetPasswordHashByUserID(ctx context.Context, userID int) (string, error) {
	query := `SELECT password_hash FROM passwords WHERE password_id = $1`
	var hash string
	err := r.db.Pg.QueryRow(ctx, query, userID).Scan(&hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return hash, nil
}

func (r *UserRepository) UpdateLastUsedStepIfGreater(ctx context.Context, userID int, step int64) (bool, error) {
	query := `
		UPDATE user_devices
		SET last_used_step = $1, updated_at = NOW()
		WHERE user_id = $2
		  AND (last_used_step IS NULL OR last_used_step < $1)
`

	cmd, err := r.db.Pg.Exec(ctx, query, step, userID)
	if err != nil {
		return false, err
	}

	return cmd.RowsAffected() == 1, nil
}
