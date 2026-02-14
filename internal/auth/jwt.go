package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	ClientID uuid.UUID `json:"client_id"`
	Email    string    `json:"email"`
	Version  int       `json:"version"` // Secret version for rotation
	jwt.RegisteredClaims
}

type JWTService struct {
	secrets map[int][]byte // version -> secret
	current int            // current version
	expiry  time.Duration
}

func NewJWTService(secret string, expiry time.Duration) *JWTService {
	// Initialize with single secret at version 1
	return &JWTService{
		secrets: map[int][]byte{
			1: []byte(secret),
		},
		current: 1,
		expiry:  expiry,
	}
}

// NewJWTServiceWithRotation creates a JWT service with multiple secret versions
// secrets map: version -> secret string
// currentVersion: the version to use for new tokens
func NewJWTServiceWithRotation(secrets map[int]string, currentVersion int, expiry time.Duration) *JWTService {
	secretBytes := make(map[int][]byte)
	for v, s := range secrets {
		secretBytes[v] = []byte(s)
	}
	return &JWTService{
		secrets: secretBytes,
		current: currentVersion,
		expiry:  expiry,
	}
}

func (s *JWTService) Generate(userID, clientID uuid.UUID, email string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		ClientID: clientID,
		Email:    email,
		Version:  s.current, // Include version in claims
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secrets[s.current])
}

func (s *JWTService) Verify(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf(msgUnexpectedSigningMethod, token.Header["alg"])
		}

		// Extract version from claims to use correct secret
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			// If no version in claims, assume version 1 (backward compatibility)
			if secret, exists := s.secrets[1]; exists {
				return secret, nil
			}
			return nil, fmt.Errorf("no secret available for token verification")
		}

		// Use the secret version from the token
		secret, exists := s.secrets[claims.Version]
		if !exists {
			return nil, fmt.Errorf("unknown secret version: %d", claims.Version)
		}

		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, fmt.Errorf(msgTokenParseFailed, err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf(msgInvalidTokenClaims)
	}

	return claims, nil
}
