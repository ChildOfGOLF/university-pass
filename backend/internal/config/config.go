package config

import (
	"log"
	"os"
)

type Config struct {
	Port        string
	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	JWTSecret   string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		PostgresDSN: mustGetEnv("POSTGRES_DSN"),
		RedisAddr:   getEnv("REDIS_ADDR", "redis:6379"),
		RedisPass:   getEnv("REDIS_PASSWORD", ""),
		JWTSecret:   mustGetEnv("JWT_SECRET"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("переменная окружения %s обязательна, но не задана", key)
	}
	return v
}
