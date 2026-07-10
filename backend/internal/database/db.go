package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type DB struct {
	Pg  *pgxpool.Pool
	Rdb *redis.Client
}

func InitDB(ctx context.Context) (*DB, error) {
	// TODO: move to env
	pgConnStr := "postgres://postgres:postgres@postgres:5432/unipass?sslmode=disable"

	pgPool, err := pgxpool.New(ctx, pgConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	for i := 0; i < 10; i++ {
		err = pgPool.Ping(ctx)
		if err == nil {
			break
		}

		fmt.Println("waiting for postgres...")
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	fmt.Println("Connected")

	return &DB{
		Pg:  pgPool,
		Rdb: rdb,
	}, nil
}
