package repository

import (
	"context"
	"fmt"
	"university-pass/internal/database"
	"university-pass/internal/model"

	"github.com/jackc/pgx/v5"
)

type AccessPointRepository struct {
	db *database.DB
}

func NewAccessPointRepository(db *database.DB) *AccessPointRepository {
	return &AccessPointRepository{db: db}
}

func (r *AccessPointRepository) GetByScannerID(ctx context.Context, scannerID string) (*model.AccessPoint, error) {
	query := `SELECT id, building_id, scanner_id, gate_number, description FROM access_points WHERE scanner_id = $1`
	var ap model.AccessPoint
	err := r.db.Pg.QueryRow(ctx, query, scannerID).Scan(&ap.ID, &ap.BuildingID, &ap.ScannerID, &ap.GateNumber, &ap.Description)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get access point by scanner_id: %w", err)
	}
	return &ap, nil
}

func (r *AccessPointRepository) GetByAPIKey(ctx context.Context, apiKey string) (*model.AccessPoint, error) {
	query := `SELECT id, building_id, scanner_id, gate_number, description FROM access_points WHERE api_key = $1`
	var ap model.AccessPoint
	err := r.db.Pg.QueryRow(ctx, query, apiKey).Scan(&ap.ID, &ap.BuildingID, &ap.ScannerID, &ap.GateNumber, &ap.Description)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get access point by api_key: %w", err)
	}
	return &ap, nil
}
