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

// Для теста запись если точки нет
// TODO: лучше явно управлять данными в бд в проде
func (r *AccessPointRepository) GetOrCreateByScannerID(ctx context.Context, scannerID string) (int, error) {
	ap, err := r.GetByScannerID(ctx, scannerID)
	if err != nil {
		return 0, err
	}
	if ap != nil {
		return ap.ID, nil
	}

	var buildingID int
	err = r.db.Pg.QueryRow(ctx, `SELECT id FROM buildings LIMIT 1`).Scan(&buildingID)
	if err != nil {
		err = r.db.Pg.QueryRow(ctx, `INSERT INTO buildings (name, address) VALUES ($1, '') RETURNING id`, "default").Scan(&buildingID)
		if err != nil {
			return 0, fmt.Errorf("failed to create default building: %w", err)
		}
	}

	var newID int
	err = r.db.Pg.QueryRow(ctx,
		`INSERT INTO access_points (building_id, scanner_id, gate_number, description) VALUES ($1,$2,$3,$4) RETURNING id`,
		buildingID, scannerID, "unknown", "",
	).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("failed to create access_point: %w", err)
	}
	return newID, nil
}
