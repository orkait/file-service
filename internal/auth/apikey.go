package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"file-service/internal/domain/apikey"
	"file-service/internal/repository"
	apperrors "file-service/pkg/errors"
)

type APIKeyService struct {
	repo repository.APIKeyRepository
}

func NewAPIKeyService(repo repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

func (s *APIKeyService) ValidatePermissions(key *apikey.APIKey, required apikey.Permission) error {
	if !key.IsActive() {
		if key.RevokedAt != nil {
			return apperrors.Revoked(msgAPIKeyRevoked)
		}
		return apperrors.Expired(msgAPIKeyExpired)
	}

	if !key.HasPermission(required) {
		return apperrors.Forbidden(msgAPIKeyPermissionDenied)
	}

	return nil
}

func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
