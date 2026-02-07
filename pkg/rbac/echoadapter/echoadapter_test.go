package echoadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"file-service/pkg/rbac"
	"file-service/pkg/rbac/echoadapter"
	"file-service/pkg/rbac/presets"

	"github.com/labstack/echo/v4"
)

func newChecker(t *testing.T) *rbac.RBACChecker {
	t.Helper()
	return rbac.MustNew(presets.FileManagement())
}

// ============================================================================
// Context Extraction Tests
// ============================================================================

func TestExtractAuthSubjectFromJWT(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "admin")

	subject, err := echoadapter.ExtractAuthSubject(c, checker)
	if err != nil {
		t.Fatalf("ExtractAuthSubject failed: %v", err)
	}

	if subject.Type != rbac.AuthTypeJWT {
		t.Errorf("Expected AuthTypeJWT, got %s", subject.Type)
	}
	if subject.UserRole != presets.RoleAdmin {
		t.Errorf("Expected RoleAdmin, got %s", subject.UserRole)
	}
}

func TestExtractAuthSubjectFromAPIKey(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set(echoadapter.ContextKeyAuthType, "api_key")
	c.Set(echoadapter.ContextKeyAPIKeyPermissions, []rbac.Permission{"read", "write"})

	subject, err := echoadapter.ExtractAuthSubject(c, checker)
	if err != nil {
		t.Fatalf("ExtractAuthSubject failed: %v", err)
	}

	if subject.Type != rbac.AuthTypeAPIKey {
		t.Errorf("Expected AuthTypeAPIKey, got %s", subject.Type)
	}
	if len(subject.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(subject.Permissions))
	}
}

func TestExtractAuthSubjectMissingContext(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := echoadapter.ExtractAuthSubject(c, checker)
	if err == nil {
		t.Error("Expected error when auth type is missing, got nil")
	}
}

func TestExtractAuthSubjectInvalidRole(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "superuser")

	_, err := echoadapter.ExtractAuthSubject(c, checker)
	if err == nil {
		t.Error("Expected error for invalid role, got nil")
	}
}

func TestSetAuthSubject(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	subject := &rbac.AuthSubject{
		Type:     rbac.AuthTypeJWT,
		UserRole: presets.RoleEditor,
	}
	echoadapter.SetAuthSubject(c, subject)

	if c.Get(echoadapter.ContextKeyAuthType) != "jwt" {
		t.Errorf("Expected auth type jwt, got %v", c.Get(echoadapter.ContextKeyAuthType))
	}
	if c.Get(echoadapter.ContextKeyUserRole) != "editor" {
		t.Errorf("Expected user role editor, got %v", c.Get(echoadapter.ContextKeyUserRole))
	}
}

func TestSetAuthSubjectNil(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Should not panic
	echoadapter.SetAuthSubject(c, nil)

	if c.Get(echoadapter.ContextKeyAuthType) != nil {
		t.Errorf("Expected nil auth type after SetAuthSubject(nil), got %v", c.Get(echoadapter.ContextKeyAuthType))
	}
}

// ============================================================================
// Middleware Tests
// ============================================================================

func TestRequireActionMiddleware(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "admin")

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequireAction(checker, presets.ResourceFile, presets.ActionRead)
	h := mw(handler)

	err := h(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequireActionMiddlewareDenied(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "viewer")

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called for unauthorized action")
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequireAction(checker, presets.ResourceFile, presets.ActionDelete)
	h := mw(handler)

	_ = h(c)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}

	// Verify sanitized error response
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err == nil {
		if body["error"] != "Forbidden" {
			t.Errorf("Expected generic 'Forbidden' error, got %q", body["error"])
		}
	}
}

func TestRequireActionMiddlewareUnauthorized(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No auth context set

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called without auth context")
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequireAction(checker, presets.ResourceFile, presets.ActionRead)
	h := mw(handler)

	_ = h(c)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// Verify sanitized error response
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err == nil {
		if body["error"] != "Unauthorized" {
			t.Errorf("Expected generic 'Unauthorized' error, got %q", body["error"])
		}
	}
}

func TestRequirePermissionMiddleware(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "api_key")
	c.Set(echoadapter.ContextKeyAPIKeyPermissions, []rbac.Permission{"read", "write"})

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequirePermission(checker, presets.ResourceFile, presets.PermissionWrite)
	h := mw(handler)

	err := h(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequirePermissionMiddlewareDenied(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "api_key")
	c.Set(echoadapter.ContextKeyAPIKeyPermissions, []rbac.Permission{"read"})

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called for missing permission")
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequirePermission(checker, presets.ResourceFile, presets.PermissionWrite)
	h := mw(handler)

	_ = h(c)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

func TestRequireRoleMiddleware(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "admin")

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequireRole(checker, presets.RoleAdmin)
	h := mw(handler)

	err := h(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequireRoleMiddlewareDenied(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "viewer")

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called for insufficient role")
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequireRole(checker, presets.RoleAdmin)
	h := mw(handler)

	_ = h(c)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

func TestRequirePermissionMiddlewareJWT(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(echoadapter.ContextKeyAuthType, "jwt")
	c.Set(echoadapter.ContextKeyUserRole, "admin")

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequirePermission(checker, presets.ResourceFile, presets.PermissionWrite)
	h := mw(handler)

	err := h(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequirePermissionMiddlewareUnknownAuthType(t *testing.T) {
	checker := newChecker(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Manually set an unknown auth type via context
	c.Set(echoadapter.ContextKeyAuthType, "magic")
	// ExtractAuthSubject will fail for unknown auth type, returning 401.

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called for unknown auth type")
		return c.String(http.StatusOK, "success")
	}

	mw := echoadapter.RequirePermission(checker, presets.ResourceFile, presets.PermissionRead)
	h := mw(handler)

	_ = h(c)

	// ExtractAuthSubject returns error for unknown auth type → 401
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unknown auth type, got %d", rec.Code)
	}
}
