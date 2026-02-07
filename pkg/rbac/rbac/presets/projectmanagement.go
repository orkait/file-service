package presets

import "file-management-service/pkg/rbac"

// Project management roles.
const (
	PMRoleOwner   rbac.Role = "owner"
	PMRoleManager rbac.Role = "manager"
	PMRoleMember  rbac.Role = "member"
	PMRoleGuest   rbac.Role = "guest"
)

// Project management permissions (API key scoped).
const (
	PMPermissionRead   rbac.Permission = "read"
	PMPermissionWrite  rbac.Permission = "write"
	PMPermissionDelete rbac.Permission = "delete"
)

// Project management resources.
const (
	PMResourceProject   rbac.Resource = "project"
	PMResourceTask      rbac.Resource = "task"
	PMResourceComment   rbac.Resource = "comment"
	PMResourceMilestone rbac.Resource = "milestone"
	PMResourceMember    rbac.Resource = "member"
	PMResourceLabel     rbac.Resource = "label"
)

// Project management actions.
const (
	PMActionRead   rbac.Action = "read"
	PMActionWrite  rbac.Action = "write"
	PMActionDelete rbac.Action = "delete"
	PMActionAssign rbac.Action = "assign"
	PMActionManage rbac.Action = "manage"
)

// ProjectManagement returns an RBAC configuration for a project management tool.
//
// Role hierarchy:
//
//	owner   (4) — full access, member management, project settings
//	manager (3) — manage tasks/milestones, assign work, manage labels
//	member  (2) — create/edit tasks and comments, read milestones
//	guest   (1) — read-only across tasks and comments
//
// API keys are scoped to tasks and comments (for CI/bot integrations).
func ProjectManagement() rbac.Config {
	return rbac.Config{
		Roles: []rbac.RoleDefinition{
			{Name: PMRoleOwner, Level: 4},
			{Name: PMRoleManager, Level: 3},
			{Name: PMRoleMember, Level: 2},
			{Name: PMRoleGuest, Level: 1},
		},
		Permissions: []rbac.Permission{
			PMPermissionRead,
			PMPermissionWrite,
			PMPermissionDelete,
		},
		Resources: []rbac.Resource{
			PMResourceProject,
			PMResourceTask,
			PMResourceComment,
			PMResourceMilestone,
			PMResourceMember,
			PMResourceLabel,
		},
		Actions: []rbac.Action{
			PMActionRead,
			PMActionWrite,
			PMActionDelete,
			PMActionAssign,
			PMActionManage,
		},
		Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
			PMRoleOwner: {
				PMResourceProject:   {PMActionRead, PMActionWrite, PMActionDelete, PMActionManage},
				PMResourceTask:      {PMActionRead, PMActionWrite, PMActionDelete, PMActionAssign, PMActionManage},
				PMResourceComment:   {PMActionRead, PMActionWrite, PMActionDelete, PMActionManage},
				PMResourceMilestone: {PMActionRead, PMActionWrite, PMActionDelete, PMActionManage},
				PMResourceMember:    {PMActionRead, PMActionWrite, PMActionDelete, PMActionManage},
				PMResourceLabel:     {PMActionRead, PMActionWrite, PMActionDelete, PMActionManage},
			},
			PMRoleManager: {
				PMResourceProject:   {PMActionRead, PMActionWrite},
				PMResourceTask:      {PMActionRead, PMActionWrite, PMActionDelete, PMActionAssign},
				PMResourceComment:   {PMActionRead, PMActionWrite, PMActionDelete},
				PMResourceMilestone: {PMActionRead, PMActionWrite, PMActionDelete},
				PMResourceMember:    {PMActionRead},
				PMResourceLabel:     {PMActionRead, PMActionWrite, PMActionDelete},
			},
			PMRoleMember: {
				PMResourceProject:   {PMActionRead},
				PMResourceTask:      {PMActionRead, PMActionWrite},
				PMResourceComment:   {PMActionRead, PMActionWrite},
				PMResourceMilestone: {PMActionRead},
				PMResourceMember:    {PMActionRead},
				PMResourceLabel:     {PMActionRead},
			},
			PMRoleGuest: {
				PMResourceProject:   {PMActionRead},
				PMResourceTask:      {PMActionRead},
				PMResourceComment:   {PMActionRead},
				PMResourceMilestone: {PMActionRead},
				PMResourceMember:    {PMActionRead},
				PMResourceLabel:     {PMActionRead},
			},
		},
		PermissionToActionMap: []rbac.PermissionMapping{
			{Permission: PMPermissionRead, Action: PMActionRead},
			{Permission: PMPermissionWrite, Action: PMActionWrite},
			{Permission: PMPermissionDelete, Action: PMActionDelete},
		},
		APIKeyScope: &rbac.APIKeyResourceScope{
			AllowedResources: []rbac.Resource{PMResourceTask, PMResourceComment},
		},
	}
}
