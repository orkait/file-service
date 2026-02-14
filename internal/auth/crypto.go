package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters for key derivation
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	// Default salt for API key hashing (should be overridden via config in production)
	defaultAPIKeySalt = "file-service-api-key-salt-v1"
)

// Pre-computed dummy hashes for constant-time operations
const (
	dummyShareTokenHash = "0000000000000000000000000000000000000000000000000000000000000000"
	dummyAPIKeyHash     = "0000000000000000000000000000000000000000000000000000000000000000"
)

// DummyShareTokenHash returns a dummy hash for constant-time operations
func DummyShareTokenHash() string {
	return dummyShareTokenHash
}

// DummyAPIKeyHash returns a dummy hash for constant-time operations
func DummyAPIKeyHash() string {
	return dummyAPIKeyHash
}

// ConstantTimeCompareHashes compares two hex-encoded hash strings in constant time.
// This prevents timing attacks that could leak information about valid hashes.
func ConstantTimeCompareHashes(a, b string) bool {
	aBytes := []byte(a)
	bBytes := []byte(b)

	// If lengths differ, still do comparison to maintain constant time
	if len(aBytes) != len(bBytes) {
		// Pad shorter to match longer
		if len(aBytes) < len(bBytes) {
			aBytes = make([]byte, len(bBytes))
		} else {
			bBytes = make([]byte, len(aBytes))
		}
	}

	return subtle.ConstantTimeCompare(aBytes, bBytes) == 1
}

// ConstantTimeCompareBytes compares two byte slices in constant time.
func ConstantTimeCompareBytes(a, b []byte) bool {
	if len(a) != len(b) {
		// Pad shorter to match longer
		if len(a) < len(b) {
			a = make([]byte, len(b))
		} else {
			b = make([]byte, len(a))
		}
	}

	return subtle.ConstantTimeCompare(a, b) == 1
}

// HashKey hashes a key using SHA256.
// Note: This is kept for backward compatibility. New code should use HashKeySecure.
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// HashKeySecure hashes a key using Argon2id for better security.
// This should be used for all new API keys.
func HashKeySecure(key string) string {
	return HashKeySecureWithSalt(key, []byte(defaultAPIKeySalt))
}

// HashKeySecureWithSalt hashes a key using Argon2id with a custom salt.
func HashKeySecureWithSalt(key string, salt []byte) string {
	hash := argon2.IDKey([]byte(key), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return hex.EncodeToString(hash)
}

// VerifyKeyHash verifies a key against a stored hash, supporting both old (SHA256) and new (Argon2) formats.
// This provides backward compatibility during migration.
func VerifyKeyHash(key, storedHash string) bool {
	// Try new Argon2 method first
	newHash := HashKeySecure(key)
	if ConstantTimeCompareHashes(newHash, storedHash) {
		return true
	}

	// Fall back to old SHA256 method for backward compatibility
	oldHash := HashKey(key)
	return ConstantTimeCompareHashes(oldHash, storedHash)
}

// BurnHashTime performs a dummy hash operation to equalize timing.
// Use this when you want to prevent timing oracles in authentication flows.
func BurnHashTime(input string) {
	_ = HashKey(input)
}

// ConstantTimeHashAndCompare hashes the input and compares it to the expected hash
// in constant time, with a dummy operation if the expected hash is empty.
func ConstantTimeHashAndCompare(input, expectedHash string) bool {
	actualHash := HashKey(input)

	if expectedHash == "" {
		// Burn time even if no expected hash
		ConstantTimeCompareHashes(actualHash, dummyAPIKeyHash)
		return false
	}

	return ConstantTimeCompareHashes(actualHash, expectedHash)
}
