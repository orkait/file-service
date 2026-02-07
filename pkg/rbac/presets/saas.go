package presets

import "file-management-service/pkg/rbac"

// SaaS multi-tenant roles.
const (
	SaaSRoleSuperAdmin rbac.Role = "super_admin"
	SaaSRoleAdmin      rbac.Role = "admin"
	SaaSRoleMember     rbac.Role = "member"
	SaaSRoleReadonly   rbac.Role = "readonly"
)

// SaaS permissions (API key scoped).
const (
	SaaSPermissionRead   rbac.Permission = "read"
	SaaSPermissionWrite  rbac.Permission = "write"
	SaaSPermissionDelete rbac.Permission = "delete"
)

// SaaS resources.
const (
	SaaSResourceTenant   rbac.Resource = "tenant"
	SaaSResourceUser     rbac.Resource = "user"
	SaaSResourceBilling  rbac.Resource = "billing"
	SaaSResourceSetting  rbac.Resource = "setting"
	SaaSResourceAPIKey   rbac.Resource = "api_key"
	SaaSResourceAuditLog rbac.Resource = "audit_log"
	SaaSResourceWebhook  rbac.Resource = "webhook"
)

// SaaS actions.
const (
	SaaSActionRead   rbac.Action = "read"
	SaaSActionWrite  rbac.Action = "write"
	SaaSActionDelete rbac.Action = "delete"
	SaaSActionManage rbac.Action = "manage"
)

// SaaS returns an RBAC configuration for a SaaS multi-tenant platform.
//
// Role hierarchy:
//
//	super_admin (4) — full access, tenant management, billing, audit logs
//	admin       (3) — manage users, settings, API keys, webhooks
//	member      (2) — read/write settings and webhooks, read users
//	readonly    (1) — read-only across accessible resources
//
// API keys are scoped to webhooks only (for external integrations).
func SaaS() rbac.Config {
	return rbac.Config{
		Roles: []rbac.RoleDefinition{
			{Name: SaaSRoleSuperAdmin, Level: 4},
			{Name: SaaSRoleAdmin, Level: 3},
			{Name: SaaSRoleMember, Level: 2},
			{Name: SaaSRoleReadonly, Level: 1},
		},
		Permissions: []rbac.Permission{
			SaaSPermissionRead,
			SaaSPermissionWrite,
			SaaSPermissionDelete,
		},
		Resources: []rbac.Resource{
			SaaSResourceTenant,
			SaaSResourceUser,
			SaaSResourceBilling,
			SaaSResourceSetting,
			SaaSResourceAPIKey,
			SaaSResourceAuditLog,
			SaaSResourceWebhook,
		},
		Actions: []rbac.Action{
			SaaSActionRead,
			SaaSActionWrite,
			SaaSActionDelete,
			SaaSActionManage,
		},
		Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
			SaaSRoleSuperAdmin: {
				SaaSResourceTenant:   {SaaSActionRead, SaaSActionWrite, SaaSActionDelete, SaaSActionManage},
				SaaSResourceUser:     {SaaSActionRead, SaaSActionWrite, SaaSActionDelete, SaaSActionManage},
				SaaSResourceBilling:  {SaaSActionRead, SaaSActionWrite, SaaSActionManage},
				SaaSResourceSetting:  {SaaSActionRead, SaaSActionWrite, SaaSActionManage},
				SaaSResourceAPIKey:   {SaaSActionRead, SaaSActionWrite, SaaSActionDelete, SaaSActionManage},
				SaaSResourceAuditLog: {SaaSActionRead, SaaSActionManage},
				SaaSResourceWebhook:  {SaaSActionRead, SaaSActionWrite, SaaSActionDelete, SaaSActionManage},
			},
			SaaSRoleAdmin: {
				SaaSResourceTenant:   {SaaSActionRead},
				SaaSResourceUser:     {SaaSActionRead, SaaSActionWrite, SaaSActionDelete},
				SaaSResourceBilling:  {SaaSActionRead},
				SaaSResourceSetting:  {SaaSActionRead, SaaSActionWrite},
				SaaSResourceAPIKey:   {SaaSActionRead, SaaSActionWrite, SaaSActionDelete},
				SaaSResourceAuditLog: {SaaSActionRead},
				SaaSResourceWebhook:  {SaaSActionRead, SaaSActionWrite, SaaSActionDelete},
			},
			SaaSRoleMember: {
				SaaSResourceTenant:   {SaaSActionRead},
				SaaSResourceUser:     {SaaSActionRead},
				SaaSResourceBilling:  {SaaSActionRead},
				SaaSResourceSetting:  {SaaSActionRead, SaaSActionWrite},
				SaaSResourceAPIKey:   {SaaSActionRead},
				SaaSResourceAuditLog: {SaaSActionRead},
				SaaSResourceWebhook:  {SaaSActionRead, SaaSActionWrite},
			},
			SaaSRoleReadonly: {
				SaaSResourceTenant:   {SaaSActionRead},
				SaaSResourceUser:     {SaaSActionRead},
				SaaSResourceBilling:  {SaaSActionRead},
				SaaSResourceSetting:  {SaaSActionRead},
				SaaSResourceAPIKey:   {SaaSActionRead},
				SaaSResourceAuditLog: {SaaSActionRead},
				SaaSResourceWebhook:  {SaaSActionRead},
			},
		},
		PermissionToActionMap: []rbac.PermissionMapping{
			{Permission: SaaSPermissionRead, Action: SaaSActionRead},
			{Permission: SaaSPermissionWrite, Action: SaaSActionWrite},
			{Permission: SaaSPermissionDelete, Action: SaaSActionDelete},
		},
		APIKeyScope: &rbac.APIKeyResourceScope{
			AllowedResources: []rbac.Resource{SaaSResourceWebhook},
		},
	}
}
