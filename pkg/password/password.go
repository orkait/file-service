package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinCost is the minimum bcrypt cost (4)
	MinCost = bcrypt.MinCost
	// DefaultCost is the recommended bcrypt cost (12)
	DefaultCost = 12
	// MaxCost is the maximum bcrypt cost (31)
	MaxCost            = bcrypt.MaxCost
	errPasswordEmpty   = "password cannot be empty"
	errHashPasswordFmt = "failed to hash password: %w"
	errGetHashCostFmt  = "failed to get hash cost: %w"
)

// Hash generates a bcrypt hash of the password
func Hash(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf(errPasswordEmpty)
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf(errHashPasswordFmt, err)
	}

	return string(bytes), nil
}

// Verify checks if the password matches the hash
func Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// NeedsRehash checks if the hash needs to be rehashed with a higher cost
func NeedsRehash(hash string, cost int) (bool, error) {
	hashCost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false, fmt.Errorf(errGetHashCostFmt, err)
	}

	return hashCost < cost, nil
}
