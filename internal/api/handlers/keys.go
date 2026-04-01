package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/digimon99/go2postgres/internal/api/middleware"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// CreateAPIKeyRequest is the request body for creating an API key.
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	KeyType     string `json:"key_type"` // "readonly" or "fullaccess" (default)
	IPAllowlist string `json:"ip_allowlist"` // JSON array of CIDRs, e.g. ["1.2.3.0/24"]
}

// CreateAPIKeyResponse is the response with the one-time plaintext key.
type CreateAPIKeyResponse struct {
	Key       string `json:"key"` // shown once
	KeyID     string `json:"key_id"`
	Name      string `json:"name"`
	KeyType   string `json:"key_type"`
	Preview   string `json:"key_preview"`
	CreatedAt string `json:"created_at"`
}

// CreateAPIKey creates a new API key for an instance.
// POST /api/v1/instances/:id/keys
func (h *Handler) CreateAPIKey(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)
	instanceID := c.Param("id")

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	result, err := h.svc.CreateAPIKey(c.Request.Context(), userID.(string), instanceID, req.Name, req.KeyType, req.IPAllowlist)
	if err != nil {
		switch err {
		case services.ErrInstanceNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		case services.ErrAPIKeyLimitReached:
			c.JSON(http.StatusConflict, gin.H{"error": "api key limit reached for this instance"})
		default:
			logger.ErrorContext(c.Request.Context(), "create api key failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api key"})
		}
		return
	}

	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		Key:       result.PlaintextKey,
		KeyID:     result.APIKey.ID,
		Name:      result.APIKey.Name,
		KeyType:   result.APIKey.KeyType,
		Preview:   result.APIKey.KeyPreview,
		CreatedAt: result.APIKey.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ListAPIKeys lists all API keys for an instance.
// GET /api/v1/instances/:id/keys
func (h *Handler) ListAPIKeys(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)
	instanceID := c.Param("id")

	keys, err := h.svc.ListAPIKeys(c.Request.Context(), userID.(string), instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		logger.ErrorContext(c.Request.Context(), "list api keys failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list api keys"})
		return
	}

	// Return safe representation (no key_hash)
	out := make([]gin.H, 0, len(keys))
	for _, k := range keys {
		out = append(out, gin.H{
			"key_id":      k.ID,
			"name":        k.Name,
			"key_preview": k.KeyPreview,
			"key_type":    k.KeyType,
			"ip_allowlist": k.IPAllowlist,
			"is_active":   k.IsActive,
			"last_used_at": formatNullableTime(k.LastUsedAt),
			"created_at":  k.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"keys": out})
}

// RevokeAPIKey revokes an API key.
// DELETE /api/v1/keys/:keyId
func (h *Handler) RevokeAPIKey(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)
	keyID := c.Param("keyId")

	if err := h.svc.RevokeAPIKey(c.Request.Context(), userID.(string), keyID); err != nil {
		if err == services.ErrAPIKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
			return
		}
		logger.ErrorContext(c.Request.Context(), "revoke api key failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke api key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "api key revoked"})
}

func formatNullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02T15:04:05Z")
}
