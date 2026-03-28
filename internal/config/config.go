package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	AdminUser        string
	AdminPass        string
	JWTSecret        string
	JWTExpireHours   int
	DBType           string
	DBDSN            string
	Port             string
	SyncIntervalSecs int
}

func Load() *Config {
	cfg := &Config{
		AdminUser:        getEnv("ADMIN_USER", "admin"),
		AdminPass:        getEnv("ADMIN_PASS", "changeme"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTExpireHours:   getEnvInt("JWT_EXPIRE_HOURS", 24),
		DBType:           getEnv("DB_TYPE", "sqlite"),
		DBDSN:            getEnv("DB_DSN", "/data/sky-guardwall.db"),
		Port:             getEnv("PORT", "9176"),
		SyncIntervalSecs: getEnvInt("SYNC_INTERVAL_SECS", 60),
	}

	if cfg.JWTSecret == "" {
		slog.Warn("JWT_SECRET not set — using insecure default; tokens will be invalid after restart")
		cfg.JWTSecret = "sky-guardwall-insecure-default-secret"
	}
	if cfg.AdminPass == "changeme" {
		slog.Warn("ADMIN_PASS is default 'changeme' — please set a strong password in production")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
