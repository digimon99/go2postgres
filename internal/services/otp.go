// Package services - OTP authentication support.
package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/digimon99/go2postgres/internal/database"
	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/pkg/crypto"
	"github.com/digimon99/go2postgres/pkg/email"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// OTP-related errors.
var (
	ErrOTPInvalid  = errors.New("invalid or expired OTP")
	ErrOTPRequired = errors.New("OTP required")
	ErrEmailNotConfigured = errors.New("email service not configured")
)

// OTPService handles OTP-based authentication.
type OTPService struct {
	svc         *Service
	emailClient *email.ResendClient
	otpExpiry   time.Duration
}

// NewOTPService creates a new OTP service.
func NewOTPService(svc *Service, emailClient *email.ResendClient, otpExpiry time.Duration) *OTPService {
	return &OTPService{
		svc:         svc,
		emailClient: emailClient,
		otpExpiry:   otpExpiry,
	}
}

// generateOTP creates a 6-digit OTP code.
func generateOTP() string {
	// Generate a random 6-digit number
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to time-based if crypto fails
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// SendOTP sends an OTP code to the specified email.
// Returns whether this is a new user (signup) or existing user (signin).
func (o *OTPService) SendOTP(ctx context.Context, emailAddr string) (isNewUser bool, err error) {
	logger.InfoContext(ctx, "sending OTP", "email", emailAddr)

	if o.emailClient == nil {
		return false, ErrEmailNotConfigured
	}

	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))

	// Check if user exists
	exists, err := o.svc.repo.UserExistsByEmail(ctx, emailAddr)
	if err != nil {
		return false, fmt.Errorf("checking user existence: %w", err)
	}

	isNewUser = !exists
	purpose := "signin"
	if isNewUser {
		purpose = "signup"
	}

	// Generate OTP
	code := generateOTP()

	// Store OTP
	otp := &database.OTP{
		ID:        crypto.GenerateID("otp"),
		Email:     emailAddr,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(o.otpExpiry),
		CreatedAt: time.Now(),
	}

	if err := o.svc.repo.CreateOTP(ctx, otp); err != nil {
		return false, fmt.Errorf("storing OTP: %w", err)
	}

	// Send email
	if err := o.emailClient.SendOTP(emailAddr, code, isNewUser); err != nil {
		logger.ErrorContext(ctx, "failed to send OTP email", "error", err, "email", emailAddr)
		return false, fmt.Errorf("sending email: %w", err)
	}

	logger.InfoContext(ctx, "OTP sent", "email", emailAddr, "purpose", purpose)
	return isNewUser, nil
}

// VerifyOTP verifies the OTP and returns tokens if valid.
// For new users, it creates the account first.
func (o *OTPService) VerifyOTP(ctx context.Context, emailAddr, code string) (accessToken, refreshToken string, user *models.User, isNewUser bool, err error) {
	logger.InfoContext(ctx, "verifying OTP", "email", emailAddr)

	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))

	// Check if user exists to determine purpose
	exists, err := o.svc.repo.UserExistsByEmail(ctx, emailAddr)
	if err != nil {
		return "", "", nil, false, fmt.Errorf("checking user existence: %w", err)
	}

	purpose := "signin"
	isNewUser = !exists
	if isNewUser {
		purpose = "signup"
	}

	// Verify OTP
	_, err = o.svc.repo.VerifyOTP(ctx, emailAddr, code, purpose)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return "", "", nil, false, ErrOTPInvalid
		}
		return "", "", nil, false, fmt.Errorf("verifying OTP: %w", err)
	}

	// Create user if new
	if isNewUser {
		user, err = o.createUserFromOTP(ctx, emailAddr)
		if err != nil {
			return "", "", nil, false, fmt.Errorf("creating user: %w", err)
		}
		o.svc.logAudit(ctx, &user.ID, "user.registered.otp", "user", user.ID, nil)
	} else {
		user, err = o.svc.repo.GetUserByEmail(ctx, emailAddr)
		if err != nil {
			return "", "", nil, false, fmt.Errorf("fetching user: %w", err)
		}
	}

	// Check user status
	if !user.IsActive {
		return "", "", nil, false, ErrUserInactive
	}
	if !user.IsApproved {
		return "", "", nil, false, ErrUserNotApproved
	}

	// Generate tokens
	accessToken, refreshToken, err = o.svc.jwt.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return "", "", nil, false, fmt.Errorf("generating tokens: %w", err)
	}

	// Store refresh token
	tokenHash := crypto.HashToken(refreshToken)
	rt := &models.RefreshToken{
		ID:        crypto.GenerateID("rt"),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(o.svc.jwt.GetRefreshExpiry()),
		CreatedAt: time.Now(),
	}
	if err := o.svc.repo.CreateRefreshToken(ctx, rt); err != nil {
		logger.WarnContext(ctx, "failed to store refresh token", "error", err)
	}

	o.svc.logAudit(ctx, &user.ID, "user.login.otp", "user", user.ID, nil)

	return accessToken, refreshToken, user, isNewUser, nil
}

// createUserFromOTP creates a new user account from OTP signup.
func (o *OTPService) createUserFromOTP(ctx context.Context, emailAddr string) (*models.User, error) {
	// Determine role
	role := models.RoleUser
	isApproved := false

	// Check if this is the admin email
	if strings.EqualFold(emailAddr, o.svc.cfg.AdminEmail) {
		role = models.RoleAdmin
		isApproved = true
	}

	// Auto-approve if registration is open
	if o.svc.cfg.AllowRegistration {
		isApproved = true
	}

	now := time.Now()
	user := &models.User{
		ID:         crypto.GenerateID("usr"),
		Email:      emailAddr,
		Role:       role,
		IsActive:   true,
		IsApproved: isApproved,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := o.svc.repo.CreateUserWithoutPassword(ctx, user); err != nil {
		if errors.Is(err, database.ErrDuplicateKey) {
			return nil, ErrEmailExists
		}
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves user by ID (for profile endpoint).
func (o *OTPService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return o.svc.repo.GetUserByID(ctx, userID)
}
