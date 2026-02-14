package handler

import (
	"context"
	"errors"
	"file-service/internal/auth"
	"file-service/internal/domain/client"
	"file-service/internal/domain/project"
	"file-service/internal/domain/user"
	"file-service/internal/repository"
	apperrors "file-service/pkg/errors"
	"file-service/pkg/password"
	"file-service/pkg/validator"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Pre-computed bcrypt hash (cost 12) used to equalize timing on failed lookups.
// The actual plaintext is irrelevant — this just ensures constant-time response.
const dummyBcryptHash = "$2a$12$dWR5CQpS4zNHLavLSIr4o.P6QDQEUJKv7mJ7WekUHHqyRSRMJzH0S"

type AuthHandler struct {
	userRepo repository.UserRepository
	db       interface {
		SignupTransaction(ctx context.Context, email, passwordHash string) (*user.User, *client.Client, *project.Project, error)
		RollbackSignup(ctx context.Context, clientID uuid.UUID) error
	}
	s3Client interface {
		CreateBucket(bucketName, region string) error
	}
	jwtService   *auth.JWTService
	bucketRegion string
}

func NewAuthHandler(userRepo repository.UserRepository, db interface {
	SignupTransaction(ctx context.Context, email, passwordHash string) (*user.User, *client.Client, *project.Project, error)
	RollbackSignup(ctx context.Context, clientID uuid.UUID) error
}, s3Client interface {
	CreateBucket(bucketName, region string) error
}, jwtService *auth.JWTService, bucketRegion string) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		db:           db,
		s3Client:     s3Client,
		jwtService:   jwtService,
		bucketRegion: bucketRegion,
	}
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	ClientID string `json:"client_id"`
	Token    string `json:"token"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Signup(c echo.Context) error {
	var req SignupRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validator.Email(req.Email); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if err := validator.Password(req.Password); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	passwordHash, err := password.Hash(req.Password)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgPasswordProcessFail)
	}

	ctx := c.Request().Context()
	u, client, proj, err := h.db.SignupTransaction(ctx, req.Email, passwordHash)
	if err != nil {
		if errors.Is(err, apperrors.ErrEmailExists) {
			return respondError(c, http.StatusConflict, msgEmailAlreadyExists)
		}
		return respondError(c, http.StatusInternalServerError, msgCreateAccountFail)
	}

	if err := h.s3Client.CreateBucket(proj.S3BucketName, h.bucketRegion); err != nil {
		c.Logger().Errorf("Failed to create S3 bucket for client %s: %v", client.ID, err)
		if rollbackErr := h.db.RollbackSignup(ctx, client.ID); rollbackErr != nil {
			c.Logger().Errorf("Failed to rollback signup after bucket creation failure: %v", rollbackErr)
		}
		return respondError(c, http.StatusInternalServerError, msgCreateS3BucketFail)
	}

	token, err := h.jwtService.Generate(u.ID, u.ClientID, u.Email)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgGenerateTokenFail)
	}

	return c.JSON(http.StatusCreated, SignupResponse{
		UserID:   u.ID.String(),
		Email:    u.Email,
		ClientID: client.ID.String(),
		Token:    token,
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		password.Verify("", dummyBcryptHash)
		return respondError(c, http.StatusUnauthorized, msgInvalidCredentials)
	}

	u, err := h.userRepo.GetByEmail(c.Request().Context(), req.Email)
	if err != nil {
		// Run bcrypt against a dummy hash to prevent timing oracle.
		// Without this, "user not found" returns in ~1ms while
		// "wrong password" takes ~200ms, leaking email existence.
		password.Verify(req.Password, dummyBcryptHash)
		return respondError(c, http.StatusUnauthorized, msgInvalidCredentials)
	}

	if !password.Verify(req.Password, u.PasswordHash) {
		return respondError(c, http.StatusUnauthorized, msgInvalidCredentials)
	}

	token, err := h.jwtService.Generate(u.ID, u.ClientID, u.Email)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgGenerateTokenFail)
	}

	return c.JSON(http.StatusOK, LoginResponse{
		Token: token,
	})
}
