// Package apikey provides generation and validation utilities for API keys.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	prefix    = "g2p_"
	rawBytes  = 32
	previewLen = 16 // chars of full key to store as preview (includes prefix)
)

// base58 alphabet (Bitcoin-style, avoids 0/O/I/l).
const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// encodeBase58 encodes a byte slice to a base58 string.
func encodeBase58(b []byte) string {
	var result []byte
	x := new([32]byte)
	copy(x[:], b)

	// Count leading zeros
	leading := 0
	for _, v := range b {
		if v != 0 {
			break
		}
		leading++
	}

	// Simple big-number base58 encoding
	num := make([]byte, len(b))
	copy(num, b)

	for len(num) > 0 {
		rem := 0
		var next []byte
		for _, v := range num {
			cur := rem*256 + int(v)
			if len(next) > 0 || cur/58 > 0 {
				next = append(next, byte(cur/58))
			}
			rem = cur % 58
		}
		result = append(result, alphabet[rem])
		num = next
	}

	for i := 0; i < leading; i++ {
		result = append(result, alphabet[0])
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	_ = x
	return string(result)
}

// Generate creates a new random API key in format g2p_<base58(32 random bytes)>.
// Returns the full plaintext key (shown once) and any error.
func Generate() (string, error) {
	raw := make([]byte, rawBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + encodeBase58(raw), nil
}

// Hash returns the hex-encoded SHA-256 hash of the key (what is stored in DB).
func Hash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Preview returns the first previewLen characters of the key for display.
func Preview(key string) string {
	if len(key) <= previewLen {
		return key
	}
	return key[:previewLen] + "..."
}

// IsValidFormat returns true if key looks like a g2p_ prefixed key.
func IsValidFormat(key string) bool {
	return strings.HasPrefix(key, prefix) && len(key) > len(prefix)+10
}
