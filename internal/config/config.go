package config

import (
	"os"
)

type Config struct {
	Port          string
	BaseURL       string
	Env           string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		Env:           getEnv("ENV", "development"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
