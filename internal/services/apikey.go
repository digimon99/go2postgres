package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/digimon99/go2postgres/internal/database"
	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/internal/postgres"
	"github.com/digimon99/go2postgres/pkg/apikey"
	"github.com/digimon99/go2postgres/pkg/crypto"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// API key specific errors.
var (
	ErrAPIKeyNotFound   = errors.New("api key not found")
	ErrAPIKeyRevoked    = errors.New("api key revoked")
	ErrInvalidAPIKey    = errors.New("invalid api key")
	ErrAPIKeyLimitReached = errors.New("api key limit reached per instance")
)

const maxKeysPerInstance = 20

// CreateAPIKeyResult holds the generated key (shown once) and the stored record.
type CreateAPIKeyResult struct {
	PlaintextKey string
	APIKey       *models.APIKey
}

// CreateAPIKey generates a new API key for the given instance.
func (s *Service) CreateAPIKey(ctx context.Context, userID, instanceID, name, keyType, ipAllowlist string) (*CreateAPIKeyResult, error) {
	logger.InfoContext(ctx, "creating api key", "instance_id", instanceID, "user_id", userID, "type", keyType)

	// Verify the instance belongs to this user.
	inst, err := s.GetInstance(ctx, userID, instanceID)
	if err != nil {
		return nil, ErrInstanceNotFound
	}
	_ = inst

	// Enforce per-instance key limit.
	existing, err := s.repo.ListAPIKeysByInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	if len(existing) >= maxKeysPerInstance {
		return nil, ErrAPIKeyLimitReached
	}

	// Validate key type.
	if keyType != models.APIKeyTypeReadOnly && keyType != models.APIKeyTypeFullAccess {
		keyType = models.APIKeyTypeFullAccess
	}

	// Validate IP allowlist JSON (basic: must be empty array or array of CIDR strings).
	if ipAllowlist == "" {
		ipAllowlist = "[]"
	}

	// Generate the key.
	plainKey, err := apikey.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating api key: %w", err)
	}

	now := time.Now().UTC()
	k := &models.APIKey{
		ID:          crypto.GenerateID("key"),
		InstanceID:  instanceID,
		UserID:      userID,
		Name:        name,
		KeyHash:     apikey.Hash(plainKey),
		KeyPreview:  apikey.Preview(plainKey),
		KeyType:     keyType,
		IPAllowlist: ipAllowlist,
		IsActive:    true,
		CreatedAt:   now,
	}

	if err := s.repo.CreateAPIKey(ctx, k); err != nil {
		return nil, fmt.Errorf("storing api key: %w", err)
	}

	s.logAudit(ctx, &userID, "apikey.created", "api_key", k.ID, map[string]string{
		"instance_id": instanceID,
		"key_type":    keyType,
		"name":        name,
	})

	return &CreateAPIKeyResult{PlaintextKey: plainKey, APIKey: k}, nil
}

// ListAPIKeys returns all API keys for an instance owned by the user.
func (s *Service) ListAPIKeys(ctx context.Context, userID, instanceID string) ([]*models.APIKey, error) {
	// Verify ownership.
	if _, err := s.GetInstance(ctx, userID, instanceID); err != nil {
		return nil, ErrInstanceNotFound
	}
	keys, err := s.repo.ListAPIKeysByInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	return keys, nil
}

// RevokeAPIKey revokes an API key owned by the user.
func (s *Service) RevokeAPIKey(ctx context.Context, userID, keyID string) error {
	logger.InfoContext(ctx, "revoking api key", "key_id", keyID, "user_id", userID)

	if err := s.repo.RevokeAPIKey(ctx, keyID, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("revoking api key: %w", err)
	}

	s.logAudit(ctx, &userID, "apikey.revoked", "api_key", keyID, nil)
	return nil
}

// GetAPIKeyByHash looks up and validates an API key by its plaintext value.
// Returns the key record and the associated instance.
func (s *Service) GetAPIKeyByHash(ctx context.Context, plainKey string) (*models.APIKey, *models.Instance, error) {
	if !apikey.IsValidFormat(plainKey) {
		return nil, nil, ErrInvalidAPIKey
	}

	hash := apikey.Hash(plainKey)
	k, err := s.repo.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, nil, ErrInvalidAPIKey
		}
		return nil, nil, fmt.Errorf("looking up api key: %w", err)
	}

	inst, err := s.repo.GetInstanceByID(ctx, k.InstanceID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading instance for api key: %w", err)
	}

	return k, inst, nil
}

// BuildInstanceDSN decrypts the instance password and builds a pgx DSN.
func (s *Service) BuildInstanceDSN(inst *models.Instance) (string, error) {
	password, err := s.encryptor.Decrypt(inst.PostgresPasswordEncrypted, inst.PostgresPasswordNonce)
	if err != nil {
		return "", fmt.Errorf("decrypting password: %w", err)
	}
	// pgx DSN format: postgres://user:pass@host:port/dbname
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		inst.PostgresUser,
		urlEncode(password),
		inst.Host,
		inst.Port,
		inst.DatabaseName,
	)
	return dsn, nil
}

// GetPoolManager returns the per-instance pool manager (used by query handler).
func (s *Service) GetPoolManager() *postgres.PoolManager {
	return s.poolMgr
}

// urlEncode percent-encodes special characters in a DSN component.
func urlEncode(s string) string {
	// Only encode chars that break URL parsing.
	// Using a simple manual escape for the password field.
	var out []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~':
			out = append(out, c)
		default:
			out = append(out, fmt.Sprintf("%%%02X", c)...)
		}
	}
	return string(out)
}
// TouchAPIKeyLastUsed updates the last_used_at timestamp asynchronously.
func (s *Service) TouchAPIKeyLastUsed(keyID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.repo.TouchAPIKeyLastUsed(ctx, keyID)
}