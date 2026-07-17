package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	JWTSecret   string
}

var JWTSecret = getEnv("JWT_SECRET", "vBS0K4W5DRo2iTQI1JmnuqIouvnHaBbsyvXxqk1Ibhz")

func Load() Config {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:postgres@postgres:5432/unipass?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "redis:6379"),
		RedisPass:   getEnv("REDIS_PASSWORD", ""),
		JWTSecret:   getEnv("JWT_SECRET", "vBS0K4W5DRo2iTQI1JmnuqIouvnHaBbsyvXxqk1Ibhz"),
	}

	if cfg.JWTSecret == "vBS0K4W5DRo2iTQI1JmnuqIouvnHaBbsyvXxqk1Ibhz" {
		fmt.Println("using default secret")
	}

	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
