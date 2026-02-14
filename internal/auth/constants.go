package auth

const (
	ContextKeyUserID    = "user_id"
	ContextKeyClientID  = "client_id"
	ContextKeyProjectID = "project_id"
	ContextKeyAuthType  = "auth_type"
	ContextKeyAPIKey    = "api_key"

	jsonKeyError = "error"

	headerAuthorization = "Authorization"
	headerAPIKey        = "X-API-Key"

	paramProjectID = "project_id"
	paramID        = "id"

	bearerScheme    = "bearer"
	apiKeyPrefix    = "pk_"
	authHeaderParts = 2
)

const (
	msgMissingAuthorization    = "missing authorization token"
	msgInvalidOrExpiredToken   = "invalid or expired token"
	msgMissingAPIKey           = "missing API key"
	msgInvalidAPIKey           = "invalid API key"
	msgProjectIDRequired       = "project_id required"
	msgInvalidAPIKeyFormat     = "invalid API key format"
	msgFileIDRequired          = "file_id required"
	msgInvalidFileID           = "invalid file_id"
	msgFileNotFound            = "file not found"
	msgProjectIDNotFound       = "project_id not found in request"
	msgUserNotAuthenticated    = "user not authenticated"
	msgAPIKeyRevoked           = "API key has been revoked"
	msgAPIKeyExpired           = "API key has expired"
	msgAPIKeyPermissionDenied  = "API key lacks required permission"
	msgClientNotFound          = "client not found"
	msgProjectNotFound         = "project not found"
	msgInvalidUserIDCtx        = "invalid user ID in context"
	msgInvalidClientIDCtx      = "invalid client ID in context"
	msgInvalidProjectIDCtx     = "invalid project ID in context"
	msgUnexpectedSigningMethod = "unexpected signing method: %v"
	msgTokenParseFailed        = "failed to parse token: %w"
	msgInvalidTokenClaims      = "invalid token claims"
)

type AuthType string

const (
	AuthTypeJWT    AuthType = "jwt"
	AuthTypeAPIKey AuthType = "api_key"
)
