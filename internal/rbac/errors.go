package rbac

import "errors"

var (
	ErrDenied            = errors.New("authorization denied")
	ErrNilSubject        = errors.New("subject is nil")
	ErrInvalidRole       = errors.New("invalid role")
	ErrInvalidPermission = errors.New("invalid permission")
)

const (
	errConfigRolesEmpty                    = "rbac config: roles must not be empty"
	errConfigPermissionsEmpty              = "rbac config: permissions must not be empty"
	errConfigResourcesEmpty                = "rbac config: resources must not be empty"
	errConfigActionsEmpty                  = "rbac config: actions must not be empty"
	errConfigCapabilitiesEmpty             = "rbac config: capabilities must not be empty"
	errConfigPermissionToActionMapEmpty    = "rbac config: permission-to-action map must not be empty"
	errConfigRoleNameEmpty                 = "rbac config: role name must not be empty"
	errConfigDuplicateRoleNameFmt          = "rbac config: duplicate role name: %s"
	errConfigDuplicateRoleLevelFmt         = "rbac config: duplicate role level %d (roles %s and %s)"
	errConfigPermissionEmpty               = "rbac config: permission must not be empty"
	errConfigDuplicatePermissionFmt        = "rbac config: duplicate permission: %s"
	errConfigResourceEmpty                 = "rbac config: resource must not be empty"
	errConfigDuplicateResourceFmt          = "rbac config: duplicate resource: %s"
	errConfigActionEmpty                   = "rbac config: action must not be empty"
	errConfigDuplicateActionFmt            = "rbac config: duplicate action: %s"
	errConfigCapabilityUnknownRoleFmt      = "rbac config: capability references unknown role: %s"
	errConfigCapabilityUnknownResourceFmt  = "rbac config: capability for role %s references unknown resource: %s"
	errConfigCapabilityUnknownActionFmt    = "rbac config: capability for role %s on resource %s references unknown action: %s"
	errConfigMappingUnknownPermissionFmt   = "rbac config: permission mapping references unknown permission: %s"
	errConfigMappingUnknownActionFmt       = "rbac config: permission mapping references unknown action: %s"
	errConfigMappingDuplicatePermissionFmt = "rbac config: duplicate permission in mapping: %s"
	errConfigMappingDuplicateActionFmt     = "rbac config: duplicate action in mapping: %s"
	errConfigAPIKeyScopeUnknownResourceFmt = "rbac config: API key scope references unknown resource: %s"
	errMustNewPanicFmt                     = "rbac.MustNew: %v"
	errDeniedUnknownAuthTypeFmt            = "unknown auth type: %s"
	errDeniedRoleCheckRequiresJWT          = "role check requires JWT authentication"
	errDeniedMinRoleRequiredFmt            = "requires minimum role '%s', but user has role '%s'"
	errDeniedUserRoleEmpty                 = "user role is empty"
	errDeniedRoleCannotPerformActionFmt    = "role '%s' cannot perform action '%s' on resource '%s'"
	errDeniedAPIKeyAccessNotConfigured     = "API key access is not configured"
	errDeniedAPIKeyResourceNotAllowedFmt   = "API keys cannot access resource '%s'"
	errDeniedAPIKeyActionNotSupportedFmt   = "action '%s' is not supported for API keys"
	errDeniedAPIKeyPermissionMissingFmt    = "API key lacks required permission '%s' for action '%s'"
	errInvalidPermissionArrayCannotBeEmpty = "permissions array cannot be empty"
)
