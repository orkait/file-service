package rbac

import (
	"errors"
	"fmt"
)

// Sentinel errors for use with errors.Is().
var (
	ErrDenied            = errors.New("authorization denied")
	ErrNilSubject        = errors.New("subject is nil")
	ErrInvalidRole       = errors.New("invalid role")
	ErrInvalidPermission = errors.New("invalid permission")
)

// RBACChecker provides authorization checking based on a validated Config.
// It is thread-safe: all internal state is read-only after construction.
type RBACChecker struct {
	config Config

	// Pre-computed lookup tables (all read-only after New)
	roleIndex      map[Role]int                       // role → level
	capabilities   map[Role]map[Resource]map[Action]bool // O(1) lookup
	permToAction   map[Permission]Action
	actionToPerm   map[Action]Permission
	validPerms     map[Permission]bool
	validRoles     map[Role]bool
	apiKeyResources map[Resource]bool // nil when APIKeyScope is nil (deny all)
}

// New creates an RBACChecker from a validated Config.
// Returns an error if the config is invalid.
func New(cfg Config) (*RBACChecker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	rc := &RBACChecker{config: cfg}
	rc.buildLookups()
	return rc, nil
}

// MustNew creates an RBACChecker and panics on invalid config.
// Use this with known-good presets at init time.
func MustNew(cfg Config) *RBACChecker {
	rc, err := New(cfg)
	if err != nil {
		panic(fmt.Sprintf("rbac.MustNew: %v", err))
	}
	return rc
}

func (rc *RBACChecker) buildLookups() {
	cfg := rc.config

	// Role index
	rc.roleIndex = make(map[Role]int, len(cfg.Roles))
	rc.validRoles = make(map[Role]bool, len(cfg.Roles))
	for _, rd := range cfg.Roles {
		rc.roleIndex[rd.Name] = rd.Level
		rc.validRoles[rd.Name] = true
	}

	// Capabilities — nested map for O(1) lookup
	rc.capabilities = make(map[Role]map[Resource]map[Action]bool, len(cfg.Capabilities))
	for role, resources := range cfg.Capabilities {
		rc.capabilities[role] = make(map[Resource]map[Action]bool, len(resources))
		for res, actions := range resources {
			rc.capabilities[role][res] = make(map[Action]bool, len(actions))
			for _, act := range actions {
				rc.capabilities[role][res][act] = true
			}
		}
	}

	// Permission ↔ Action mappings
	rc.permToAction = make(map[Permission]Action, len(cfg.PermissionToActionMap))
	rc.actionToPerm = make(map[Action]Permission, len(cfg.PermissionToActionMap))
	rc.validPerms = make(map[Permission]bool, len(cfg.Permissions))
	for _, pm := range cfg.PermissionToActionMap {
		rc.permToAction[pm.Permission] = pm.Action
		rc.actionToPerm[pm.Action] = pm.Permission
	}
	for _, p := range cfg.Permissions {
		rc.validPerms[p] = true
	}

	// API key allowed resources
	if cfg.APIKeyScope != nil {
		rc.apiKeyResources = make(map[Resource]bool, len(cfg.APIKeyScope.AllowedResources))
		for _, res := range cfg.APIKeyScope.AllowedResources {
			rc.apiKeyResources[res] = true
		}
	}
}

// ---------------------------------------------------------------------------
// Authorization
// ---------------------------------------------------------------------------

// Authorize checks if the subject can perform an action on a resource.
// Returns nil if authorized, error with details if denied.
func (rc *RBACChecker) Authorize(subject *AuthSubject, resource Resource, action Action) error {
	if subject == nil {
		return fmt.Errorf("%w: %w", ErrDenied, ErrNilSubject)
	}

	switch subject.Type {
	case AuthTypeJWT:
		return rc.authorizeUserRole(subject.UserRole, resource, action)
	case AuthTypeAPIKey:
		return rc.authorizeAPIKey(subject.Permissions, resource, action)
	default:
		return fmt.Errorf("%w: unknown auth type: %s", ErrDenied, subject.Type)
	}
}

// IsAuthorized returns a boolean version of Authorize.
func (rc *RBACChecker) IsAuthorized(subject *AuthSubject, resource Resource, action Action) bool {
	return rc.Authorize(subject, resource, action) == nil
}

// RequireRole checks if the subject has at least the minimum required role.
func (rc *RBACChecker) RequireRole(subject *AuthSubject, minRole Role) error {
	if subject == nil {
		return fmt.Errorf("%w: %w", ErrDenied, ErrNilSubject)
	}
	if subject.Type != AuthTypeJWT {
		return fmt.Errorf("%w: role check requires JWT authentication", ErrDenied)
	}
	if !rc.IsRoleElevated(subject.UserRole, minRole) {
		return fmt.Errorf("%w: requires minimum role '%s', but user has role '%s'", ErrDenied, minRole, subject.UserRole)
	}
	return nil
}

func (rc *RBACChecker) authorizeUserRole(role Role, resource Resource, action Action) error {
	if role == "" {
		return fmt.Errorf("%w: user role is empty", ErrDenied)
	}
	if !rc.canRolePerformAction(role, resource, action) {
		return fmt.Errorf("%w: role '%s' cannot perform action '%s' on resource '%s'", ErrDenied, role, action, resource)
	}
	return nil
}

func (rc *RBACChecker) authorizeAPIKey(permissions []Permission, resource Resource, action Action) error {
	if rc.apiKeyResources == nil {
		return fmt.Errorf("%w: API key access is not configured", ErrDenied)
	}
	if !rc.apiKeyResources[resource] {
		return fmt.Errorf("%w: API keys cannot access resource '%s'", ErrDenied, resource)
	}

	requiredPermission := rc.ActionToPermission(action)
	if requiredPermission == "" {
		return fmt.Errorf("%w: action '%s' is not supported for API keys", ErrDenied, action)
	}

	if !rc.HasPermission(permissions, requiredPermission) {
		return fmt.Errorf("%w: API key lacks required permission '%s' for action '%s'", ErrDenied, requiredPermission, action)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Role helpers (absorbed from roles.go)
// ---------------------------------------------------------------------------

func (rc *RBACChecker) canRolePerformAction(role Role, resource Resource, action Action) bool {
	resources, ok := rc.capabilities[role]
	if !ok {
		return false
	}
	actions, ok := resources[resource]
	if !ok {
		return false
	}
	return actions[action]
}

// IsRoleElevated checks if role1 has equal or higher privilege than role2.
func (rc *RBACChecker) IsRoleElevated(role1, role2 Role) bool {
	level1, exists1 := rc.roleIndex[role1]
	level2, exists2 := rc.roleIndex[role2]
	if !exists1 || !exists2 {
		return false
	}
	return level1 >= level2
}

// ValidateRole validates a role string against configured roles.
func (rc *RBACChecker) ValidateRole(role string) (Role, error) {
	r := Role(role)
	if rc.validRoles[r] {
		return r, nil
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidRole, role)
}

// ---------------------------------------------------------------------------
// Permission helpers (absorbed from permissions.go)
// ---------------------------------------------------------------------------

// HasPermission checks if a permission array contains the required permission.
func (rc *RBACChecker) HasPermission(permissions []Permission, required Permission) bool {
	for _, perm := range permissions {
		if perm == required {
			return true
		}
	}
	return false
}

// ValidatePermissions validates an array of permissions against configured values.
func (rc *RBACChecker) ValidatePermissions(permissions []Permission) error {
	if len(permissions) == 0 {
		return fmt.Errorf("%w: permissions array cannot be empty", ErrInvalidPermission)
	}
	for _, perm := range permissions {
		if !rc.validPerms[perm] {
			return fmt.Errorf("%w: %s", ErrInvalidPermission, perm)
		}
	}
	return nil
}

// PermissionToAction maps a permission to its corresponding action.
func (rc *RBACChecker) PermissionToAction(perm Permission) Action {
	return rc.permToAction[perm]
}

// ActionToPermission maps an action to its corresponding permission.
func (rc *RBACChecker) ActionToPermission(action Action) Permission {
	return rc.actionToPerm[action]
}
