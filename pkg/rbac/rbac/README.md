# RBAC Package

Configurable Role-Based Access Control for Go services. Zero hardcoded defaults in core — all roles, permissions, and capabilities are injected via `Config`.

## Package Structure

```
pkg/rbac/
├── types.go          # Pure type definitions
├── config.go         # Config struct + Validate()
├── checker.go        # RBACChecker — accepts Config, all auth logic
├── rbac_test.go      # Core tests (zero Echo dependency)
├── config_test.go    # Config validation tests
├── presets/
│   ├── filemanagement.go       # Orka file service preset
│   └── filemanagement_test.go  # Preset tests
└── echoadapter/
    ├── context.go              # Echo context extraction
    ├── middleware.go            # Echo middleware
    └── echoadapter_test.go     # Echo-specific tests
```

**Dependency graph (no circular imports):**

```
presets/      → imports rbac/
echoadapter/  → imports rbac/
rbac/ (core)  → imports nothing (pure Go + fmt)
```

## Quick Start

### Using a Preset

```go
import (
    "file-management-service/pkg/rbac"
    "file-management-service/pkg/rbac/presets"
    "file-management-service/pkg/rbac/echoadapter"
)

// Create checker from the file management preset
checker := rbac.MustNew(presets.FileManagement())

// Use Echo middleware
e.POST("/upload", handler,
    echoadapter.RequirePermission(checker, presets.PermissionWrite),
)
```

### Bring Your Own Config

```go
checker, err := rbac.New(rbac.Config{
    Roles: []rbac.RoleDefinition{
        {Name: "superadmin", Level: 2},
        {Name: "user", Level: 1},
    },
    Permissions: []rbac.Permission{"read", "write"},
    Resources:   []rbac.Resource{"document"},
    Actions:     []rbac.Action{"read", "write"},
    Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
        "superadmin": {"document": {"read", "write"}},
        "user":       {"document": {"read"}},
    },
    PermissionToActionMap: []rbac.PermissionMapping{
        {Permission: "read", Action: "read"},
        {Permission: "write", Action: "write"},
    },
})
```

## Core API

### RBACChecker Methods

```go
// Construction
func New(cfg Config) (*RBACChecker, error)
func MustNew(cfg Config) *RBACChecker  // panics on invalid config

// Authorization
func (rc *RBACChecker) Authorize(subject *AuthSubject, resource Resource, action Action) error
func (rc *RBACChecker) IsAuthorized(subject *AuthSubject, resource Resource, action Action) bool
func (rc *RBACChecker) RequireRole(subject *AuthSubject, minRole Role) error

// Role helpers
func (rc *RBACChecker) IsRoleElevated(role1, role2 Role) bool
func (rc *RBACChecker) ValidateRole(role string) (Role, error)

// Permission helpers
func (rc *RBACChecker) HasPermission(permissions []string, required Permission) bool
func (rc *RBACChecker) ValidatePermissions(permissions []string) error
func (rc *RBACChecker) PermissionToAction(perm Permission) Action
func (rc *RBACChecker) ActionToPermission(action Action) Permission
```

### Echo Adapter

```go
import "file-management-service/pkg/rbac/echoadapter"

// Middleware
func RequireAction(checker, resource, action) echo.MiddlewareFunc
func RequireRole(checker, minRole) echo.MiddlewareFunc
func RequirePermission(checker, permission) echo.MiddlewareFunc

// Context helpers
func ExtractAuthSubject(c echo.Context, checker *rbac.RBACChecker) (*rbac.AuthSubject, error)
func SetAuthSubject(c echo.Context, subject *rbac.AuthSubject)
```

### Context Requirements

The Echo adapter reads these context keys (set by your auth middleware):

```go
// JWT auth
c.Set("auth_type", "jwt")
c.Set("user_role", "admin")

// API key auth
c.Set("auth_type", "api_key")
c.Set("api_key_permissions", []string{"read", "write"})
```

## File Management Preset

The `presets.FileManagement()` config defines:

| Role   | Files (R/W/D) | Folders (R/W/D) | API Keys        | Members         |
|--------|---------------|-----------------|-----------------|-----------------|
| admin  | R, W, D       | R, W, D         | R, W, D, Manage | R, W, D, Manage |
| editor | R, W, D       | R, W, D         | R, W            | R               |
| viewer | R             | R               | R               | R               |

API keys are scoped to **files and folders only**.

## Testing

```bash
# Core + config tests
go test ./pkg/rbac/... -v -count=1

# Preset tests
go test ./pkg/rbac/presets/... -v -count=1

# Echo adapter tests
go test ./pkg/rbac/echoadapter/... -v -count=1

# All packages
go test ./pkg/rbac/... -v
```
