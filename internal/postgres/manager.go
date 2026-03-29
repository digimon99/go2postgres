// Package postgres manages PostgreSQL database provisioning.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// Common errors.
var (
	ErrDatabaseExists    = errors.New("database already exists")
	ErrDatabaseNotFound  = errors.New("database not found")
	ErrUserExists        = errors.New("user already exists")
	ErrInvalidIdentifier = errors.New("invalid identifier")
)

// Manager handles PostgreSQL database operations.
type Manager struct {
	pool *pgxpool.Pool
	host string
	port int
}

// NewManager creates a new PostgreSQL manager.
func NewManager(dsn string, host string, port int) (*Manager, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	// Configure pool
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to PostgreSQL: %w", err)
	}

	return &Manager{
		pool: pool,
		host: host,
		port: port,
	}, nil
}

// Close closes the connection pool.
func (m *Manager) Close() {
	if m.pool != nil {
		m.pool.Close()
	}
}

// CreateDatabase creates a new database with an owner user.
func (m *Manager) CreateDatabase(ctx context.Context, dbName, username, password string, connLimit, stmtTimeoutMs int) error {
	// Validate identifiers
	if !isValidIdentifier(dbName) {
		return fmt.Errorf("%w: database name", ErrInvalidIdentifier)
	}
	if !isValidIdentifier(username) {
		return fmt.Errorf("%w: username", ErrInvalidIdentifier)
	}

	logger.InfoContext(ctx, "creating PostgreSQL database",
		"database", dbName, "user", username, "conn_limit", connLimit)

	// Create user
	createUserSQL := fmt.Sprintf(
		`CREATE USER %s WITH PASSWORD %s CONNECTION LIMIT %d`,
		pgx.Identifier{username}.Sanitize(),
		quoteLiteral(password),
		connLimit,
	)
	if _, err := m.pool.Exec(ctx, createUserSQL); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return ErrUserExists
		}
		return fmt.Errorf("creating user: %w", err)
	}

	// Create database
	createDBSQL := fmt.Sprintf(
		`CREATE DATABASE %s OWNER %s`,
		pgx.Identifier{dbName}.Sanitize(),
		pgx.Identifier{username}.Sanitize(),
	)
	if _, err := m.pool.Exec(ctx, createDBSQL); err != nil {
		// Rollback: drop the user
		dropUserSQL := fmt.Sprintf(`DROP USER IF EXISTS %s`, pgx.Identifier{username}.Sanitize())
		m.pool.Exec(ctx, dropUserSQL)
		
		if strings.Contains(err.Error(), "already exists") {
			return ErrDatabaseExists
		}
		return fmt.Errorf("creating database: %w", err)
	}

	// Set statement timeout for the user
	if stmtTimeoutMs > 0 {
		alterUserSQL := fmt.Sprintf(
			`ALTER ROLE %s SET statement_timeout = '%dms'`,
			pgx.Identifier{username}.Sanitize(),
			stmtTimeoutMs,
		)
		if _, err := m.pool.Exec(ctx, alterUserSQL); err != nil {
			logger.WarnContext(ctx, "failed to set statement timeout", "error", err)
		}
	}

	// Revoke public access
	revokeSQL := fmt.Sprintf(
		`REVOKE ALL ON DATABASE %s FROM PUBLIC`,
		pgx.Identifier{dbName}.Sanitize(),
	)
	if _, err := m.pool.Exec(ctx, revokeSQL); err != nil {
		logger.WarnContext(ctx, "failed to revoke public access", "error", err)
	}

	logger.InfoContext(ctx, "created PostgreSQL database successfully", "database", dbName)
	return nil
}

// DropDatabase drops a database and its owner user.
func (m *Manager) DropDatabase(ctx context.Context, dbName, username string) error {
	if !isValidIdentifier(dbName) || !isValidIdentifier(username) {
		return ErrInvalidIdentifier
	}

	logger.InfoContext(ctx, "dropping PostgreSQL database", "database", dbName, "user", username)

	// Terminate existing connections
	terminateSQL := fmt.Sprintf(`
		SELECT pg_terminate_backend(pid) 
		FROM pg_stat_activity 
		WHERE datname = %s AND pid <> pg_backend_pid()`,
		quoteLiteral(dbName),
	)
	if _, err := m.pool.Exec(ctx, terminateSQL); err != nil {
		logger.WarnContext(ctx, "failed to terminate connections", "error", err)
	}

	// Drop database
	dropDBSQL := fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, pgx.Identifier{dbName}.Sanitize())
	if _, err := m.pool.Exec(ctx, dropDBSQL); err != nil {
		return fmt.Errorf("dropping database: %w", err)
	}

	// Drop user
	dropUserSQL := fmt.Sprintf(`DROP USER IF EXISTS %s`, pgx.Identifier{username}.Sanitize())
	if _, err := m.pool.Exec(ctx, dropUserSQL); err != nil {
		return fmt.Errorf("dropping user: %w", err)
	}

	logger.InfoContext(ctx, "dropped PostgreSQL database successfully", "database", dbName)
	return nil
}

// SuspendDatabase revokes the user's login privilege.
func (m *Manager) SuspendDatabase(ctx context.Context, dbName, username string) error {
	if !isValidIdentifier(username) {
		return ErrInvalidIdentifier
	}

	logger.InfoContext(ctx, "suspending database", "database", dbName, "user", username)

	// Revoke login
	alterSQL := fmt.Sprintf(`ALTER USER %s NOLOGIN`, pgx.Identifier{username}.Sanitize())
	if _, err := m.pool.Exec(ctx, alterSQL); err != nil {
		return fmt.Errorf("revoking login: %w", err)
	}

	// Terminate existing connections
	terminateSQL := fmt.Sprintf(`
		SELECT pg_terminate_backend(pid) 
		FROM pg_stat_activity 
		WHERE datname = %s`,
		quoteLiteral(dbName),
	)
	m.pool.Exec(ctx, terminateSQL)

	return nil
}

// ResumeDatabase restores the user's login privilege.
func (m *Manager) ResumeDatabase(ctx context.Context, username string) error {
	if !isValidIdentifier(username) {
		return ErrInvalidIdentifier
	}

	logger.InfoContext(ctx, "resuming database access", "user", username)

	alterSQL := fmt.Sprintf(`ALTER USER %s LOGIN`, pgx.Identifier{username}.Sanitize())
	if _, err := m.pool.Exec(ctx, alterSQL); err != nil {
		return fmt.Errorf("granting login: %w", err)
	}

	return nil
}

// ChangePassword changes the user's password.
func (m *Manager) ChangePassword(ctx context.Context, username, newPassword string) error {
	if !isValidIdentifier(username) {
		return ErrInvalidIdentifier
	}

	alterSQL := fmt.Sprintf(
		`ALTER USER %s WITH PASSWORD %s`,
		pgx.Identifier{username}.Sanitize(),
		quoteLiteral(newPassword),
	)
	if _, err := m.pool.Exec(ctx, alterSQL); err != nil {
		return fmt.Errorf("changing password: %w", err)
	}

	return nil
}

// GetDatabaseSize returns the size of a database in bytes.
func (m *Manager) GetDatabaseSize(ctx context.Context, dbName string) (int64, error) {
	if !isValidIdentifier(dbName) {
		return 0, ErrInvalidIdentifier
	}

	query := `SELECT pg_database_size($1)`
	var size int64
	if err := m.pool.QueryRow(ctx, query, dbName).Scan(&size); err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return 0, ErrDatabaseNotFound
		}
		return 0, fmt.Errorf("getting database size: %w", err)
	}
	return size, nil
}

// GetConnectionCount returns the current connection count for a database.
func (m *Manager) GetConnectionCount(ctx context.Context, dbName string) (int, error) {
	query := `SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1`
	var count int
	if err := m.pool.QueryRow(ctx, query, dbName).Scan(&count); err != nil {
		return 0, fmt.Errorf("getting connection count: %w", err)
	}
	return count, nil
}

// DatabaseExists checks if a database exists.
func (m *Manager) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	var exists bool
	if err := m.pool.QueryRow(ctx, query, dbName).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking database existence: %w", err)
	}
	return exists, nil
}

// Ping checks if PostgreSQL is reachable.
func (m *Manager) Ping(ctx context.Context) error {
	return m.pool.Ping(ctx)
}

// GetHost returns the PostgreSQL host.
func (m *Manager) GetHost() string {
	return m.host
}

// GetPort returns the PostgreSQL port.
func (m *Manager) GetPort() int {
	return m.port
}

// EnableExtension enables a PostgreSQL extension in a database.
func (m *Manager) EnableExtension(ctx context.Context, dbName, extension string) error {
	// Connect to the specific database
	dsn := fmt.Sprintf("postgres://%s:%d/%s?sslmode=disable", m.host, m.port, dbName)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer conn.Close(ctx)

	createExtSQL := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS %s`, pgx.Identifier{extension}.Sanitize())
	if _, err := conn.Exec(ctx, createExtSQL); err != nil {
		return fmt.Errorf("creating extension: %w", err)
	}

	return nil
}

// Helper functions

// isValidIdentifier checks if a string is a valid PostgreSQL identifier.
var identifierRegex = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func isValidIdentifier(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	return identifierRegex.MatchString(strings.ToLower(s))
}

// quoteLiteral safely quotes a string literal for PostgreSQL.
func quoteLiteral(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}
