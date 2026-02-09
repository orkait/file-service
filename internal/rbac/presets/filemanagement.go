package presets

import "file-service/internal/rbac"

const (
	RoleAdmin  rbac.Role = "admin"
	RoleEditor rbac.Role = "editor"
	RoleViewer rbac.Role = "viewer"

	PermissionRead   rbac.Permission = "read"
	PermissionWrite  rbac.Permission = "write"
	PermissionDelete rbac.Permission = "delete"

	ResourceFile   rbac.Resource = "file"
	ResourceFolder rbac.Resource = "folder"
	ResourceAPIKey rbac.Resource = "api_key"
	ResourceMember rbac.Resource = "member"

	ActionRead   rbac.Action = "read"
	ActionWrite  rbac.Action = "write"
	ActionDelete rbac.Action = "delete"
	ActionManage rbac.Action = "manage"
)

// FileManagement returns the RBAC configuration for file management service
func FileManagement() rbac.Config {
	return rbac.Config{
		Roles: []rbac.RoleDefinition{
			{Name: RoleAdmin, Level: 3},
			{Name: RoleEditor, Level: 2},
			{Name: RoleViewer, Level: 1},
		},
		Permissions: []rbac.Permission{
			PermissionRead,
			PermissionWrite,
			PermissionDelete,
		},
		Resources: []rbac.Resource{
			ResourceFile,
			ResourceFolder,
			ResourceAPIKey,
			ResourceMember,
		},
		Actions: []rbac.Action{
			ActionRead,
			ActionWrite,
			ActionDelete,
			ActionManage,
		},
		Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
			RoleAdmin: {
				ResourceFile:   {ActionRead, ActionWrite, ActionDelete},
				ResourceFolder: {ActionRead, ActionWrite, ActionDelete},
				ResourceAPIKey: {ActionRead, ActionWrite, ActionDelete, ActionManage},
				ResourceMember: {ActionRead, ActionWrite, ActionDelete, ActionManage},
			},
			RoleEditor: {
				ResourceFile:   {ActionRead, ActionWrite, ActionDelete},
				ResourceFolder: {ActionRead, ActionWrite, ActionDelete},
				ResourceAPIKey: {ActionRead, ActionWrite},
				ResourceMember: {ActionRead},
			},
			RoleViewer: {
				ResourceFile:   {ActionRead},
				ResourceFolder: {ActionRead},
				ResourceAPIKey: {ActionRead},
				ResourceMember: {ActionRead},
			},
		},
		PermissionToActionMap: []rbac.PermissionMapping{
			{Permission: PermissionRead, Action: ActionRead},
			{Permission: PermissionWrite, Action: ActionWrite},
			{Permission: PermissionDelete, Action: ActionDelete},
		},
		APIKeyScope: &rbac.APIKeyResourceScope{
			AllowedResources: []rbac.Resource{ResourceFile, ResourceFolder},
		},
	}
}
