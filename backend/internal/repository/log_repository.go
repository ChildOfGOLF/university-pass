package repository

import (
	"context"
	"fmt"
	"time"
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

func (lr *LogRepository) SaveAccessLogBatch(ctx context.Context, logs []*model.AccessLog) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := lr.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx) // no-op если коммит прошел
	}()

	var batch pgx.Batch
	for _, log := range logs {
		batch.Queue(
			`INSERT INTO access_logs (user_id, guest_pass_id, access_point_id, direction, is_allowed, reason, logged_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			log.UserID, log.GuestPassID, log.AccessPointID,
			log.Direction, log.IsAllowed, log.Reason, log.LoggedAt,
		)
	}

	results := tx.SendBatch(ctx, &batch)
	for range logs {
		if _, err := results.Exec(); err != nil {
			if closeErr := results.Close(); closeErr != nil {
				return fmt.Errorf("batch insert failed: %w (failed to close batch: %v)", err, closeErr)
			}
			return fmt.Errorf("batch insert failed: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}

	return tx.Commit(ctx)
}

type ListAccessLogsFilter struct {
	UserID        *int
	GuestPassID   *string
	AccessPointID *int
	Direction     string
	IsAllowed     *bool
	From          *time.Time
	To            *time.Time
	Limit         int
	Offset        int
}

func (lr *LogRepository) ListAccessLogs(ctx context.Context, f ListAccessLogsFilter) ([]*model.AccessLogResponse, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	query := `
		SELECT
			al.id,
			CASE
				WHEN al.user_id IS NOT NULL THEN r.name
				ELSE 'guest'
			END AS person_type,
			CASE
				WHEN al.user_id IS NOT NULL THEN CONCAT_WS(' ', u.last_name, u.first_name, NULLIF(u.patronymic, ''))
				ELSE CONCAT_WS(' ', gp.last_name, gp.first_name, NULLIF(gp.patronymic, ''))
			END AS full_name,
			b.name AS building,
			COALESCE(ap.description, '') AS access_point,
			ap.gate_number AS gate,
			al.direction,
			al.is_allowed,
			COALESCE(al.reason, '') AS reason,
			al.logged_at
		FROM access_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN guest_passes gp ON al.guest_pass_id = gp.id
		JOIN access_points ap ON al.access_point_id = ap.id
		JOIN buildings b ON ap.building_id = b.id
		WHERE ($1::int IS NULL OR al.user_id = $1)
		  AND ($2::uuid IS NULL OR al.guest_pass_id = $2)
		  AND ($3::int IS NULL OR al.access_point_id = $3)
		  AND ($4 = '' OR al.direction = $4)
		  AND ($5::bool IS NULL OR al.is_allowed = $5)
		  AND ($6::timestamptz IS NULL OR al.logged_at >= $6)
		  AND ($7::timestamptz IS NULL OR al.logged_at <= $7)
		ORDER BY al.logged_at DESC
		LIMIT $8 OFFSET $9
	`

	rows, err := lr.db.Query(ctx, query,
		f.UserID, f.GuestPassID, f.AccessPointID, f.Direction, f.IsAllowed, f.From, f.To,
		limit, f.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list access logs: %w", err)
	}
	defer rows.Close()

	var result []*model.AccessLogResponse
	for rows.Next() {
		var l model.AccessLogResponse
		if err := rows.Scan(&l.ID, &l.PersonType, &l.FullName, &l.Building,
			&l.AccessPoint, &l.Gate, &l.Direction, &l.IsAllowed, &l.Reason, &l.LoggedAt); err != nil {
			return nil, err
		}
		result = append(result, &l)
	}
	return result, rows.Err()
}
