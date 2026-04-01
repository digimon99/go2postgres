// Package services implements business logic.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/digimon99/go2postgres/internal/auth"
	"github.com/digimon99/go2postgres/internal/config"
	"github.com/digimon99/go2postgres/internal/database"
	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/internal/postgres"
	"github.com/digimon99/go2postgres/pkg/crypto"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// Common errors.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotApproved    = errors.New("user not approved")
	ErrUserInactive       = errors.New("user is inactive")
	ErrEmailExists        = errors.New("email already registered")
	ErrInstanceNotFound   = errors.New("instance not found")
	ErrInstanceLimitReached = errors.New("instance limit reached")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrWeakPassword       = errors.New("password does not meet requirements")
)

// Service provides all business logic operations.
type Service struct {
	cfg       *config.Config
	repo      *database.Repository
	pgMgr     *postgres.Manager
	poolMgr   *postgres.PoolManager
	jwt       *auth.JWTManager
	encryptor *crypto.Encryptor
}

// NewService creates a new Service.
func NewService(cfg *config.Config, repo *database.Repository, pgMgr *postgres.Manager) (*Service, error) {
	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("creating encryptor: %w", err)
	}

	jwt := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)

	return &Service{
		cfg:       cfg,
		repo:      repo,
		pgMgr:     pgMgr,
		poolMgr:   postgres.NewPoolManager(),
		jwt:       jwt,
		encryptor: encryptor,
	}, nil
}

// --- User Management ---

// RegisterUser creates a new user account.
func (s *Service) RegisterUser(ctx context.Context, email, password, fullName string) (*models.User, error) {
	logger.InfoContext(ctx, "registering user", "email", email)

	// Validate password strength
	if err := crypto.ValidatePasswordStrength(password, s.cfg.MinPasswordLength); err != nil {
		return nil, ErrWeakPassword
	}

	// Check if email already exists
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, ErrEmailExists
	}

	// Hash password
	hashedPassword, err := crypto.HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	// Determine if this is the first user (admin)
	role := models.RoleUser
	isApproved := false
	
	// Check if this is the admin email
	if strings.EqualFold(email, s.cfg.AdminEmail) {
		role = models.RoleAdmin
		isApproved = true
	}

	now := time.Now()
	user := &models.User{
		ID:           crypto.GenerateID("usr"),
		Email:        strings.ToLower(email),
		PasswordHash: hashedPassword,
		FullName:     fullName,
		Role:         role,
		IsActive:     true,
		IsApproved:   isApproved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, database.ErrDuplicateKey) {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Log audit event
	s.logAudit(ctx, &user.ID, "user.registered", "user", user.ID, nil)

	return user, nil
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *models.User, err error) {
	logger.InfoContext(ctx, "user login attempt", "email", email)

	user, err = s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return "", "", nil, ErrInvalidCredentials
		}
		return "", "", nil, fmt.Errorf("fetching user: %w", err)
	}

	// Check password
	if !crypto.CheckPassword(password, user.PasswordHash) {
		s.logAudit(ctx, &user.ID, "user.login.failed", "user", user.ID, map[string]string{"reason": "invalid_password"})
		return "", "", nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return "", "", nil, ErrUserInactive
	}

	// Check if user is approved
	if !user.IsApproved {
		return "", "", nil, ErrUserNotApproved
	}

	// Generate tokens
	accessToken, refreshToken, err = s.jwt.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return "", "", nil, fmt.Errorf("generating tokens: %w", err)
	}

	// Store refresh token hash
	tokenHash := crypto.HashToken(refreshToken)
	rt := &models.RefreshToken{
		ID:        crypto.GenerateID("rt"),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.jwt.GetRefreshExpiry()),
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		logger.WarnContext(ctx, "failed to store refresh token", "error", err)
	}

	s.logAudit(ctx, &user.ID, "user.login.success", "user", user.ID, nil)

	return accessToken, refreshToken, user, nil
}

// RefreshTokens validates a refresh token and issues new tokens.
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error) {
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	// Verify token exists in database
	tokenHash := crypto.HashToken(refreshToken)
	storedToken, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return "", "", ErrUnauthorized
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	if !user.IsActive || !user.IsApproved {
		return "", "", ErrUnauthorized
	}

	// Revoke old token
	s.repo.RevokeRefreshToken(ctx, tokenHash)

	// Generate new tokens
	accessToken, newRefreshToken, err = s.jwt.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return "", "", fmt.Errorf("generating tokens: %w", err)
	}

	// Store new refresh token
	newTokenHash := crypto.HashToken(newRefreshToken)
	rt := &models.RefreshToken{
		ID:        crypto.GenerateID("rt"),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: time.Now().Add(s.jwt.GetRefreshExpiry()),
		CreatedAt: time.Now(),
	}
	s.repo.CreateRefreshToken(ctx, rt)

	return accessToken, newRefreshToken, nil
}

// Logout revokes a refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := crypto.HashToken(refreshToken)
	return s.repo.RevokeRefreshToken(ctx, tokenHash)
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// ApproveUser approves a user (admin only).
func (s *Service) ApproveUser(ctx context.Context, adminID, targetUserID string) error {
	user, err := s.repo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	user.IsApproved = true
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	s.logAudit(ctx, &adminID, "user.approved", "user", targetUserID, nil)
	return nil
}

// --- Instance Management ---

// CreateInstance provisions a new PostgreSQL database.
func (s *Service) CreateInstance(ctx context.Context, userID, projectID string, extensions []string) (*models.Instance, string, error) {
	logger.InfoContext(ctx, "creating instance", "user_id", userID, "project_id", projectID)

	// Check instance limit
	count, err := s.repo.CountInstancesByUserID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if count >= s.cfg.MaxInstancesPerUser {
		return nil, "", ErrInstanceLimitReached
	}

	// Generate unique names
	instanceID := crypto.GenerateID("inst")
	dbName := fmt.Sprintf("db_%s", strings.ToLower(projectID))
	username := fmt.Sprintf("u_%s", strings.ToLower(projectID))

	// Ensure valid PostgreSQL identifiers
	dbName = sanitizeIdentifier(dbName)
	username = sanitizeIdentifier(username)

	// Purge any soft-deleted instances with the same database_name
	// This allows re-creating instances with the same project_id after deletion
	if err := s.repo.PurgeSoftDeletedByDatabaseName(ctx, dbName); err != nil {
		logger.WarnContext(ctx, "failed to purge soft-deleted instance", "database", dbName, "error", err)
	}

	// Generate secure password
	password, err := crypto.GenerateSecurePassword(24)
	if err != nil {
		return nil, "", fmt.Errorf("generating password: %w", err)
	}

	// Encrypt password for storage
	encryptedPass, nonce, err := s.encryptor.Encrypt(password)
	if err != nil {
		return nil, "", fmt.Errorf("encrypting password: %w", err)
	}

	// Create database in PostgreSQL
	connLimit := s.cfg.DefaultConnLimit
	stmtTimeout := int(s.cfg.DefaultStatementTimeout.Milliseconds())
	
	if err := s.pgMgr.CreateDatabase(ctx, dbName, username, password, connLimit, stmtTimeout); err != nil {
		return nil, "", fmt.Errorf("creating database: %w", err)
	}

	// Enable extensions
	extJSON := "[]"
	if len(extensions) > 0 {
		for _, ext := range extensions {
			if err := s.pgMgr.EnableExtension(ctx, dbName, ext); err != nil {
				logger.WarnContext(ctx, "failed to enable extension", "extension", ext, "error", err)
			}
		}
		extBytes, _ := json.Marshal(extensions)
		extJSON = string(extBytes)
	}

	// Create instance record
	now := time.Now()
	instance := &models.Instance{
		ID:                        instanceID,
		UserID:                    userID,
		ProjectID:                 projectID,
		DatabaseName:              dbName,
		PostgresUser:              username,
		PostgresPasswordEncrypted: encryptedPass,
		PostgresPasswordNonce:     nonce,
		Host:                      s.pgMgr.GetHost(),
		Port:                      s.pgMgr.GetPort(),
		ConnectionLimit:           connLimit,
		StatementTimeoutMs:        stmtTimeout,
		Extensions:                extJSON,
		Status:                    models.StatusActive,
		HealthStatus:              models.HealthStatusUnknown,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	if err := s.repo.CreateInstance(ctx, instance); err != nil {
		// Rollback: drop the database
		s.pgMgr.DropDatabase(ctx, dbName, username)
		return nil, "", fmt.Errorf("storing instance: %w", err)
	}

	s.logAudit(ctx, &userID, "instance.created", "instance", instanceID, map[string]string{
		"database": dbName,
		"project":  projectID,
	})

	return instance, password, nil
}

// GetInstance retrieves an instance by ID.
func (s *Service) GetInstance(ctx context.Context, userID, instanceID string) (*models.Instance, error) {
	instance, err := s.repo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}

	// Check ownership (unless admin check is done elsewhere)
	if instance.UserID != userID {
		return nil, ErrUnauthorized
	}

	return instance, nil
}

// GetUserInstances retrieves all instances for a user.
func (s *Service) GetUserInstances(ctx context.Context, userID string) ([]*models.Instance, error) {
	return s.repo.GetInstancesByUserID(ctx, userID)
}

// DeleteInstance deletes a database instance.
func (s *Service) DeleteInstance(ctx context.Context, userID, instanceID string) error {
	instance, err := s.GetInstance(ctx, userID, instanceID)
	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "deleting instance", "instance_id", instanceID, "database", instance.DatabaseName)

	// Drop database in PostgreSQL
	if err := s.pgMgr.DropDatabase(ctx, instance.DatabaseName, instance.PostgresUser); err != nil {
		logger.WarnContext(ctx, "failed to drop PostgreSQL database", "error", err)
	}

	// Mark as deleted in metadata
	if err := s.repo.SoftDeleteInstance(ctx, instanceID); err != nil {
		return err
	}

	s.logAudit(ctx, &userID, "instance.deleted", "instance", instanceID, nil)
	return nil
}

// RevealPassword decrypts and returns the database password.
func (s *Service) RevealPassword(ctx context.Context, userID, instanceID string) (string, error) {
	instance, err := s.GetInstance(ctx, userID, instanceID)
	if err != nil {
		return "", err
	}

	password, err := s.encryptor.Decrypt(instance.PostgresPasswordEncrypted, instance.PostgresPasswordNonce)
	if err != nil {
		return "", fmt.Errorf("decrypting password: %w", err)
	}

	s.logAudit(ctx, &userID, "instance.password.revealed", "instance", instanceID, nil)
	return password, nil
}

// SuspendInstance suspends a database (admin only).
func (s *Service) SuspendInstance(ctx context.Context, adminID, instanceID string) error {
	instance, err := s.repo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return ErrInstanceNotFound
	}

	if err := s.pgMgr.SuspendDatabase(ctx, instance.DatabaseName, instance.PostgresUser); err != nil {
		return err
	}

	instance.Status = models.StatusSuspended
	if err := s.repo.UpdateInstance(ctx, instance); err != nil {
		return err
	}

	s.logAudit(ctx, &adminID, "instance.suspended", "instance", instanceID, nil)
	return nil
}

// ResumeInstance resumes a suspended database (admin only).
func (s *Service) ResumeInstance(ctx context.Context, adminID, instanceID string) error {
	instance, err := s.repo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return ErrInstanceNotFound
	}

	if err := s.pgMgr.ResumeDatabase(ctx, instance.PostgresUser); err != nil {
		return err
	}

	instance.Status = models.StatusActive
	if err := s.repo.UpdateInstance(ctx, instance); err != nil {
		return err
	}

	s.logAudit(ctx, &adminID, "instance.resumed", "instance", instanceID, nil)
	return nil
}

// --- Health Checks ---

// HealthCheck performs a health check on an instance.
func (s *Service) HealthCheck(ctx context.Context, instance *models.Instance) error {
	// Get current metrics
	size, err := s.pgMgr.GetDatabaseSize(ctx, instance.DatabaseName)
	if err != nil {
		instance.HealthStatus = models.HealthStatusUnhealthy
	} else {
		instance.DiskUsageBytes = size
		instance.HealthStatus = models.HealthStatusHealthy
	}

	connCount, _ := s.pgMgr.GetConnectionCount(ctx, instance.DatabaseName)
	instance.ConnectionCount = connCount

	now := time.Now()
	instance.LastHealthCheck = &now

	return s.repo.UpdateInstance(ctx, instance)
}

// ValidateToken validates a JWT access token.
func (s *Service) ValidateToken(tokenString string) (*auth.Claims, error) {
	return s.jwt.ValidateAccessToken(tokenString)
}

// --- Helpers ---

func (s *Service) logAudit(ctx context.Context, userID *string, action, resourceType, resourceID string, metadata map[string]string) {
	metaJSON := ""
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		metaJSON = string(b)
	}

	log := &models.AuditLog{
		ID:           crypto.GenerateID("log"),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metaJSON,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateAuditLog(ctx, log); err != nil {
		logger.WarnContext(ctx, "failed to create audit log", "error", err)
	}
}

// sanitizeIdentifier makes a string safe for use as a PostgreSQL identifier.
func sanitizeIdentifier(s string) string {
	// Keep only alphanumeric and underscore
	var result strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result.WriteRune(c)
		}
	}
	
	// Ensure it starts with a letter
	r := result.String()
	if len(r) > 0 && r[0] >= '0' && r[0] <= '9' {
		r = "x" + r
	}
	
	// Truncate to 63 characters
	if len(r) > 63 {
		r = r[:63]
	}
	
	return r
}

// --- Admin Methods ---

// CountUsers returns total user count.
func (s *Service) CountUsers(ctx context.Context) (int, error) {
	return s.repo.CountUsers(ctx)
}

// CountInstances returns total instance count.
func (s *Service) CountInstances(ctx context.Context) (int, error) {
	return s.repo.CountInstances(ctx)
}

// ListUsers returns all users (admin).
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return s.repo.ListUsers(ctx, limit, offset)
}

// ListAllInstances returns all instances (admin).
func (s *Service) ListAllInstances(ctx context.Context) ([]*models.Instance, error) {
	return s.repo.ListAllInstances(ctx)
}
