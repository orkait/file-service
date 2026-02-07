package rbac_test

import (
	"errors"
	"testing"

	"file-service/pkg/rbac"
	"file-service/pkg/rbac/presets"
)

func newChecker(t *testing.T) *rbac.RBACChecker {
	t.Helper()
	rc, err := rbac.New(presets.FileManagement())
	if err != nil {
		t.Fatalf("failed to create checker: %v", err)
	}
	return rc
}

// ============================================================================
// Role Hierarchy Tests
// ============================================================================

func TestIsRoleElevated(t *testing.T) {
	checker := newChecker(t)

	tests := []struct {
		name     string
		role1    rbac.Role
		role2    rbac.Role
		expected bool
	}{
		{"Admin >= Admin", presets.RoleAdmin, presets.RoleAdmin, true},
		{"Admin >= Editor", presets.RoleAdmin, presets.RoleEditor, true},
		{"Admin >= Viewer", presets.RoleAdmin, presets.RoleViewer, true},
		{"Editor >= Editor", presets.RoleEditor, presets.RoleEditor, true},
		{"Editor >= Viewer", presets.RoleEditor, presets.RoleViewer, true},
		{"Editor < Admin", presets.RoleEditor, presets.RoleAdmin, false},
		{"Viewer >= Viewer", presets.RoleViewer, presets.RoleViewer, true},
		{"Viewer < Editor", presets.RoleViewer, presets.RoleEditor, false},
		{"Viewer < Admin", presets.RoleViewer, presets.RoleAdmin, false},
		{"Invalid role1", rbac.Role("invalid"), presets.RoleAdmin, false},
		{"Invalid role2", presets.RoleAdmin, rbac.Role("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsRoleElevated(tt.role1, tt.role2)
			if result != tt.expected {
				t.Errorf("IsRoleElevated(%s, %s) = %v, expected %v", tt.role1, tt.role2, result, tt.expected)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	checker := newChecker(t)

	tests := []struct {
		name      string
		role      string
		expected  rbac.Role
		shouldErr bool
	}{
		{"Valid admin", "admin", presets.RoleAdmin, false},
		{"Valid editor", "editor", presets.RoleEditor, false},
		{"Valid viewer", "viewer", presets.RoleViewer, false},
		{"Invalid role", "superuser", "", true},
		{"Empty role", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checker.ValidateRole(tt.role)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("ValidateRole(%s) expected error, got nil", tt.role)
				}
				if !errors.Is(err, rbac.ErrInvalidRole) {
					t.Errorf("ValidateRole(%s) error should wrap ErrInvalidRole, got: %v", tt.role, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRole(%s) unexpected error: %v", tt.role, err)
				}
				if result != tt.expected {
					t.Errorf("ValidateRole(%s) = %s, expected %s", tt.role, result, tt.expected)
				}
			}
		})
	}
}

// ============================================================================
// Permission Tests
// ============================================================================

func TestHasPermission(t *testing.T) {
	checker := newChecker(t)

	tests := []struct {
		name        string
		permissions []rbac.Permission
		required    rbac.Permission
		expected    bool
	}{
		{"Has read permission", []rbac.Permission{"read", "write"}, presets.PermissionRead, true},
		{"Has write permission", []rbac.Permission{"read", "write"}, presets.PermissionWrite, true},
		{"Lacks delete permission", []rbac.Permission{"read", "write"}, presets.PermissionDelete, false},
		{"Empty permissions", []rbac.Permission{}, presets.PermissionRead, false},
		{"Case sensitive check", []rbac.Permission{"Read"}, presets.PermissionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.HasPermission(tt.permissions, tt.required)
			if result != tt.expected {
				t.Errorf("HasPermission(%v, %s) = %v, expected %v", tt.permissions, tt.required, result, tt.expected)
			}
		})
	}
}

func TestValidatePermissions(t *testing.T) {
	checker := newChecker(t)

	tests := []struct {
		name        string
		permissions []rbac.Permission
		shouldErr   bool
	}{
		{"Valid permissions", []rbac.Permission{"read", "write"}, false},
		{"Valid all permissions", []rbac.Permission{"read", "write", "delete"}, false},
		{"Invalid permission", []rbac.Permission{"read", "execute"}, true},
		{"Empty array", []rbac.Permission{}, true},
		{"Mixed valid/invalid", []rbac.Permission{"read", "invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.ValidatePermissions(tt.permissions)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("ValidatePermissions(%v) expected error, got nil", tt.permissions)
				}
				if !errors.Is(err, rbac.ErrInvalidPermission) {
					t.Errorf("ValidatePermissions(%v) error should wrap ErrInvalidPermission, got: %v", tt.permissions, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePermissions(%v) unexpected error: %v", tt.permissions, err)
				}
			}
		})
	}
}

func TestPermissionActionMapping(t *testing.T) {
	checker := newChecker(t)

	t.Run("PermissionToAction", func(t *testing.T) {
		if a := checker.PermissionToAction(presets.PermissionRead); a != presets.ActionRead {
			t.Errorf("expected %s, got %s", presets.ActionRead, a)
		}
		if a := checker.PermissionToAction(presets.PermissionWrite); a != presets.ActionWrite {
			t.Errorf("expected %s, got %s", presets.ActionWrite, a)
		}
		if a := checker.PermissionToAction(presets.PermissionDelete); a != presets.ActionDelete {
			t.Errorf("expected %s, got %s", presets.ActionDelete, a)
		}
		if a := checker.PermissionToAction(rbac.Permission("unknown")); a != "" {
			t.Errorf("expected empty action, got %s", a)
		}
	})

	t.Run("ActionToPermission", func(t *testing.T) {
		if p := checker.ActionToPermission(presets.ActionRead); p != presets.PermissionRead {
			t.Errorf("expected %s, got %s", presets.PermissionRead, p)
		}
		if p := checker.ActionToPermission(presets.ActionManage); p != "" {
			t.Errorf("expected empty permission for manage, got %s", p)
		}
	})
}

// ============================================================================
// Authorization Tests
// ============================================================================

func TestAuthorizeAdminUser(t *testing.T) {
	checker := newChecker(t)
	subject := &rbac.AuthSubject{
		Type:     rbac.AuthTypeJWT,
		UserRole: presets.RoleAdmin,
	}

	tests := []struct {
		name      string
		resource  rbac.Resource
		action    rbac.Action
		shouldErr bool
	}{
		{"Admin can read files", presets.ResourceFile, presets.ActionRead, false},
		{"Admin can write files", presets.ResourceFile, presets.ActionWrite, false},
		{"Admin can delete files", presets.ResourceFile, presets.ActionDelete, false},
		{"Admin can manage API keys", presets.ResourceAPIKey, presets.ActionManage, false},
		{"Admin can manage members", presets.ResourceMember, presets.ActionManage, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Authorize(subject, tt.resource, tt.action)
			if tt.shouldErr && err == nil {
				t.Errorf("Authorize expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Authorize unexpected error: %v", err)
			}
		})
	}
}

func TestAuthorizeEditorUser(t *testing.T) {
	checker := newChecker(t)
	subject := &rbac.AuthSubject{
		Type:     rbac.AuthTypeJWT,
		UserRole: presets.RoleEditor,
	}

	tests := []struct {
		name      string
		resource  rbac.Resource
		action    rbac.Action
		shouldErr bool
	}{
		{"Editor can read files", presets.ResourceFile, presets.ActionRead, false},
		{"Editor can write files", presets.ResourceFile, presets.ActionWrite, false},
		{"Editor can write API keys", presets.ResourceAPIKey, presets.ActionWrite, false},
		{"Editor cannot manage API keys", presets.ResourceAPIKey, presets.ActionManage, true},
		{"Editor cannot manage members", presets.ResourceMember, presets.ActionManage, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Authorize(subject, tt.resource, tt.action)
			if tt.shouldErr && err == nil {
				t.Errorf("Authorize expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Authorize unexpected error: %v", err)
			}
		})
	}
}

func TestDenyViewerDelete(t *testing.T) {
	checker := newChecker(t)
	subject := &rbac.AuthSubject{
		Type:     rbac.AuthTypeJWT,
		UserRole: presets.RoleViewer,
	}

	err := checker.Authorize(subject, presets.ResourceFile, presets.ActionDelete)
	if err == nil {
		t.Error("Expected viewer to be denied delete action")
	}
	if !errors.Is(err, rbac.ErrDenied) {
		t.Errorf("Expected ErrDenied, got: %v", err)
	}
}

func TestAuthorizeAPIKeyWithPermission(t *testing.T) {
	checker := newChecker(t)
	subject := &rbac.AuthSubject{
		Type:        rbac.AuthTypeAPIKey,
		Permissions: []rbac.Permission{"read", "write"},
	}

	tests := []struct {
		name      string
		resource  rbac.Resource
		action    rbac.Action
		shouldErr bool
	}{
		{"API key can read files", presets.ResourceFile, presets.ActionRead, false},
		{"API key can write files", presets.ResourceFile, presets.ActionWrite, false},
		{"API key cannot delete files", presets.ResourceFile, presets.ActionDelete, true},
		{"API key cannot access API keys", presets.ResourceAPIKey, presets.ActionRead, true},
		{"API key cannot access members", presets.ResourceMember, presets.ActionRead, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Authorize(subject, tt.resource, tt.action)
			if tt.shouldErr && err == nil {
				t.Errorf("Authorize expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Authorize unexpected error: %v", err)
			}
		})
	}
}

func TestDenyAPIKeyWithoutPermission(t *testing.T) {
	checker := newChecker(t)
	subject := &rbac.AuthSubject{
		Type:        rbac.AuthTypeAPIKey,
		Permissions: []rbac.Permission{"read"},
	}

	err := checker.Authorize(subject, presets.ResourceFile, presets.ActionWrite)
	if err == nil {
		t.Error("Expected API key without write permission to be denied")
	}
	if !errors.Is(err, rbac.ErrDenied) {
		t.Errorf("Expected ErrDenied, got: %v", err)
	}
}

func TestAuthorizeNilSubject(t *testing.T) {
	checker := newChecker(t)

	err := checker.Authorize(nil, presets.ResourceFile, presets.ActionRead)
	if err == nil {
		t.Error("Expected error for nil subject")
	}
	if !errors.Is(err, rbac.ErrNilSubject) {
		t.Errorf("Expected ErrNilSubject, got: %v", err)
	}
	if !errors.Is(err, rbac.ErrDenied) {
		t.Errorf("Expected ErrDenied, got: %v", err)
	}
}

func TestAuthorizeUnknownAuthType(t *testing.T) {
	checker := newChecker(t)
	subject := &rbac.AuthSubject{Type: rbac.AuthType("magic")}

	err := checker.Authorize(subject, presets.ResourceFile, presets.ActionRead)
	if err == nil {
		t.Error("Expected error for unknown auth type")
	}
	if !errors.Is(err, rbac.ErrDenied) {
		t.Errorf("Expected ErrDenied, got: %v", err)
	}
}

func TestRequireRole(t *testing.T) {
	checker := newChecker(t)

	tests := []struct {
		name      string
		subject   *rbac.AuthSubject
		minRole   rbac.Role
		shouldErr bool
	}{
		{
			"Admin meets admin requirement",
			&rbac.AuthSubject{Type: rbac.AuthTypeJWT, UserRole: presets.RoleAdmin},
			presets.RoleAdmin, false,
		},
		{
			"Editor meets viewer requirement",
			&rbac.AuthSubject{Type: rbac.AuthTypeJWT, UserRole: presets.RoleEditor},
			presets.RoleViewer, false,
		},
		{
			"Viewer does not meet admin requirement",
			&rbac.AuthSubject{Type: rbac.AuthTypeJWT, UserRole: presets.RoleViewer},
			presets.RoleAdmin, true,
		},
		{
			"API key auth fails role check",
			&rbac.AuthSubject{Type: rbac.AuthTypeAPIKey, Permissions: []rbac.Permission{"read"}},
			presets.RoleViewer, true,
		},
		{
			"Nil subject fails",
			nil, presets.RoleViewer, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.RequireRole(tt.subject, tt.minRole)
			if tt.shouldErr && err == nil {
				t.Errorf("RequireRole expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("RequireRole unexpected error: %v", err)
			}
			if tt.shouldErr && err != nil && !errors.Is(err, rbac.ErrDenied) {
				t.Errorf("RequireRole error should wrap ErrDenied, got: %v", err)
			}
		})
	}
}

func TestMustNewPanicsOnInvalidConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNew should panic on invalid config")
		}
	}()
	rbac.MustNew(rbac.Config{})
}

func TestMustNewSucceedsWithPreset(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustNew should not panic with valid preset: %v", r)
		}
	}()
	rbac.MustNew(presets.FileManagement())
}
