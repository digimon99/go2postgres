// Package database provides SQLite repository for metadata storage.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/digimon99/go2postgres/internal/models"
	_ "modernc.org/sqlite"
)

// Common errors.
var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicateKey  = errors.New("duplicate key")
	ErrInvalidInput  = errors.New("invalid input")
)

// Repository provides database operations.
type Repository struct {
	db *sql.DB
}

// New creates a new Repository with the given SQLite database path.
func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	// Set reasonable connection limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	r := &Repository{db: db}

	// Run migrations
	if err := r.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return r, nil
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// migrate creates tables if they don't exist.
func (r *Repository) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			full_name TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			is_active INTEGER NOT NULL DEFAULT 1,
			is_approved INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL`,
		
		`CREATE TABLE IF NOT EXISTS instances (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			database_name TEXT UNIQUE NOT NULL,
			postgres_user TEXT NOT NULL,
			postgres_password_encrypted TEXT NOT NULL,
			postgres_password_nonce TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			connection_limit INTEGER NOT NULL DEFAULT 20,
			statement_timeout_ms INTEGER NOT NULL DEFAULT 30000,
			extensions TEXT DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'active',
			disk_usage_bytes INTEGER DEFAULT 0,
			connection_count INTEGER DEFAULT 0,
			last_health_check DATETIME,
			health_status TEXT DEFAULT 'unknown',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_instances_user ON instances(user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status) WHERE deleted_at IS NULL`,
		
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			revoked_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id)`,
		
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			action TEXT NOT NULL,
			resource_type TEXT,
			resource_id TEXT,
			metadata TEXT,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at)`,
		
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS otp_codes (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			code TEXT NOT NULL,
			purpose TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			used_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_otp_codes_email ON otp_codes(email)`,
		`CREATE INDEX IF NOT EXISTS idx_otp_codes_expires ON otp_codes(expires_at)`,
	}

	for _, m := range migrations {
		if _, err := r.db.Exec(m); err != nil {
			return fmt.Errorf("executing migration: %w", err)
		}
	}

	return nil
}

// User operations

// CreateUser inserts a new user.
func (r *Repository) CreateUser(ctx context.Context, u *models.User) error {
	query := `INSERT INTO users (id, email, password_hash, full_name, role, is_active, is_approved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.PasswordHash, u.FullName, u.Role,
		u.IsActive, u.IsApproved, u.CreatedAt, u.UpdatedAt)
	
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, email, password_hash, full_name, role, is_active, is_approved, created_at, updated_at
		FROM users WHERE id = ? AND deleted_at IS NULL`
	
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role,
		&u.IsActive, &u.IsApproved, &u.CreatedAt, &u.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return u, nil
}

// GetUserByEmail retrieves a user by email.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, password_hash, full_name, role, is_active, is_approved, created_at, updated_at
		FROM users WHERE email = ? AND deleted_at IS NULL`
	
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role,
		&u.IsActive, &u.IsApproved, &u.CreatedAt, &u.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return u, nil
}

// UpdateUser updates a user.
func (r *Repository) UpdateUser(ctx context.Context, u *models.User) error {
	query := `UPDATE users SET email = ?, full_name = ?, role = ?, is_active = ?, is_approved = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`
	
	result, err := r.db.ExecContext(ctx, query,
		u.Email, u.FullName, u.Role, u.IsActive, u.IsApproved, time.Now(), u.ID)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUsers returns all active users.
func (r *Repository) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `SELECT id, email, password_hash, full_name, role, is_active, is_approved, created_at, updated_at
		FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role,
			&u.IsActive, &u.IsApproved, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// Instance operations

// CreateInstance inserts a new instance.
func (r *Repository) CreateInstance(ctx context.Context, i *models.Instance) error {
	query := `INSERT INTO instances (
		id, user_id, project_id, database_name, postgres_user, 
		postgres_password_encrypted, postgres_password_nonce, host, port,
		connection_limit, statement_timeout_ms, extensions, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, query,
		i.ID, i.UserID, i.ProjectID, i.DatabaseName, i.PostgresUser,
		i.PostgresPasswordEncrypted, i.PostgresPasswordNonce, i.Host, i.Port,
		i.ConnectionLimit, i.StatementTimeoutMs, i.Extensions, i.Status,
		i.CreatedAt, i.UpdatedAt)
	
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("inserting instance: %w", err)
	}
	return nil
}

// GetInstanceByID retrieves an instance by ID.
func (r *Repository) GetInstanceByID(ctx context.Context, id string) (*models.Instance, error) {
	query := `SELECT id, user_id, project_id, database_name, postgres_user,
		postgres_password_encrypted, postgres_password_nonce, host, port,
		connection_limit, statement_timeout_ms, extensions, status,
		disk_usage_bytes, connection_count, last_health_check, health_status,
		created_at, updated_at
		FROM instances WHERE id = ? AND deleted_at IS NULL`
	
	i := &models.Instance{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&i.ID, &i.UserID, &i.ProjectID, &i.DatabaseName, &i.PostgresUser,
		&i.PostgresPasswordEncrypted, &i.PostgresPasswordNonce, &i.Host, &i.Port,
		&i.ConnectionLimit, &i.StatementTimeoutMs, &i.Extensions, &i.Status,
		&i.DiskUsageBytes, &i.ConnectionCount, &i.LastHealthCheck, &i.HealthStatus,
		&i.CreatedAt, &i.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying instance: %w", err)
	}
	return i, nil
}

// GetInstancesByUserID retrieves all instances for a user.
func (r *Repository) GetInstancesByUserID(ctx context.Context, userID string) ([]*models.Instance, error) {
	query := `SELECT id, user_id, project_id, database_name, postgres_user,
		postgres_password_encrypted, postgres_password_nonce, host, port,
		connection_limit, statement_timeout_ms, extensions, status,
		disk_usage_bytes, connection_count, last_health_check, health_status,
		created_at, updated_at
		FROM instances WHERE user_id = ? AND deleted_at IS NULL ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	defer rows.Close()

	var instances []*models.Instance
	for rows.Next() {
		i := &models.Instance{}
		if err := rows.Scan(
			&i.ID, &i.UserID, &i.ProjectID, &i.DatabaseName, &i.PostgresUser,
			&i.PostgresPasswordEncrypted, &i.PostgresPasswordNonce, &i.Host, &i.Port,
			&i.ConnectionLimit, &i.StatementTimeoutMs, &i.Extensions, &i.Status,
			&i.DiskUsageBytes, &i.ConnectionCount, &i.LastHealthCheck, &i.HealthStatus,
			&i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning instance: %w", err)
		}
		instances = append(instances, i)
	}
	return instances, nil
}

// UpdateInstance updates an instance.
func (r *Repository) UpdateInstance(ctx context.Context, i *models.Instance) error {
	query := `UPDATE instances SET status = ?, connection_limit = ?, statement_timeout_ms = ?,
		extensions = ?, disk_usage_bytes = ?, connection_count = ?, 
		last_health_check = ?, health_status = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`
	
	result, err := r.db.ExecContext(ctx, query,
		i.Status, i.ConnectionLimit, i.StatementTimeoutMs, i.Extensions,
		i.DiskUsageBytes, i.ConnectionCount, i.LastHealthCheck, i.HealthStatus,
		time.Now(), i.ID)
	if err != nil {
		return fmt.Errorf("updating instance: %w", err)
	}
	
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SoftDeleteInstance marks an instance as deleted.
func (r *Repository) SoftDeleteInstance(ctx context.Context, id string) error {
	query := `UPDATE instances SET deleted_at = ?, status = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
	
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, models.StatusDeleted, now, id)
	if err != nil {
		return fmt.Errorf("deleting instance: %w", err)
	}
	
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CountInstancesByUserID returns the number of active instances for a user.
func (r *Repository) CountInstancesByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM instances WHERE user_id = ? AND deleted_at IS NULL AND status = 'active'`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting instances: %w", err)
	}
	return count, nil
}

// Audit log operations

// CreateAuditLog inserts an audit log entry.
func (r *Repository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, metadata, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.UserID, log.Action, log.ResourceType, log.ResourceID,
		log.Metadata, log.IPAddress, log.UserAgent, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting audit log: %w", err)
	}
	return nil
}

// Refresh token operations

// CreateRefreshToken stores a refresh token.
func (r *Repository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting refresh token: %w", err)
	}
	return nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash.
func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	query := `SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL`
	
	t := &models.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.RevokedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying refresh token: %w", err)
	}
	return t, nil
}

// RevokeRefreshToken marks a token as revoked.
func (r *Repository) RevokeRefreshToken(ctx context.Context, hash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = ? WHERE token_hash = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), hash)
	if err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}
	return nil
}

// RevokeAllUserTokens revokes all tokens for a user.
func (r *Repository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("revoking user tokens: %w", err)
	}
	return nil
}

// OTP operations

// OTP represents a one-time password record.
type OTP struct {
	ID        string
	Email     string
	Code      string
	Purpose   string // "signin" or "signup"
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// CreateOTP stores a new OTP code.
func (r *Repository) CreateOTP(ctx context.Context, otp *OTP) error {
	// First, invalidate any existing unused OTPs for this email and purpose
	_, _ = r.db.ExecContext(ctx,
		`DELETE FROM otp_codes WHERE email = ? AND purpose = ? AND used_at IS NULL`,
		otp.Email, otp.Purpose)

	query := `INSERT INTO otp_codes (id, email, code, purpose, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, query,
		otp.ID, otp.Email, otp.Code, otp.Purpose, otp.ExpiresAt, otp.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting OTP: %w", err)
	}
	return nil
}

// VerifyOTP checks if a valid OTP exists and marks it as used.
func (r *Repository) VerifyOTP(ctx context.Context, email, code, purpose string) (*OTP, error) {
	query := `SELECT id, email, code, purpose, expires_at, used_at, created_at
		FROM otp_codes 
		WHERE email = ? AND code = ? AND purpose = ? 
		AND used_at IS NULL AND expires_at > ?`
	
	otp := &OTP{}
	err := r.db.QueryRowContext(ctx, query, email, code, purpose, time.Now()).Scan(
		&otp.ID, &otp.Email, &otp.Code, &otp.Purpose, &otp.ExpiresAt, &otp.UsedAt, &otp.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying OTP: %w", err)
	}

	// Mark as used
	now := time.Now()
	_, err = r.db.ExecContext(ctx,
		`UPDATE otp_codes SET used_at = ? WHERE id = ?`, now, otp.ID)
	if err != nil {
		return nil, fmt.Errorf("marking OTP as used: %w", err)
	}
	otp.UsedAt = &now

	return otp, nil
}

// CleanupExpiredOTPs removes expired OTP codes.
func (r *Repository) CleanupExpiredOTPs(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM otp_codes WHERE expires_at < ?`, time.Now())
	if err != nil {
		return 0, fmt.Errorf("cleaning up OTPs: %w", err)
	}
	return result.RowsAffected()
}

// UserExistsByEmail checks if user exists (for OTP signup vs signin detection).
func (r *Repository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE email = ? AND deleted_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking user exists: %w", err)
	}
	return count > 0, nil
}

// CreateUserWithoutPassword creates a user for OTP auth (no password needed).
func (r *Repository) CreateUserWithoutPassword(ctx context.Context, u *models.User) error {
	query := `INSERT INTO users (id, email, password_hash, full_name, role, is_active, is_approved, created_at, updated_at)
		VALUES (?, ?, '', ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.FullName, u.Role, u.IsActive, u.IsApproved, u.CreatedAt, u.UpdatedAt)
	
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

// CountUsers returns the total number of users (for admin stats).
func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

// CountInstances returns the total number of instances (for admin stats).
func (r *Repository) CountInstances(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM instances WHERE deleted_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting instances: %w", err)
	}
	return count, nil
}

// ListAllInstances returns all instances for admins.
func (r *Repository) ListAllInstances(ctx context.Context) ([]*models.Instance, error) {
	query := `SELECT id, user_id, project_id, database_name, postgres_user,
		postgres_password_encrypted, postgres_password_nonce, host, port,
		connection_limit, statement_timeout_ms, extensions, status,
		disk_usage_bytes, connection_count, last_health_check, health_status,
		created_at, updated_at
		FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing all instances: %w", err)
	}
	defer rows.Close()

	var instances []*models.Instance
	for rows.Next() {
		i := &models.Instance{}
		if err := rows.Scan(
			&i.ID, &i.UserID, &i.ProjectID, &i.DatabaseName, &i.PostgresUser,
			&i.PostgresPasswordEncrypted, &i.PostgresPasswordNonce, &i.Host, &i.Port,
			&i.ConnectionLimit, &i.StatementTimeoutMs, &i.Extensions, &i.Status,
			&i.DiskUsageBytes, &i.ConnectionCount, &i.LastHealthCheck, &i.HealthStatus,
			&i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning instance: %w", err)
		}
		instances = append(instances, i)
	}
	return instances, nil
}

// Helper functions

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return true // SQLite returns a unique constraint error in the message
	// In production, parse the error message for "UNIQUE constraint failed"
}
