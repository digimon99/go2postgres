package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/gin-gonic/gin"
)

// API key context keys
const (
	ContextAPIKey     = "api_key"
	ContextAPIKeyInst = "api_key_instance"
)

// APIKeyAuth extracts an API key from Authorization header or X-API-Key header,
// validates it, checks IP allowlist, and stores the key + instance in context.
func APIKeyAuth(svc *services.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractAPIKey(c)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			return
		}

		keyRec, inst, err := svc.GetAPIKeyByHash(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		// Check IP allowlist if set
		if err := checkIPAllowlist(c, keyRec.IPAllowlist); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		// Check instance status
		if inst.Status != models.StatusActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "database instance is not active"})
			return
		}

		// Store in context
		c.Set(ContextAPIKey, keyRec)
		c.Set(ContextAPIKeyInst, inst)

		// Async update last_used_at (fire-and-forget)
		go svc.TouchAPIKeyLastUsed(keyRec.ID)

		c.Next()
	}
}

func extractAPIKey(c *gin.Context) string {
	// Check Authorization: Bearer <key>
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return strings.TrimSpace(parts[1])
		}
	}

	// Fallback to X-API-Key header
	return strings.TrimSpace(c.GetHeader("X-API-Key"))
}

func checkIPAllowlist(c *gin.Context, allowlistJSON string) error {
	if allowlistJSON == "" || allowlistJSON == "[]" {
		return nil // empty = allow all
	}

	var cidrs []string
	if err := json.Unmarshal([]byte(allowlistJSON), &cidrs); err != nil {
		return nil // malformed => allow (fail open for now)
	}
	if len(cidrs) == 0 {
		return nil
	}

	clientIP := c.ClientIP()
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return &ipDeniedError{clientIP}
	}

	for _, cidr := range cidrs {
		// Check for plain IP (no /)
		if !strings.Contains(cidr, "/") {
			if cidr == clientIP {
				return nil
			}
			continue
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return nil
		}
	}
	return &ipDeniedError{clientIP}
}

type ipDeniedError struct {
	ip string
}

func (e *ipDeniedError) Error() string {
	return "ip address not in allowlist"
}
