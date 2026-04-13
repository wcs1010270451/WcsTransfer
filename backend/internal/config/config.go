package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName            string
	Env                string
	HTTPPort           string
	GinMode            string
	AdminToken         string
	CORSAllowedOrigins []string
	DatabaseURL        string
	DatabaseMaxConns   int32
	DatabaseMinConns   int32
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	DependencyTimeout  time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ShutdownTimeout    time.Duration
}

func Load() Config {
	env := getEnv("APP_ENV", "development")

	return Config{
		AppName:    getEnv("APP_NAME", "wcstransfer-gateway"),
		Env:        env,
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		GinMode:    getEnv("GIN_MODE", defaultGinMode(env)),
		AdminToken: getEnv("ADMIN_TOKEN", ""),
		CORSAllowedOrigins: getCSVEnv("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:3211",
			"http://127.0.0.1:3211",
		}),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		DatabaseMaxConns:  getInt32("DATABASE_MAX_CONNS", 20),
		DatabaseMinConns:  getInt32("DATABASE_MIN_CONNS", 2),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getInt("REDIS_DB", 0),
		DependencyTimeout: getDuration("DEPENDENCY_TIMEOUT", 3*time.Second),
		ReadTimeout:       getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      getDuration("HTTP_WRITE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:   getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
	}
}

func (c Config) Address() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func defaultGinMode(env string) string {
	if env == "production" {
		return "release"
	}

	return "debug"
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getInt32(key string, fallback int32) int32 {
	return int32(getInt(key, int(fallback)))
}

func getCSVEnv(key string, fallback []string) []string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	if len(items) == 0 {
		return fallback
	}

	return items
}
