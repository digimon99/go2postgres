// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/digimon99/go2postgres/internal/api/middleware"
	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// Handler contains all HTTP handlers.
type Handler struct {
	svc    *services.Service
	otpSvc *services.OTPService
}

// NewHandler creates a new Handler.
func NewHandler(svc *services.Service, otpSvc *services.OTPService) *Handler {
	return &Handler{svc: svc, otpSvc: otpSvc}
}

// --- Auth Handlers ---

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=12"`
	FullName string `json:"full_name"`
}

// Register handles user registration.
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	user, err := h.svc.RegisterUser(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		switch err {
		case services.ErrEmailExists:
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case services.ErrWeakPassword:
			c.JSON(http.StatusBadRequest, gin.H{"error": "password does not meet requirements"})
		default:
			logger.ErrorContext(c.Request.Context(), "registration failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "registration successful",
		"user_id": user.ID,
		"email":   user.Email,
		"status":  getUserStatus(user),
	})
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	accessToken, refreshToken, user, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case services.ErrUserInactive:
			c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		case services.ErrUserNotApproved:
			c.JSON(http.StatusForbidden, gin.H{"error": "account pending approval"})
		default:
			logger.ErrorContext(c.Request.Context(), "login failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"user": gin.H{
			"user_id":   user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles token refresh.
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	accessToken, refreshToken, err := h.svc.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
	})
}

// Logout handles user logout.
func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	h.svc.Logout(c.Request.Context(), req.RefreshToken)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// --- OTP Auth Handlers ---

// SendOTPRequest represents a request to send OTP.
type SendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SendOTP sends an OTP to the user's email.
func (h *Handler) SendOTP(c *gin.Context) {
	var req SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
		return
	}

	isNewUser, err := h.otpSvc.SendOTP(c.Request.Context(), req.Email)
	if err != nil {
		switch err {
		case services.ErrEmailNotConfigured:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "email service not configured"})
		default:
			logger.ErrorContext(c.Request.Context(), "failed to send OTP", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send verification code"})
		}
		return
	}

	action := "sign in"
	if isNewUser {
		action = "sign up"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "verification code sent",
		"email":       req.Email,
		"is_new_user": isNewUser,
		"action":      action,
	})
}

// VerifyOTPRequest represents a request to verify OTP.
type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

// VerifyOTP verifies the OTP code and returns authentication tokens.
func (h *Handler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	accessToken, refreshToken, user, isNewUser, err := h.otpSvc.VerifyOTP(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		switch err {
		case services.ErrOTPInvalid:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired verification code"})
		case services.ErrUserInactive:
			c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		case services.ErrUserNotApproved:
			c.JSON(http.StatusForbidden, gin.H{"error": "account pending approval"})
		default:
			logger.ErrorContext(c.Request.Context(), "OTP verification failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"is_new_user":   isNewUser,
		"user": gin.H{
			"user_id":   user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// GetProfile returns the current user's profile.
func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)

	user, err := h.otpSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.ID,
		"email":      user.Email,
		"full_name":  user.FullName,
		"role":       user.Role,
		"is_active":  user.IsActive,
		"created_at": user.CreatedAt,
	})
}

// --- Instance Handlers ---

// CreateInstanceRequest represents an instance creation request.
type CreateInstanceRequest struct {
	ProjectID  string   `json:"project_id" binding:"required,alphanum,min=3,max=32"`
	Extensions []string `json:"extensions"`
}

// CreateInstance handles instance creation.
func (h *Handler) CreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	userID := c.GetString(middleware.ContextUserID)

	instance, password, err := h.svc.CreateInstance(c.Request.Context(), userID, req.ProjectID, req.Extensions)
	if err != nil {
		switch err {
		case services.ErrInstanceLimitReached:
			c.JSON(http.StatusForbidden, gin.H{"error": "instance limit reached"})
		default:
			logger.ErrorContext(c.Request.Context(), "instance creation failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "instance creation failed"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"instance_id":   instance.ID,
		"project_id":    instance.ProjectID,
		"database_name": instance.DatabaseName,
		"host":          instance.Host,
		"port":          instance.Port,
		"username":      instance.PostgresUser,
		"password":      password,
		"connection_string": buildConnectionString(instance, password),
		"status":        instance.Status,
		"created_at":    instance.CreatedAt,
	})
}

// ListInstances handles listing user instances.
func (h *Handler) ListInstances(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)

	instances, err := h.svc.GetUserInstances(c.Request.Context(), userID)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to list instances", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list instances"})
		return
	}

	// Convert to response format
	var result []gin.H
	for _, inst := range instances {
		result = append(result, gin.H{
			"instance_id":      inst.ID,
			"project_id":       inst.ProjectID,
			"database_name":    inst.DatabaseName,
			"host":             inst.Host,
			"port":             inst.Port,
			"username":         inst.PostgresUser,
			"status":           inst.Status,
			"disk_usage_bytes": inst.DiskUsageBytes,
			"connection_count": inst.ConnectionCount,
			"health_status":    inst.HealthStatus,
			"created_at":       inst.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": result,
		"count":     len(result),
	})
}

// GetInstance handles getting a single instance.
func (h *Handler) GetInstance(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	instance, err := h.svc.GetInstance(c.Request.Context(), userID, instanceID)
	if err != nil {
		switch err {
		case services.ErrInstanceNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get instance"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id":           instance.ID,
		"project_id":            instance.ProjectID,
		"database_name":         instance.DatabaseName,
		"host":                  instance.Host,
		"port":                  instance.Port,
		"username":              instance.PostgresUser,
		"connection_limit":      instance.ConnectionLimit,
		"statement_timeout_ms":  instance.StatementTimeoutMs,
		"status":                instance.Status,
		"disk_usage_bytes":      instance.DiskUsageBytes,
		"connection_count":      instance.ConnectionCount,
		"health_status":         instance.HealthStatus,
		"last_health_check":     instance.LastHealthCheck,
		"created_at":            instance.CreatedAt,
		"updated_at":            instance.UpdatedAt,
	})
}

// DeleteInstance handles instance deletion.
func (h *Handler) DeleteInstance(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	err := h.svc.DeleteInstance(c.Request.Context(), userID, instanceID)
	if err != nil {
		switch err {
		case services.ErrInstanceNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		default:
			logger.ErrorContext(c.Request.Context(), "instance deletion failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "deletion failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "instance deleted successfully"})
}

// RevealPassword handles password reveal requests.
func (h *Handler) RevealPassword(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	password, err := h.svc.RevealPassword(c.Request.Context(), userID, instanceID)
	if err != nil {
		switch err {
		case services.ErrInstanceNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reveal password"})
		}
		return
	}

	// Get instance for connection string
	instance, _ := h.svc.GetInstance(c.Request.Context(), userID, instanceID)

	c.JSON(http.StatusOK, gin.H{
		"password":          password,
		"connection_string": buildConnectionString(instance, password),
	})
}

// --- Admin Handlers ---

// SuspendInstance handles instance suspension (admin).
func (h *Handler) SuspendInstance(c *gin.Context) {
	adminID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	if err := h.svc.SuspendInstance(c.Request.Context(), adminID, instanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "suspension failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "instance suspended"})
}

// ResumeInstance handles instance resumption (admin).
func (h *Handler) ResumeInstance(c *gin.Context) {
	adminID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	if err := h.svc.ResumeInstance(c.Request.Context(), adminID, instanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "resumption failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "instance resumed"})
}

// ApproveUser handles user approval (admin).
func (h *Handler) ApproveUser(c *gin.Context) {
	adminID := c.GetString(middleware.ContextUserID)
	targetUserID := c.Param("id")

	if err := h.svc.ApproveUser(c.Request.Context(), adminID, targetUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "approval failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user approved"})
}

// AdminStats returns system statistics for admin dashboard.
func (h *Handler) AdminStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get counts
	userCount, _ := h.svc.CountUsers(ctx)
	instanceCount, _ := h.svc.CountInstances(ctx)

	c.JSON(http.StatusOK, gin.H{
		"total_users":     userCount,
		"total_instances": instanceCount,
	})
}

// ListAllUsers returns all users (admin).
func (h *Handler) ListAllUsers(c *gin.Context) {
	users, err := h.svc.ListUsers(c.Request.Context(), 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	var result []gin.H
	for _, u := range users {
		result = append(result, gin.H{
			"user_id":     u.ID,
			"email":       u.Email,
			"full_name":   u.FullName,
			"role":        u.Role,
			"is_active":   u.IsActive,
			"is_approved": u.IsApproved,
			"created_at":  u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": result,
		"count": len(result),
	})
}

// ListAllInstances returns all instances (admin).
func (h *Handler) ListAllInstances(c *gin.Context) {
	instances, err := h.svc.ListAllInstances(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list instances"})
		return
	}

	var result []gin.H
	for _, inst := range instances {
		result = append(result, gin.H{
			"instance_id":      inst.ID,
			"user_id":          inst.UserID,
			"project_id":       inst.ProjectID,
			"database_name":    inst.DatabaseName,
			"host":             inst.Host,
			"port":             inst.Port,
			"status":           inst.Status,
			"disk_usage_bytes": inst.DiskUsageBytes,
			"connection_count": inst.ConnectionCount,
			"health_status":    inst.HealthStatus,
			"created_at":       inst.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": result,
		"count":     len(result),
	})
}

// --- Health Handlers ---

// Health handles health check requests.
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   strconv.FormatInt(c.GetInt64("timestamp"), 10),
	})
}

// Ready handles readiness check requests.
func (h *Handler) Ready(c *gin.Context) {
	// TODO: Check database connections
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// --- Helpers ---

func buildConnectionString(inst *models.Instance, password string) string {
	if inst == nil {
		return ""
	}
	return "postgres://" + inst.PostgresUser + ":" + password + "@" +
		inst.Host + ":" + strconv.Itoa(inst.Port) + "/" + inst.DatabaseName + "?sslmode=disable"
}

func getUserStatus(user *models.User) string {
	if !user.IsActive {
		return "inactive"
	}
	if !user.IsApproved {
		return "pending_approval"
	}
	return "active"
}

// GetUserID extracts user ID from context (helper for other packages).
func GetUserID(c *gin.Context) string {
	return c.GetString(middleware.ContextUserID)
}

// IsAdmin checks if user is admin (helper for other packages).
func IsAdmin(c *gin.Context) bool {
	return strings.ToLower(c.GetString(middleware.ContextUserRole)) == models.RoleAdmin
}
