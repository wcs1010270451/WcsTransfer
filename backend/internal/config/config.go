package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName                     string
	Env                         string
	HTTPPort                    string
	GinMode                     string
	EnableDocs                  bool
	EnableAdminDebug            bool
	AdminBootstrapUsername      string
	AdminBootstrapPassword      string
	AdminBootstrapDisplayName   string
	AuthTokenSecret             string
	CORSAllowedOrigins          []string
	DatabaseURL                 string
	DatabaseMaxConns            int32
	DatabaseMinConns            int32
	RedisAddr                   string
	RedisPassword               string
	RedisDB                     int
	DependencyTimeout           time.Duration
	ReadTimeout                 time.Duration
	WriteTimeout                time.Duration
	ShutdownTimeout             time.Duration
	ReconciliationEnabled       bool
	ReconciliationInterval      time.Duration
	ReconciliationDiffThreshold float64
	ProviderAlertEnabled        bool
	ProviderAlertWindow         time.Duration
	ProviderAlertInterval       time.Duration
	ProviderAlertMinRequests    int
	ProviderAlert429Threshold   float64
	ProviderAlert5xxThreshold   float64
	TenantWalletAlertEnabled    bool
	TenantWalletAlertWindow     time.Duration
	TenantWalletAlertInterval   time.Duration
	TenantWalletAlertMinBlocks  int
	TenantReserveAlertMinBlocks int
	BillingAlertEnabled         bool
	BillingAlertWindow          time.Duration
	BillingAlertInterval        time.Duration
	BillingAlertMinCount        int
	BillingAlertMinAmount       float64
	DependencyAlertEnabled      bool
	DependencyAlertInterval     time.Duration
	AlertWebhookURL             string
	AlertWebhookProvider        string
	AlertWebhookTimeout         time.Duration
}

func Load() Config {
	env := getEnv("APP_ENV", "development")

	return Config{
		AppName:                   getEnv("APP_NAME", "wcstransfer-gateway"),
		Env:                       env,
		HTTPPort:                  getEnv("HTTP_PORT", "8080"),
		GinMode:                   getEnv("GIN_MODE", defaultGinMode(env)),
		EnableDocs:                getBool("ENABLE_DOCS", env != "production"),
		EnableAdminDebug:          getBool("ENABLE_ADMIN_DEBUG", env != "production"),
		AuthTokenSecret:           getEnv("AUTH_TOKEN_SECRET", "dev-auth-secret"),
		AdminBootstrapUsername:    getEnv("ADMIN_BOOTSTRAP_USERNAME", ""),
		AdminBootstrapPassword:    getEnv("ADMIN_BOOTSTRAP_PASSWORD", ""),
		AdminBootstrapDisplayName: getEnv("ADMIN_BOOTSTRAP_DISPLAY_NAME", "Platform Admin"),
		CORSAllowedOrigins: getCSVEnv("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:3211",
			"http://127.0.0.1:3211",
		}),
		DatabaseURL:                 getEnv("DATABASE_URL", ""),
		DatabaseMaxConns:            getInt32("DATABASE_MAX_CONNS", 20),
		DatabaseMinConns:            getInt32("DATABASE_MIN_CONNS", 2),
		RedisAddr:                   getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:               getEnv("REDIS_PASSWORD", ""),
		RedisDB:                     getInt("REDIS_DB", 0),
		DependencyTimeout:           getDuration("DEPENDENCY_TIMEOUT", 3*time.Second),
		ReadTimeout:                 getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:                getDuration("HTTP_WRITE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:             getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReconciliationEnabled:       getBool("RECONCILIATION_ENABLED", env == "production"),
		ReconciliationInterval:      getDuration("RECONCILIATION_INTERVAL", time.Hour),
		ReconciliationDiffThreshold: getFloat64("RECONCILIATION_DIFF_THRESHOLD", 0.0001),
		ProviderAlertEnabled:        getBool("PROVIDER_ALERT_ENABLED", env == "production"),
		ProviderAlertWindow:         getDuration("PROVIDER_ALERT_WINDOW", 5*time.Minute),
		ProviderAlertInterval:       getDuration("PROVIDER_ALERT_INTERVAL", time.Minute),
		ProviderAlertMinRequests:    getInt("PROVIDER_ALERT_MIN_REQUESTS", 10),
		ProviderAlert429Threshold:   getFloat64("PROVIDER_ALERT_429_THRESHOLD", 0.2),
		ProviderAlert5xxThreshold:   getFloat64("PROVIDER_ALERT_5XX_THRESHOLD", 0.2),
		TenantWalletAlertEnabled:    getBool("TENANT_WALLET_ALERT_ENABLED", env == "production"),
		TenantWalletAlertWindow:     getDuration("TENANT_WALLET_ALERT_WINDOW", 5*time.Minute),
		TenantWalletAlertInterval:   getDuration("TENANT_WALLET_ALERT_INTERVAL", time.Minute),
		TenantWalletAlertMinBlocks:  getInt("TENANT_WALLET_ALERT_MIN_BLOCKS", 5),
		TenantReserveAlertMinBlocks: getInt("TENANT_RESERVE_ALERT_MIN_BLOCKS", 5),
		BillingAlertEnabled:         getBool("BILLING_ALERT_ENABLED", env == "production"),
		BillingAlertWindow:          getDuration("BILLING_ALERT_WINDOW", 10*time.Minute),
		BillingAlertInterval:        getDuration("BILLING_ALERT_INTERVAL", time.Minute),
		BillingAlertMinCount:        getInt("BILLING_ALERT_MIN_COUNT", 1),
		BillingAlertMinAmount:       getFloat64("BILLING_ALERT_MIN_AMOUNT", 0.01),
		DependencyAlertEnabled:      getBool("DEPENDENCY_ALERT_ENABLED", env == "production"),
		DependencyAlertInterval:     getDuration("DEPENDENCY_ALERT_INTERVAL", time.Minute),
		AlertWebhookURL:             getEnv("ALERT_WEBHOOK_URL", ""),
		AlertWebhookProvider:        strings.ToLower(getEnv("ALERT_WEBHOOK_PROVIDER", "generic")),
		AlertWebhookTimeout:         getDuration("ALERT_WEBHOOK_TIMEOUT", 5*time.Second),
	}
}

func (c Config) Address() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func (c Config) Validate() error {
	if c.Env != "production" {
		return nil
	}

	issues := make([]string, 0)
	if insecureSecret(c.AuthTokenSecret) {
		issues = append(issues, "AUTH_TOKEN_SECRET is missing or uses an insecure default value")
	}
	if c.EnableDocs {
		issues = append(issues, "ENABLE_DOCS must be false in production unless explicitly reviewed")
	}
	if c.EnableAdminDebug {
		issues = append(issues, "ENABLE_ADMIN_DEBUG must be false in production")
	}
	if len(c.CORSAllowedOrigins) == 0 {
		issues = append(issues, "CORS_ALLOWED_ORIGINS must contain at least one trusted origin")
	}
	for _, origin := range c.CORSAllowedOrigins {
		lower := strings.ToLower(strings.TrimSpace(origin))
		switch {
		case lower == "":
			issues = append(issues, "CORS_ALLOWED_ORIGINS contains an empty origin")
		case lower == "*":
			issues = append(issues, "CORS_ALLOWED_ORIGINS must not contain wildcard '*' in production")
		case strings.Contains(lower, "localhost"), strings.Contains(lower, "127.0.0.1"):
			issues = append(issues, "CORS_ALLOWED_ORIGINS must not contain localhost origins in production")
		}
	}

	if len(issues) == 0 {
		return nil
	}

	return fmt.Errorf("invalid production configuration:\n- %s", strings.Join(issues, "\n- "))
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

func getFloat64(key string, fallback float64) float64 {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
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

func insecureSecret(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 24 {
		return true
	}
	lower := strings.ToLower(trimmed)
	badMarkers := []string{
		"change-me",
		"dev-auth-secret",
		"example",
		"test",
	}
	for _, marker := range badMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
