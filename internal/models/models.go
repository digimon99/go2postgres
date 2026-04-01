// Package models defines the core data structures for go2postgres.
package models

import (
	"time"
)

// User represents a system user.
type User struct {
	ID           string     `json:"user_id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	FullName     string     `json:"full_name,omitempty" db:"full_name"`
	Role         string     `json:"role" db:"role"` // "admin" or "user"
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsApproved   bool       `json:"is_approved" db:"is_approved"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"-" db:"deleted_at"`
}

// Instance represents a provisioned PostgreSQL database instance.
type Instance struct {
	ID                        string     `json:"instance_id" db:"id"`
	UserID                    string     `json:"user_id" db:"user_id"`
	ProjectID                 string     `json:"project_id" db:"project_id"`
	DatabaseName              string     `json:"database_name" db:"database_name"`
	PostgresUser              string     `json:"username" db:"postgres_user"`
	PostgresPasswordEncrypted string     `json:"-" db:"postgres_password_encrypted"`
	PostgresPasswordNonce     string     `json:"-" db:"postgres_password_nonce"`
	Host                      string     `json:"host" db:"host"`
	Port                      int        `json:"port" db:"port"`
	ConnectionLimit           int        `json:"connection_limit" db:"connection_limit"`
	StatementTimeoutMs        int        `json:"statement_timeout_ms" db:"statement_timeout_ms"`
	Extensions                string     `json:"-" db:"extensions"` // JSON array stored as string
	Status                    string     `json:"status" db:"status"` // "active", "suspended", "deleted"
	DiskUsageBytes            int64      `json:"disk_usage_bytes" db:"disk_usage_bytes"`
	ConnectionCount           int        `json:"connection_count" db:"connection_count"`
	LastHealthCheck           *time.Time `json:"last_health_check,omitempty" db:"last_health_check"`
	HealthStatus              string     `json:"health_status" db:"health_status"` // "healthy", "unhealthy", "unknown"
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt                 *time.Time `json:"-" db:"deleted_at"`
}

// InstanceExtensions returns the extensions as a slice.
func (i *Instance) GetExtensions() []string {
	if i.Extensions == "" || i.Extensions == "[]" {
		return []string{}
	}
	// Parse JSON array - simplified, use encoding/json in production
	return []string{} // TODO: implement proper JSON parsing
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID           string    `json:"log_id" db:"id"`
	UserID       *string   `json:"user_id,omitempty" db:"user_id"`
	Action       string    `json:"action" db:"action"` // e.g., "database.created", "user.login"
	ResourceType string    `json:"resource_type,omitempty" db:"resource_type"`
	ResourceID   string    `json:"resource_id,omitempty" db:"resource_id"`
	Metadata     string    `json:"metadata,omitempty" db:"metadata"` // JSON blob
	IPAddress    string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Setting represents a system setting.
type Setting struct {
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	Description string    `json:"description,omitempty" db:"description"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
	RevokedAt *time.Time `db:"revoked_at"`
}

// APIKey represents a database API key for zero-client HTTP query access.
type APIKey struct {
	ID          string     `json:"key_id" db:"id"`
	InstanceID  string     `json:"instance_id" db:"instance_id"`
	UserID      string     `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	KeyHash     string     `json:"-" db:"key_hash"`
	KeyPreview  string     `json:"key_preview" db:"key_preview"`
	KeyType     string     `json:"key_type" db:"key_type"` // "readonly" or "fullaccess"
	IPAllowlist string     `json:"ip_allowlist" db:"ip_allowlist"` // JSON array of CIDRs, "" = allow all
	IsActive    bool       `json:"is_active" db:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// API key types.
const (
	APIKeyTypeReadOnly   = "readonly"
	APIKeyTypeFullAccess = "fullaccess"
)

// Constants for roles and statuses.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"

	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusUnknown   = "unknown"
)
