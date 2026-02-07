package presets

import "file-management-service/pkg/rbac"

// CMS roles.
const (
	CMSRoleAdmin      rbac.Role = "admin"
	CMSRoleEditor     rbac.Role = "editor"
	CMSRoleAuthor     rbac.Role = "author"
	CMSRoleSubscriber rbac.Role = "subscriber"
)

// CMS permissions (API key scoped).
const (
	CMSPermissionRead   rbac.Permission = "read"
	CMSPermissionWrite  rbac.Permission = "write"
	CMSPermissionDelete rbac.Permission = "delete"
)

// CMS resources.
const (
	CMSResourcePost     rbac.Resource = "post"
	CMSResourcePage     rbac.Resource = "page"
	CMSResourceMedia    rbac.Resource = "media"
	CMSResourceComment  rbac.Resource = "comment"
	CMSResourceCategory rbac.Resource = "category"
	CMSResourceUser     rbac.Resource = "user"
)

// CMS actions.
const (
	CMSActionRead    rbac.Action = "read"
	CMSActionWrite   rbac.Action = "write"
	CMSActionDelete  rbac.Action = "delete"
	CMSActionPublish rbac.Action = "publish"
	CMSActionManage  rbac.Action = "manage"
)

// CMS returns an RBAC configuration for a Content Management System.
//
// Role hierarchy:
//
//	admin      (4) — full access, user management
//	editor     (3) — manage all content, publish posts
//	author     (2) — manage own posts/media, write comments
//	subscriber (1) — read content, write comments
//
// API keys are scoped to posts and media only.
func CMS() rbac.Config {
	return rbac.Config{
		Roles: []rbac.RoleDefinition{
			{Name: CMSRoleAdmin, Level: 4},
			{Name: CMSRoleEditor, Level: 3},
			{Name: CMSRoleAuthor, Level: 2},
			{Name: CMSRoleSubscriber, Level: 1},
		},
		Permissions: []rbac.Permission{
			CMSPermissionRead,
			CMSPermissionWrite,
			CMSPermissionDelete,
		},
		Resources: []rbac.Resource{
			CMSResourcePost,
			CMSResourcePage,
			CMSResourceMedia,
			CMSResourceComment,
			CMSResourceCategory,
			CMSResourceUser,
		},
		Actions: []rbac.Action{
			CMSActionRead,
			CMSActionWrite,
			CMSActionDelete,
			CMSActionPublish,
			CMSActionManage,
		},
		Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
			CMSRoleAdmin: {
				CMSResourcePost:     {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionPublish, CMSActionManage},
				CMSResourcePage:     {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionPublish, CMSActionManage},
				CMSResourceMedia:    {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionManage},
				CMSResourceComment:  {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionManage},
				CMSResourceCategory: {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionManage},
				CMSResourceUser:     {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionManage},
			},
			CMSRoleEditor: {
				CMSResourcePost:     {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionPublish},
				CMSResourcePage:     {CMSActionRead, CMSActionWrite, CMSActionDelete, CMSActionPublish},
				CMSResourceMedia:    {CMSActionRead, CMSActionWrite, CMSActionDelete},
				CMSResourceComment:  {CMSActionRead, CMSActionWrite, CMSActionDelete},
				CMSResourceCategory: {CMSActionRead, CMSActionWrite, CMSActionDelete},
				CMSResourceUser:     {CMSActionRead},
			},
			CMSRoleAuthor: {
				CMSResourcePost:     {CMSActionRead, CMSActionWrite},
				CMSResourcePage:     {CMSActionRead},
				CMSResourceMedia:    {CMSActionRead, CMSActionWrite},
				CMSResourceComment:  {CMSActionRead, CMSActionWrite},
				CMSResourceCategory: {CMSActionRead},
				CMSResourceUser:     {CMSActionRead},
			},
			CMSRoleSubscriber: {
				CMSResourcePost:     {CMSActionRead},
				CMSResourcePage:     {CMSActionRead},
				CMSResourceMedia:    {CMSActionRead},
				CMSResourceComment:  {CMSActionRead, CMSActionWrite},
				CMSResourceCategory: {CMSActionRead},
				CMSResourceUser:     {CMSActionRead},
			},
		},
		PermissionToActionMap: []rbac.PermissionMapping{
			{Permission: CMSPermissionRead, Action: CMSActionRead},
			{Permission: CMSPermissionWrite, Action: CMSActionWrite},
			{Permission: CMSPermissionDelete, Action: CMSActionDelete},
		},
		APIKeyScope: &rbac.APIKeyResourceScope{
			AllowedResources: []rbac.Resource{CMSResourcePost, CMSResourceMedia},
		},
	}
}
