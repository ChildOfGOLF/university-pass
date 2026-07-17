package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"university-pass/internal/database"
	"university-pass/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
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

func (r *UserRepository) GetByUserID(ctx context.Context, userID int) (*model.User, error) {
	query := `SELECT u.id, u.email, u.first_name, u.last_name, u.patronymic,
       COALESCE(u.avatar_url, '') AS avatar_url,
       r.name AS role, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1`

	var u model.User

	err := r.db.Pg.QueryRow(ctx, query, userID).Scan(
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
	key := fmt.Sprintf("user:device:%d", userID)

	val, err := r.db.Rdb.Get(ctx, key).Result()
	if err == nil {
		var d model.UserDevice
		if err := json.Unmarshal([]byte(val), &d); err == nil {
			return &d, nil
		}
	} else if err != redis.Nil {
		return nil, fmt.Errorf("failed to get device from redis: %w", err)
	}

	query := `
		SELECT user_id, device_id, secret_key, last_used_step, created_at, updated_at
		FROM user_devices
		WHERE user_id = $1
	`

	var d model.UserDevice
	err = r.db.Pg.QueryRow(ctx, query, userID).Scan(
		&d.UserID, &d.DeviceID, &d.SecretKey, &d.LastUsedStep, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if b, err := json.Marshal(d); err == nil {
		_ = r.db.Rdb.Set(ctx, key, b, 24*time.Hour).Err()
	}

	return &d, nil
}

// scan/verify
func (r *UserRepository) GetDeviceByDeviceID(ctx context.Context, deviceID string) (*model.UserDevice, error) {
	key := fmt.Sprintf("device:secret:%s", deviceID)

	val, err := r.db.Rdb.Get(ctx, key).Result()
	if err == nil {
		var d model.UserDevice
		if err := json.Unmarshal([]byte(val), &d); err == nil {
			return &d, nil
		}
	} else if err != redis.Nil {
		return nil, fmt.Errorf("failed to get device from redis: %w", err)
	}

	query := `
		SELECT user_id, device_id, secret_key, last_used_step, created_at, updated_at
		FROM user_devices
		WHERE device_id = $1
	`

	var d model.UserDevice
	err = r.db.Pg.QueryRow(ctx, query, deviceID).Scan(
		&d.UserID, &d.DeviceID, &d.SecretKey, &d.LastUsedStep, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if b, err := json.Marshal(d); err == nil {
		_ = r.db.Rdb.Set(ctx, key, b, 24*time.Hour).Err()
	}

	return &d, nil
}

func (r *UserRepository) UpsertDeviceSecret(ctx context.Context, userID int, deviceID string, secretKey string) error {
	var oldDeviceID string
	_ = r.db.Pg.QueryRow(ctx, `SELECT device_id FROM user_devices WHERE user_id = $1`, userID).Scan(&oldDeviceID)

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

	if oldDeviceID != "" && oldDeviceID != deviceID {
		_ = r.db.Rdb.Del(ctx, fmt.Sprintf("device:secret:%s", oldDeviceID)).Err()
	}

	d := model.UserDevice{
		UserID:       userID,
		DeviceID:     deviceID,
		SecretKey:    secretKey,
		LastUsedStep: nil,
	}
	if b, err := json.Marshal(d); err == nil {
		_ = r.db.Rdb.Set(ctx, fmt.Sprintf("user:device:%d", userID), b, 24*time.Hour).Err()
		_ = r.db.Rdb.Set(ctx, fmt.Sprintf("device:secret:%s", deviceID), b, 24*time.Hour).Err()
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

func (r *UserRepository) EnqueueAccessLog(ctx context.Context, event model.AccessLogEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return r.db.Rdb.RPush(ctx, "logs:queue", b).Err()
}

func (r *UserRepository) GetRoleIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.Pg.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, name).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("unknown role: %s", name)
		}
		return 0, fmt.Errorf("failed to get role: %w", err)
	}
	return id, nil
}

type CreateUserParams struct {
	Email        string
	LastName     string
	FirstName    string
	Patronymic   string
	Phone        string
	RoleID       int
	RoleName     string
	GroupID      *int
	PasswordHash string
}

func (r *UserRepository) CreateUser(ctx context.Context, p CreateUserParams) (*model.User, error) {
	tx, err := r.db.Pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userID int
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO users (role_id, email, last_name, first_name, patronymic, phone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, p.RoleID, p.Email, p.LastName, p.FirstName, p.Patronymic, p.Phone).Scan(&userID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO passwords (password_id, password_hash) VALUES ($1, $2)`,
		userID, p.PasswordHash,
	); err != nil {
		return nil, fmt.Errorf("failed to insert password: %w", err)
	}

	if p.RoleName == "student" {
		if p.GroupID == nil {
			return nil, fmt.Errorf("group_id is required for student role")
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO students (student_id, group_id) VALUES ($1, $2)`,
			userID, *p.GroupID,
		); err != nil {
			return nil, fmt.Errorf("failed to insert student: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &model.User{
		ID: userID, Email: p.Email, LastName: p.LastName, FirstName: p.FirstName,
		Patronymic: p.Patronymic, Role: p.RoleName, IsActive: true, CreatedAt: createdAt,
	}, nil
}

func (r *UserRepository) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := r.db.Pg.Query(ctx, `
		SELECT u.id, u.email, u.last_name, u.first_name, u.patronymic,
		       COALESCE(u.avatar_url, ''), ro.name, u.is_active, u.created_at
		FROM users u JOIN roles ro ON ro.id = u.role_id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.LastName, &u.FirstName, &u.Patronymic,
			&u.AvatarURL, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (r *UserRepository) UpdateUser(ctx context.Context, userID int, p model.UpdateUserRequest) error {
	cmd, err := r.db.Pg.Exec(ctx, `
		UPDATE users SET
			last_name  = COALESCE($2, last_name),
			first_name = COALESCE($3, first_name),
			patronymic = COALESCE($4, patronymic),
			phone      = COALESCE($5, phone),
			is_active  = COALESCE($6, is_active)
		WHERE id = $1
	`, userID, p.LastName, p.FirstName, p.Patronymic, p.Phone, p.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) ToggleInside(ctx context.Context, userID int) (bool, error) {
	var isInside bool
	err := r.db.Pg.QueryRow(ctx, `
		UPDATE users
		SET is_inside = NOT is_inside
		WHERE id = $1
		RETURNING is_inside
	`, userID).Scan(&isInside)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, fmt.Errorf("user not found")
		}
		return false, fmt.Errorf("failed to toggle inside state: %w", err)
	}
	return isInside, nil
}
