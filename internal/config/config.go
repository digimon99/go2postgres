// Package config handles application configuration.
package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server
	ServerHost     string
	ServerPort     int
	TLSCertFile    string
	TLSKeyFile     string
	BaseDomain     string
	ShutdownTimeout time.Duration

	// Database (SQLite for metadata)
	SQLiteDBPath string

	// PostgreSQL
	PostgresHost       string
	PostgresPort       int
	PostgresSuperuser  string
	PostgresSuperPass  string
	PostgresBinPath    string
	PostgresDataDir    string

	// Security
	JWTSecret           string
	JWTAccessExpiry     time.Duration
	JWTRefreshExpiry    time.Duration
	EncryptionKey       string   // hex-encoded 32-byte key for AES-256
	EncryptionKeyBytes  []byte   // decoded 32 bytes
	BcryptCost          int
	MinPasswordLength   int
	AdminEmail          string
	AllowRegistration   bool

	// Rate Limiting
	RateLimitRequests   int
	RateLimitWindow     time.Duration
	RevealPasswordLimit int

	// Instance Defaults
	DefaultConnLimit       int
	DefaultStatementTimeout time.Duration
	MaxInstancesPerUser    int
	MaxDiskPerInstance     int64 // bytes

	// Observability
	LogLevel           string
	LogFormat          string
	MetricsEnabled     bool
	HealthCheckInterval time.Duration

	// Environment
	Environment string // "development", "staging", "production"

	// Email (Resend)
	ResendAPIKey string
	FromEmail    string
	OTPExpiry    time.Duration

	// Frontend
	FrontendURL string
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		// Server defaults
		ServerHost:      getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:      getEnvInt("SERVER_PORT", 8443),
		TLSCertFile:     getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:      getEnv("TLS_KEY_FILE", ""),
		BaseDomain:      getEnv("BASE_DOMAIN", "localhost"),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),

		// SQLite
		SQLiteDBPath: getEnv("SQLITE_DB_PATH", "./data/go2postgres.db"),

		// PostgreSQL
		PostgresHost:      getEnv("POSTGRES_HOST", "127.0.0.1"),
		PostgresPort:      getEnvInt("POSTGRES_PORT", 5438),
		PostgresSuperuser: getEnv("POSTGRES_SUPERUSER", "postgres"),
		PostgresSuperPass: getEnv("POSTGRES_SUPERPASS", ""),
		PostgresBinPath:   getEnv("POSTGRES_BIN_PATH", "/usr/lib/postgresql/16/bin"),
		PostgresDataDir:   getEnv("POSTGRES_DATA_DIR", "/var/lib/postgresql/16/main"),

		// Security
		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTAccessExpiry:   getEnvDuration("JWT_ACCESS_EXPIRY", 1*time.Hour),
		JWTRefreshExpiry:  getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		EncryptionKey:     getEnv("ENCRYPTION_KEY", ""),
		BcryptCost:        getEnvInt("BCRYPT_COST", 12),
		MinPasswordLength: getEnvInt("MIN_PASSWORD_LENGTH", 12),
		AdminEmail:        getEnv("ADMIN_EMAIL", ""),
		AllowRegistration: getEnvBool("ALLOW_REGISTRATION", true),

		// Rate limiting
		RateLimitRequests:   getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:     getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		RevealPasswordLimit: getEnvInt("REVEAL_PASSWORD_LIMIT", 3),

		// Instance defaults
		DefaultConnLimit:       getEnvInt("DEFAULT_CONN_LIMIT", 20),
		DefaultStatementTimeout: getEnvDuration("DEFAULT_STATEMENT_TIMEOUT", 30*time.Second),
		MaxInstancesPerUser:    getEnvInt("MAX_INSTANCES_PER_USER", 5),
		MaxDiskPerInstance:     getEnvInt64("MAX_DISK_PER_INSTANCE", 10*1024*1024*1024), // 10GB

		// Observability
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", true),
		HealthCheckInterval: getEnvDuration("HEALTH_CHECK_INTERVAL", 30*time.Second),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),

		// Email (Resend)
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		FromEmail:    getEnv("FROM_EMAIL", "go2postgres <noreply@go2postgres.dev>"),
		OTPExpiry:    getEnvDuration("OTP_EXPIRY", 10*time.Minute),

		// Frontend
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	var missing []string

	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.EncryptionKey == "" {
		missing = append(missing, "ENCRYPTION_KEY")
	}
	if c.PostgresSuperPass == "" {
		missing = append(missing, "POSTGRES_SUPERPASS")
	}

	// Decode hex-encoded encryption key
	if c.EncryptionKey != "" {
		keyBytes, err := hex.DecodeString(c.EncryptionKey)
		if err != nil {
			return fmt.Errorf("ENCRYPTION_KEY must be valid hex: %w", err)
		}
		if len(keyBytes) != 32 {
			return fmt.Errorf("ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(keyBytes))
		}
		c.EncryptionKeyBytes = keyBytes
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// PostgresDSN returns the PostgreSQL superuser connection string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable",
		c.PostgresSuperuser, c.PostgresSuperPass, c.PostgresHost, c.PostgresPort)
}

// Helper functions for parsing environment variables.

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
