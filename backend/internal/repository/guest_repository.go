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

	if r.db.Rdb != nil {
		val, err := r.db.Rdb.Get(ctx, key).Result()
		if err == nil {
			var g model.GuestPass
			if err := json.Unmarshal([]byte(val), &g); err == nil {
				return &g, nil
			}
		} else if err != redis.Nil {
			return nil, err
		}
	}

	query := `
		SELECT id, last_name, first_name, patronymic, purpose, valid_from, valid_to, is_used, is_entered, created_at
		FROM guest_passes
		WHERE id = $1
	`

	var g model.GuestPass
	err := r.db.Pg.QueryRow(ctx, query, guestID).Scan(
		&g.ID,
		&g.LastName,
		&g.FirstName,
		&g.Patronymic,
		&g.Purpose,
		&g.ValidFrom,
		&g.ValidTo,
		&g.IsUsed,
		&g.IsEntered,
		&g.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if r.db.Rdb != nil {
		if b, err := json.Marshal(g); err == nil {
			_ = r.db.Rdb.Set(ctx, key, b, 10*time.Minute).Err()
		}
	}

	return &g, nil
}

func (r *GuestRepository) MarkGuestPassEnteredIfValid(ctx context.Context, guestID string) (bool, error) {
	query := `
		UPDATE guest_passes
		SET is_used = TRUE,
		    is_entered = TRUE
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

	if r.db.Rdb != nil {
		_ = r.db.Rdb.Del(ctx, fmt.Sprintf("guest_pass:%s", guestID))
	}

	return true, nil
}

func (r *GuestRepository) MarkGuestPassExited(ctx context.Context, guestID string) (bool, error) {
	query := `
		UPDATE guest_passes
		SET is_entered = FALSE
		WHERE id = $1
		  AND is_entered = TRUE
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

	if r.db.Rdb != nil {
		_ = r.db.Rdb.Del(ctx, fmt.Sprintf("guest_pass:%s", guestID))
	}

	return true, nil
}

func (r *GuestRepository) EnqueueAccessLog(ctx context.Context, event model.AccessLogEvent) error {
	if r.db.Rdb == nil {
		return nil
	}

	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return r.db.Rdb.RPush(ctx, "logs:queue", b).Err()
}
