package rbac_test

import (
	"strings"
	"testing"

	"file-service/pkg/rbac"
	"file-service/pkg/rbac/presets"
)

func validBaseConfig() rbac.Config {
	return rbac.Config{
		Roles:       []rbac.RoleDefinition{{Name: "admin", Level: 2}, {Name: "user", Level: 1}},
		Permissions: []rbac.Permission{"read", "write"},
		Resources:   []rbac.Resource{"file"},
		Actions:     []rbac.Action{"read", "write"},
		Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
			"admin": {"file": {"read", "write"}},
			"user":  {"file": {"read"}},
		},
		PermissionToActionMap: []rbac.PermissionMapping{
			{Permission: "read", Action: "read"},
			{Permission: "write", Action: "write"},
		},
	}
}

func TestValidatePreset(t *testing.T) {
	cfg := presets.FileManagement()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("FileManagement preset should be valid: %v", err)
	}
}

func TestValidateEmptyConfig(t *testing.T) {
	cfg := rbac.Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("empty config should fail validation")
	}
}

func TestValidateEmptyRoles(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Roles = nil
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "Roles") {
		t.Fatalf("expected Roles error, got: %v", err)
	}
}

func TestValidateEmptyPermissions(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Permissions = nil
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "Permissions") {
		t.Fatalf("expected Permissions error, got: %v", err)
	}
}

func TestValidateEmptyResources(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Resources = nil
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "Resources") {
		t.Fatalf("expected Resources error, got: %v", err)
	}
}

func TestValidateEmptyActions(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Actions = nil
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "Actions") {
		t.Fatalf("expected Actions error, got: %v", err)
	}
}

func TestValidateDuplicateRoleNames(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Roles = []rbac.RoleDefinition{{Name: "admin", Level: 2}, {Name: "admin", Level: 1}}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate role name") {
		t.Fatalf("expected duplicate role name error, got: %v", err)
	}
}

func TestValidateDuplicateRoleLevels(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Roles = []rbac.RoleDefinition{{Name: "admin", Level: 1}, {Name: "user", Level: 1}}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate role level") {
		t.Fatalf("expected duplicate role level error, got: %v", err)
	}
}

func TestValidateEmptyRoleName(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Roles = []rbac.RoleDefinition{{Name: "", Level: 1}}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "role name must not be empty") {
		t.Fatalf("expected empty role name error, got: %v", err)
	}
}

func TestValidateCapabilityReferencesUnknownRole(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Capabilities["ghost"] = map[rbac.Resource][]rbac.Action{
		"file": {"read"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown role") {
		t.Fatalf("expected unknown role error, got: %v", err)
	}
}

func TestValidateCapabilityReferencesUnknownResource(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Capabilities["admin"]["database"] = []rbac.Action{"read"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown resource") {
		t.Fatalf("expected unknown resource error, got: %v", err)
	}
}

func TestValidateCapabilityReferencesUnknownAction(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Capabilities["admin"]["file"] = []rbac.Action{"read", "fly"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("expected unknown action error, got: %v", err)
	}
}

func TestValidatePermissionMappingUnknownPermission(t *testing.T) {
	cfg := validBaseConfig()
	cfg.PermissionToActionMap = append(cfg.PermissionToActionMap,
		rbac.PermissionMapping{Permission: "execute", Action: "read"},
	)
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown permission") {
		t.Fatalf("expected unknown permission error, got: %v", err)
	}
}

func TestValidatePermissionMappingUnknownAction(t *testing.T) {
	cfg := validBaseConfig()
	cfg.PermissionToActionMap = append(cfg.PermissionToActionMap,
		rbac.PermissionMapping{Permission: "read", Action: "fly"},
	)
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("expected unknown action error, got: %v", err)
	}
}

func TestValidateAPIKeyScopeUnknownResource(t *testing.T) {
	cfg := validBaseConfig()
	cfg.APIKeyScope = &rbac.APIKeyResourceScope{
		AllowedResources: []rbac.Resource{"database"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown resource") {
		t.Fatalf("expected unknown resource error, got: %v", err)
	}
}

func TestValidateNilAPIKeyScopeIsAllowed(t *testing.T) {
	cfg := validBaseConfig()
	cfg.APIKeyScope = nil
	if err := cfg.Validate(); err != nil {
		t.Fatalf("nil APIKeyScope should be valid: %v", err)
	}
}

func TestNewWithCustomConfig(t *testing.T) {
	cfg := validBaseConfig()
	checker, err := rbac.New(cfg)
	if err != nil {
		t.Fatalf("New with valid custom config failed: %v", err)
	}

	// Verify the custom config works
	subject := &rbac.AuthSubject{Type: rbac.AuthTypeJWT, UserRole: "admin"}
	if err := checker.Authorize(subject, "file", "write"); err != nil {
		t.Errorf("admin should be able to write file: %v", err)
	}

	subject2 := &rbac.AuthSubject{Type: rbac.AuthTypeJWT, UserRole: "user"}
	if err := checker.Authorize(subject2, "file", "write"); err == nil {
		t.Error("user should not be able to write file")
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	_, err := rbac.New(rbac.Config{})
	if err == nil {
		t.Fatal("New should reject empty config")
	}
}

func TestNilAPIKeyScopeDeniesAllAPIKeyAccess(t *testing.T) {
	cfg := validBaseConfig()
	cfg.APIKeyScope = nil
	checker, err := rbac.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	subject := &rbac.AuthSubject{
		Type:        rbac.AuthTypeAPIKey,
		Permissions: []rbac.Permission{"read"},
	}
	if err := checker.Authorize(subject, "file", "read"); err == nil {
		t.Error("API key access should be denied when APIKeyScope is nil")
	}
}

func TestValidateDuplicatePermissionInMapping(t *testing.T) {
	cfg := validBaseConfig()
	cfg.PermissionToActionMap = []rbac.PermissionMapping{
		{Permission: "read", Action: "read"},
		{Permission: "read", Action: "write"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate permission in mapping") {
		t.Fatalf("expected duplicate permission in mapping error, got: %v", err)
	}
}

func TestValidateDuplicateActionInMapping(t *testing.T) {
	cfg := validBaseConfig()
	cfg.PermissionToActionMap = []rbac.PermissionMapping{
		{Permission: "read", Action: "read"},
		{Permission: "write", Action: "read"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate action in mapping") {
		t.Fatalf("expected duplicate action in mapping error, got: %v", err)
	}
}

func TestDuplicatePermission(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Permissions = []rbac.Permission{"read", "read"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate permission") {
		t.Fatalf("expected duplicate permission error, got: %v", err)
	}
}

func TestDuplicateResource(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Resources = []rbac.Resource{"file", "file"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate resource") {
		t.Fatalf("expected duplicate resource error, got: %v", err)
	}
}

func TestDuplicateAction(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Actions = []rbac.Action{"read", "read"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate action") {
		t.Fatalf("expected duplicate action error, got: %v", err)
	}
}
