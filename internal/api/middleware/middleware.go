// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/digimon99/go2postgres/internal/auth"
	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/digimon99/go2postgres/pkg/crypto"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// Context keys.
const (
	ContextUserID   = "user_id"
	ContextUserRole = "user_role"
	ContextClaims   = "claims"
)

// TokenValidator validates JWT tokens.
type TokenValidator interface {
	ValidateToken(token string) (*auth.Claims, error)
}

// Auth returns a middleware that validates JWT tokens.
func Auth(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := validator.ValidateToken(parts[1])
		if err != nil {
			if err == auth.ErrExpiredToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Store claims in context
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)
		c.Set(ContextClaims, claims)

		// Add to request context for logging
		ctx := context.WithValue(c.Request.Context(), logger.UserIDKey, claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireAdmin returns middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextUserRole)
		if !exists || role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

// RequireApproved returns middleware that requires an approved user.
func RequireApproved(svc *services.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(ContextUserID)
		user, err := svc.GetUser(c.Request.Context(), userID.(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if !user.IsApproved {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account pending approval"})
			return
		}
		c.Next()
	}
}

// RateLimiter implements a simple in-memory rate limiter.
type RateLimiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	limit     int
	window    time.Duration
	cleanupInterval time.Duration
}

type visitor struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
		cleanupInterval: 5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, v := range rl.visitors {
			if now.After(v.resetAt) {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[key]

	if !exists || now.After(v.resetAt) {
		rl.visitors[key] = &visitor{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true
	}

	if v.count >= rl.limit {
		return false
	}

	v.count++
	return true
}

// Remaining returns the number of remaining requests.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists || time.Now().After(v.resetAt) {
		return rl.limit
	}
	return rl.limit - v.count
}

// RateLimit returns middleware that applies rate limiting.
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as key, or user ID if authenticated
		key := c.ClientIP()
		if userID, exists := c.Get(ContextUserID); exists {
			key = userID.(string)
		}

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"retry_after_seconds": int(limiter.window.Seconds()),
			})
			return
		}

		c.Header("X-RateLimit-Remaining", string(rune(limiter.Remaining(key))))
		c.Next()
	}
}

// RequestID adds a unique request ID to each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = crypto.GenerateID("req")
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Add to context for logging
		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// Logger logs request details.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		reqID, _ := c.Get("request_id")

		logger.Info("request completed",
			"request_id", reqID,
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", statusCode,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)
	}
}

// Recovery recovers from panics.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID, _ := c.Get("request_id")
				logger.Error("panic recovered",
					"request_id", reqID,
					"error", err,
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// CORS adds CORS headers.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	originsMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		originsMap[o] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		if origin != "" && (len(allowedOrigins) == 0 || originsMap[origin] || originsMap["*"]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-RateLimit-Remaining")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders adds security headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}
