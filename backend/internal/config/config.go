package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName            string
	Env                string
	HTTPAddr           string
	DatabaseDSN        string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	CORSOrigins        []string
	BootstrapEmail     string
	BootstrapPassword  string
	BootstrapName      string
	EncryptionKey      string
	DefaultUserBalance int64
}

func Load() Config {
	return Config{
		AppName:            env("BITAPI_APP_NAME", "BitAPI"),
		Env:                env("BITAPI_ENV", "development"),
		HTTPAddr:           env("BITAPI_HTTP_ADDR", ":8080"),
		DatabaseDSN:        env("BITAPI_DATABASE_DSN", "file:data/bitapi.db?_foreign_keys=on&_busy_timeout=5000"),
		JWTSecret:          env("BITAPI_JWT_SECRET", "change-me-bitapi-dev-secret"),
		AccessTokenTTL:     durationEnv("BITAPI_ACCESS_TOKEN_TTL", 30*time.Minute),
		RefreshTokenTTL:    durationEnv("BITAPI_REFRESH_TOKEN_TTL", 14*24*time.Hour),
		CORSOrigins:        csvEnv("BITAPI_CORS_ORIGINS", []string{"http://localhost:5173", "http://127.0.0.1:5173"}),
		BootstrapEmail:     env("BITAPI_BOOTSTRAP_EMAIL", "admin@bitapi.local"),
		BootstrapPassword:  env("BITAPI_BOOTSTRAP_PASSWORD", "bitapi-admin"),
		BootstrapName:      env("BITAPI_BOOTSTRAP_NAME", "BitAPI 管理员"),
		EncryptionKey:      env("BITAPI_ENCRYPTION_KEY", "dev-bitapi-encryption-key-32bytes"),
		DefaultUserBalance: int64Env("BITAPI_DEFAULT_USER_BALANCE_MICROS", 0),
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func csvEnv(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func int64Env(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}
