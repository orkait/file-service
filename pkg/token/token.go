package token

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	apiKeyPrefix              = "pk_"
	apiKeyByteLength          = 20
	shareTokenBytes           = 32
	errGenerateRandomBytesFmt = "failed to generate random bytes: %w"
	errLengthPositiveFmt      = "length must be positive"
	errByteLengthPositiveFmt  = "byteLength must be positive"
)

func Generate(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf(errLengthPositiveFmt)
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf(errGenerateRandomBytesFmt, err)
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func GenerateHex(byteLength int) (string, error) {
	if byteLength <= 0 {
		return "", fmt.Errorf(errByteLengthPositiveFmt)
	}

	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf(errGenerateRandomBytesFmt, err)
	}

	return hex.EncodeToString(bytes), nil
}

func GenerateAPIKey() (string, error) {
	bytes := make([]byte, apiKeyByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf(errGenerateRandomBytesFmt, err)
	}
	return apiKeyPrefix + hex.EncodeToString(bytes), nil
}

func ExtractPrefix(token string, length int) string {
	if len(token) < length {
		return token
	}
	return token[:length]
}

func GenerateShareToken() (string, error) {
	return GenerateHex(shareTokenBytes)
}
