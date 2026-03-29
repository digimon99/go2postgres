// Package auth provides JWT authentication.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token types.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Common errors.
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents JWT claims.
type Claims struct {
	UserID    string `json:"sub"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations.
type JWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(secret string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        "go2postgres",
	}
}

// GenerateTokenPair creates both access and refresh tokens.
func (m *JWTManager) GenerateTokenPair(userID, email, role string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.generateToken(userID, email, role, TokenTypeAccess, m.accessExpiry)
	if err != nil {
		return "", "", fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, err = m.generateToken(userID, email, role, TokenTypeRefresh, m.refreshExpiry)
	if err != nil {
		return "", "", fmt.Errorf("generating refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// GenerateAccessToken creates only an access token.
func (m *JWTManager) GenerateAccessToken(userID, email, role string) (string, error) {
	return m.generateToken(userID, email, role, TokenTypeAccess, m.accessExpiry)
}

// generateToken creates a JWT token.
func (m *JWTManager) generateToken(userID, email, role, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ValidateToken validates a JWT token and returns the claims.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAccessToken validates an access token specifically.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeAccess {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically.
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetAccessExpiry returns the access token expiry duration.
func (m *JWTManager) GetAccessExpiry() time.Duration {
	return m.accessExpiry
}

// GetRefreshExpiry returns the refresh token expiry duration.
func (m *JWTManager) GetRefreshExpiry() time.Duration {
	return m.refreshExpiry
}
