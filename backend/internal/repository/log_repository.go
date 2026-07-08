package repository

import (
	"context"
	"university-pass/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LogRepository struct {
	db *pgxpool.Pool
}

func NewLogRepository(db *pgxpool.Pool) *LogRepository {
	return &LogRepository{db: db}
}

// Deprecated
func (lr *LogRepository) SaveAccessLog(ctx context.Context, log *model.AccessLog) error {
	query := `
		INSERT INTO access_logs (user_id, guest_pass_id, access_point_id, direction, is_allowed, reason, logged_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := lr.db.QueryRow(ctx, query,
		log.UserID,
		log.GuestPassID,
		log.AccessPointID,
		log.Direction,
		log.IsAllowed,
		log.Reason,
		log.LoggedAt,
	).Scan(&log.ID)

	return err
}

func (lr *LogRepository) SaveAccessLogBatch(ctx context.Context, logs []*model.AccessLog) error {
	if len(logs) == 0 {
		return nil
	}

	var batch pgx.Batch

	for _, log := range logs {
		batch.Queue(
			`INSERT INTO access_logs (user_id, guest_pass_id, access_point_id, direction, is_allowed, reason, logged_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			log.UserID,
			log.GuestPassID,
			log.AccessPointID,
			log.Direction,
			log.IsAllowed,
			log.Reason,
			log.LoggedAt,
		)
	}

	results := lr.db.SendBatch(ctx, &batch)
	defer results.Close()

	for range logs {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}

	return results.Close()
}
