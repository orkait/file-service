package rbac

// AuthType represents the authentication method used
type AuthType string

// Structural auth type constants — the checker branches on these.
const (
	AuthTypeJWT    AuthType = "jwt"
	AuthTypeAPIKey AuthType = "api_key"
)

// Role represents a user's role in the system (hierarchical)
type Role string

// Permission represents an API key permission (granular)
type Permission string

// Resource represents a type of resource in the system
type Resource string

// Action represents an operation on a resource
type Action string

// AuthSubject represents the entity performing an action (JWT user or API key)
type AuthSubject struct {
	Type        AuthType     // "jwt" or "api_key"
	UserRole    Role         // For JWT auth
	Permissions []Permission // For API key auth
}
