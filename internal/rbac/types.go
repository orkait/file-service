package rbac

// AuthType represents the authentication method used
type AuthType string

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

// AuthSubject represents the entity performing an action
type AuthSubject struct {
	Type        AuthType
	UserRole    Role
	Permissions []Permission
}

// RoleDefinition defines a role and its privilege level
type RoleDefinition struct {
	Name  Role
	Level int
}

// PermissionMapping maps a Permission to an Action
type PermissionMapping struct {
	Permission Permission
	Action     Action
}

// APIKeyResourceScope defines which resources API keys can access
type APIKeyResourceScope struct {
	AllowedResources []Resource
}
