package handler

import (
	"errors"
	"file-service/internal/types"
	apperrors "file-service/pkg/errors"
	"file-service/pkg/password"
	"file-service/pkg/validator"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const dummyBcryptHash = "$2a$12$dWR5CQpS4zNHLavLSIr4o.P6QDQEUJKv7mJ7WekUHHqyRSRMJzH0S"

func burnBcryptTime(passwordPlain string) {
	_ = password.Verify(passwordPlain, dummyBcryptHash)
}

type AuthHandler struct {
	userRepo       UserRepository
	db             TransactionExecutor
	s3Client       BucketCreator
	jwtService     TokenGenerator
	bucketRegion   string
	auditLogger    types.AuditLogger
	csrfMiddleware types.CSRFTokenManager
}

func NewAuthHandler(
	userRepo UserRepository,
	db TransactionExecutor,
	s3Client BucketCreator,
	jwtService TokenGenerator,
	bucketRegion string,
	auditLogger types.AuditLogger,
	csrfMiddleware types.CSRFTokenManager,
) *AuthHandler {
	return &AuthHandler{
		userRepo:       userRepo,
		db:             db,
		s3Client:       s3Client,
		jwtService:     jwtService,
		bucketRegion:   bucketRegion,
		auditLogger:    auditLogger,
		csrfMiddleware: csrfMiddleware,
	}
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	ClientID  string `json:"client_id"`
	Token     string `json:"token"`
	CSRFToken string `json:"csrf_token"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	CSRFToken string `json:"csrf_token"`
}

func (h *AuthHandler) Signup(c echo.Context) error {
	var req SignupRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validator.Email(req.Email); err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", nil, "create", err)
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if err := validator.Password(req.Password); err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", nil, "create", err)
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	passwordHash, err := password.Hash(req.Password)
	if err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", nil, "create", err)
		}
		return respondError(c, http.StatusInternalServerError, msgPasswordProcessFail)
	}

	ctx := c.Request().Context()
	userResult, clientResult, projResult, err := h.db.SignupTransaction(ctx, req.Email, passwordHash)
	if err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", nil, "create", err)
		}
		if errors.Is(err, apperrors.ErrEmailExists) {
			return respondError(c, http.StatusConflict, msgEmailAlreadyExists)
		}
		return respondError(c, http.StatusInternalServerError, msgCreateAccountFail)
	}

	if err := h.s3Client.CreateBucket(ctx, projResult.S3BucketName, h.bucketRegion); err != nil {
		c.Logger().Errorf("Failed to create S3 bucket for client %s: %v", clientResult.ID, err)
		if rollbackErr := h.db.RollbackSignup(ctx, clientResult.ID); rollbackErr != nil {
			c.Logger().Errorf("Failed to rollback signup after bucket creation failure: %v", rollbackErr)
		}
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", &userResult.ID, "create", err)
		}
		return respondError(c, http.StatusInternalServerError, msgCreateS3BucketFail)
	}

	token, err := h.jwtService.Generate(userResult.ID, userResult.ClientID, userResult.Email)
	if err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", &userResult.ID, "create", err)
		}
		return respondError(c, http.StatusInternalServerError, msgGenerateTokenFail)
	}

	// Log successful signup
	if h.auditLogger != nil {
		metadata := map[string]any{
			"email":     userResult.Email,
			"client_id": clientResult.ID.String(),
		}
		_ = h.auditLogger.LogFromContext(c, "user", &userResult.ID, "create", "success", metadata)
	}

	// Generate CSRF token
	csrfToken := ""
	if h.csrfMiddleware != nil {
		if token, err := h.csrfMiddleware.GetOrCreateToken(userResult.ID); err == nil {
			csrfToken = token
		}
	}

	return c.JSON(http.StatusCreated, SignupResponse{
		UserID:    userResult.ID.String(),
		Email:     userResult.Email,
		ClientID:  clientResult.ID.String(),
		Token:     token,
		CSRFToken: csrfToken,
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		burnBcryptTime("") // burn time even for empty creds
		if h.auditLogger != nil {
			_ = h.auditLogger.LogFromContext(c, "user", nil, "login", "denied", map[string]interface{}{
				"reason": "empty credentials",
			})
		}
		return respondError(c, http.StatusUnauthorized, msgInvalidCredentials)
	}

	u, err := h.userRepo.GetByEmail(c.Request().Context(), req.Email)
	if err != nil {
		// burn time to reduce email existence timing oracle
		burnBcryptTime(req.Password)
		if h.auditLogger != nil {
			_ = h.auditLogger.LogFromContext(c, "user", nil, "login", "denied", map[string]interface{}{
				"email":  req.Email,
				"reason": "user not found",
			})
		}
		return respondError(c, http.StatusUnauthorized, msgInvalidCredentials)
	}

	if !password.Verify(req.Password, u.PasswordHash) {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogFromContext(c, "user", &u.ID, "login", "denied", map[string]interface{}{
				"email":  req.Email,
				"reason": "invalid password",
			})
		}
		return respondError(c, http.StatusUnauthorized, msgInvalidCredentials)
	}

	token, err := h.jwtService.Generate(u.ID, u.ClientID, u.Email)
	if err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "user", &u.ID, "login", err)
		}
		return respondError(c, http.StatusInternalServerError, msgGenerateTokenFail)
	}

	// Log successful login
	if h.auditLogger != nil {
		_ = h.auditLogger.LogFromContext(c, "user", &u.ID, "login", "success", map[string]any{
			"email": u.Email,
		})
	}

	// Generate CSRF token
	csrfToken := ""
	if h.csrfMiddleware != nil {
		if token, err := h.csrfMiddleware.GetOrCreateToken(u.ID); err == nil {
			csrfToken = token
		}
	}

	return c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		CSRFToken: csrfToken,
	})
}
