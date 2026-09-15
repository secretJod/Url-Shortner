package config

import (
	"os"
)

type Config struct {
	Port               string
	BaseURL            string
	Env                string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	CORSAllowedOrigins string
	IPHashSecret       string
	SMTPHost           string
	SMTPPort           string
	MailFrom           string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            0,
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://localhost:8080"),
		IPHashSecret:       getEnv("IP_HASH_SECRET", "dev-insecure-secret"),
		SMTPHost:           getEnv("SMTP_HOST", "localhost"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		MailFrom:           getEnv("MAIL_FROM", "no-reply@linksnip.local"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
