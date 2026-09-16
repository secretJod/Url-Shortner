package config

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	cfg := &Config{
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

	if err := cfg.validateForProduction(); err != nil {
		log.Fatalf("config: refusing to start: %v", err)
	}

	return cfg
}

// isPlaceholder reports whether a secret value still looks like one of the
// committed .env.example / dev defaults rather than a real, operator-set
// secret. This is the safety net that stops a production deploy from
// silently running with publicly known credentials.
func isPlaceholder(v string) bool {
	if v == "" {
		return true
	}
	lower := strings.ToLower(v)
	if strings.HasPrefix(lower, "change-me") {
		return true
	}
	switch v {
	case "urlshortener_dev_pw", "dev-insecure-secret":
		return true
	}
	if strings.Contains(v, "urlshortener_dev_pw") {
		return true
	}
	return false
}

// validateForProduction enforces that required secrets are present and not
// left as dev/placeholder values when ENV=production. Local development
// (ENV=development, the default) is intentionally left permissive so that
// `docker compose up` with a copied .env.example keeps working out of the box.
func (c *Config) validateForProduction() error {
	if strings.ToLower(c.Env) != "production" {
		return nil
	}

	var problems []string

	checks := []struct {
		name  string
		value string
	}{
		{"DATABASE_URL", c.DatabaseURL},
		{"REDIS_PASSWORD", c.RedisPassword},
		{"IP_HASH_SECRET", c.IPHashSecret},
	}
	for _, c := range checks {
		if isPlaceholder(c.value) {
			problems = append(problems, c.name)
		}
	}

	if strings.Contains(c.BaseURL, "localhost") || strings.Contains(c.BaseURL, "127.0.0.1") {
		problems = append(problems, "BASE_URL (still points at localhost)")
	}

	if len(problems) > 0 {
		return fmt.Errorf("ENV=production but the following settings are missing or still use insecure/dev placeholder values: %s. Set real secrets (see deploy/README.md) before starting the app", strings.Join(problems, ", "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
