// Package crypto provides encryption and hashing utilities.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCiphertext indicates decryption failed.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// Encryptor handles AES-256-GCM encryption/decryption.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates a new Encryptor with the given 32-byte key.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext and nonce.
func (e *Encryptor) Encrypt(plaintext string) (ciphertext, nonce string, err error) {
	nonceBytes := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return "", "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertextBytes := e.gcm.Seal(nil, nonceBytes, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(ciphertextBytes),
		base64.StdEncoding.EncodeToString(nonceBytes),
		nil
}

// Decrypt decrypts base64-encoded ciphertext using the provided nonce.
func (e *Encryptor) Decrypt(ciphertext, nonce string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	if len(nonceBytes) != e.gcm.NonceSize() {
		return "", ErrInvalidCiphertext
	}

	plaintext, err := e.gcm.Open(nil, nonceBytes, ciphertextBytes, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	return string(plaintext), nil
}

// HashPassword hashes a password using bcrypt with the given cost.
func HashPassword(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// HashToken creates a SHA-256 hash of a token (for refresh token storage).
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// GenerateSecurePassword generates a cryptographically secure random password.
func GenerateSecurePassword(length int) (string, error) {
	if length < 16 {
		length = 16
	}

	// Character sets for password generation
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?"
		allChars     = lowerChars + upperChars + digitChars + specialChars
	)

	// Ensure at least one character from each category
	password := make([]byte, length)
	
	// Use crypto/rand for secure random bytes
	randomBytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	// First 4 characters ensure one from each category
	password[0] = lowerChars[int(randomBytes[0])%len(lowerChars)]
	password[1] = upperChars[int(randomBytes[1])%len(upperChars)]
	password[2] = digitChars[int(randomBytes[2])%len(digitChars)]
	password[3] = specialChars[int(randomBytes[3])%len(specialChars)]

	// Fill remaining characters
	for i := 4; i < length; i++ {
		password[i] = allChars[int(randomBytes[i])%len(allChars)]
	}

	// Shuffle the password using Fisher-Yates
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	for i := len(password) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// GenerateID generates a unique ID with an optional prefix.
func GenerateID(prefix string) string {
	bytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		// Fallback to time-based if crypto/rand fails
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	
	id := base64.RawURLEncoding.EncodeToString(bytes)
	if prefix != "" {
		return prefix + "_" + id
	}
	return id
}

// ValidatePasswordStrength checks password meets minimum requirements.
func ValidatePasswordStrength(password string, minLength int) error {
	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()-_=+[]{}|;:,.<>?", c):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return errors.New("password must contain uppercase, lowercase, digit, and special character")
	}

	return nil
}
