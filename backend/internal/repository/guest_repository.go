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

type GuestRepository struct {
	db *database.DB
}

func NewGuestRepository(db *database.DB) *GuestRepository {
	return &GuestRepository{db: db}
}

func (r *GuestRepository) GetGuestPassByID(ctx context.Context, guestID string) (*model.GuestPass, error) {
	key := fmt.Sprintf("guest_pass:%s", guestID)

	val, err := r.db.Rdb.Get(ctx, key).Result()
	if err == nil {
		var g model.GuestPass
		if err := json.Unmarshal([]byte(val), &g); err == nil {
			return &g, nil
		}
	} else if err != redis.Nil {
		return nil, err
	}

	query := `
		SELECT id, last_name, first_name, patronymic, purpose, valid_from, valid_to, is_used, is_entered, created_at
		FROM guest_passes
		WHERE id = $1
	`

	var g model.GuestPass
	err = r.db.Pg.QueryRow(ctx, query, guestID).Scan(
		&g.ID, &g.LastName, &g.FirstName, &g.Patronymic, &g.Purpose,
		&g.ValidFrom, &g.ValidTo, &g.IsUsed, &g.IsEntered, &g.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if b, err := json.Marshal(g); err == nil {
		_ = r.db.Rdb.Set(ctx, key, b, 10*time.Minute).Err()
	}

	return &g, nil
}

func (r *GuestRepository) MarkGuestPassEnteredIfValid(ctx context.Context, guestID string) (bool, error) {
	query := `
		UPDATE guest_passes
		SET is_used = TRUE, is_entered = TRUE
		WHERE id = $1
		  AND is_used = FALSE
		  AND is_entered = FALSE
		  AND valid_from <= NOW()
		  AND valid_to >= NOW()
		RETURNING id
	`

	var returnedID string
	err := r.db.Pg.QueryRow(ctx, query, guestID).Scan(&returnedID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	_ = r.db.Rdb.Del(ctx, fmt.Sprintf("guest_pass:%s", guestID)).Err()

	return true, nil
}

func (r *GuestRepository) MarkGuestPassExited(ctx context.Context, guestID string) (bool, error) {
	query := `
		UPDATE guest_passes
		SET is_entered = FALSE
		WHERE id = $1 AND is_entered = TRUE
		RETURNING id
	`

	var returnedID string
	err := r.db.Pg.QueryRow(ctx, query, guestID).Scan(&returnedID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	_ = r.db.Rdb.Del(ctx, fmt.Sprintf("guest_pass:%s", guestID)).Err()

	return true, nil
}

func (r *GuestRepository) EnqueueAccessLog(ctx context.Context, event model.AccessLogEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return r.db.Rdb.RPush(ctx, "logs:queue", b).Err()
}

func (r *GuestRepository) CreateGuestPass(ctx context.Context, g *model.GuestPass) (*model.GuestPass, error) {
	err := r.db.Pg.QueryRow(ctx, `
		INSERT INTO guest_passes (last_name, first_name, patronymic, purpose, valid_from, valid_to)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, is_used, is_entered, created_at
	`, g.LastName, g.FirstName, g.Patronymic, g.Purpose, g.ValidFrom, g.ValidTo).
		Scan(&g.ID, &g.IsUsed, &g.IsEntered, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest pass: %w", err)
	}
	return g, nil
}

func (r *GuestRepository) ListGuestPasses(ctx context.Context) ([]*model.GuestPass, error) {
	rows, err := r.db.Pg.Query(ctx, `
		SELECT id, last_name, first_name, patronymic, COALESCE(purpose, ''),
		       valid_from, valid_to, is_used, is_entered, created_at
		FROM guest_passes ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list guest passes: %w", err)
	}
	defer rows.Close()

	var result []*model.GuestPass
	for rows.Next() {
		var g model.GuestPass
		if err := rows.Scan(&g.ID, &g.LastName, &g.FirstName, &g.Patronymic, &g.Purpose,
			&g.ValidFrom, &g.ValidTo, &g.IsUsed, &g.IsEntered, &g.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &g)
	}
	return result, rows.Err()
}

// Если гость не внутри
func (r *GuestRepository) RevokeGuestPass(ctx context.Context, id string) (bool, error) {
	cmd, err := r.db.Pg.Exec(ctx, `
		UPDATE guest_passes SET is_used = true
		WHERE id = $1 AND is_used = false AND is_entered = false
	`, id)
	if err != nil {
		return false, fmt.Errorf("failed to revoke guest pass: %w", err)
	}

	revoked := cmd.RowsAffected() > 0
	if revoked {
		_ = r.db.Rdb.Del(ctx, fmt.Sprintf("guest_pass:%s", id)).Err()
	}
	return revoked, nil
}
