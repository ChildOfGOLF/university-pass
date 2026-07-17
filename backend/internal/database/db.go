package database

import (
	"context"
	"fmt"
	"time"
	"university-pass/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type DB struct {
	Pg  *pgxpool.Pool
	Rdb *redis.Client
}

const (
	pingAttempts = 10
	pingInterval = 2 * time.Second
)

func InitDB(ctx context.Context, cfg config.Config) (*DB, error) {
	pgPool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := waitForPostgres(ctx, pgPool); err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
	})

	if err := waitForRedis(ctx, rdb); err != nil {
		return nil, err
	}

	fmt.Println("Connected")
	return &DB{Pg: pgPool, Rdb: rdb}, nil
}

func waitForPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	var err error
	for i := 0; i < pingAttempts; i++ {
		if err = pool.Ping(ctx); err == nil {
			return nil
		}
		fmt.Println("waiting for postgres")
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for postgres: %w", ctx.Err())
		case <-time.After(pingInterval):
		}
	}
	return fmt.Errorf("failed to ping postgres %w", err)
}

func waitForRedis(ctx context.Context, rdb *redis.Client) error {
	var err error
	for i := 0; i < pingAttempts; i++ {
		if err = rdb.Ping(ctx).Err(); err == nil {
			return nil
		}
		fmt.Println("waiting for redis")
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for redis %w", ctx.Err())
		case <-time.After(pingInterval):
		}
	}
	return fmt.Errorf("failed to ping redis %w", err)
}
