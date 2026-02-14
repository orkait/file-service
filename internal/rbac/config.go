package rbac

import "fmt"

// Config holds all RBAC configuration
type Config struct {
	Roles                 []RoleDefinition
	Permissions           []Permission
	Resources             []Resource
	Actions               []Action
	Capabilities          map[Role]map[Resource][]Action
	PermissionToActionMap []PermissionMapping
	APIKeyScope           *APIKeyResourceScope
}

// Validate checks internal consistency of the Config
func (c *Config) Validate() error {
	if len(c.Roles) == 0 {
		return fmt.Errorf(errConfigRolesEmpty)
	}
	if len(c.Permissions) == 0 {
		return fmt.Errorf(errConfigPermissionsEmpty)
	}
	if len(c.Resources) == 0 {
		return fmt.Errorf(errConfigResourcesEmpty)
	}
	if len(c.Actions) == 0 {
		return fmt.Errorf(errConfigActionsEmpty)
	}
	if len(c.Capabilities) == 0 {
		return fmt.Errorf(errConfigCapabilitiesEmpty)
	}
	if len(c.PermissionToActionMap) == 0 {
		return fmt.Errorf(errConfigPermissionToActionMapEmpty)
	}

	roleNames := make(map[Role]bool, len(c.Roles))
	roleLevels := make(map[int]Role, len(c.Roles))
	for _, rd := range c.Roles {
		if rd.Name == "" {
			return fmt.Errorf(errConfigRoleNameEmpty)
		}
		if roleNames[rd.Name] {
			return fmt.Errorf(errConfigDuplicateRoleNameFmt, rd.Name)
		}
		if existing, dup := roleLevels[rd.Level]; dup {
			return fmt.Errorf(errConfigDuplicateRoleLevelFmt, rd.Level, existing, rd.Name)
		}
		roleNames[rd.Name] = true
		roleLevels[rd.Level] = rd.Name
	}

	permSet := make(map[Permission]bool, len(c.Permissions))
	for _, p := range c.Permissions {
		if p == "" {
			return fmt.Errorf(errConfigPermissionEmpty)
		}
		if permSet[p] {
			return fmt.Errorf(errConfigDuplicatePermissionFmt, p)
		}
		permSet[p] = true
	}

	resSet := make(map[Resource]bool, len(c.Resources))
	for _, r := range c.Resources {
		if r == "" {
			return fmt.Errorf(errConfigResourceEmpty)
		}
		if resSet[r] {
			return fmt.Errorf(errConfigDuplicateResourceFmt, r)
		}
		resSet[r] = true
	}

	actSet := make(map[Action]bool, len(c.Actions))
	for _, a := range c.Actions {
		if a == "" {
			return fmt.Errorf(errConfigActionEmpty)
		}
		if actSet[a] {
			return fmt.Errorf(errConfigDuplicateActionFmt, a)
		}
		actSet[a] = true
	}

	for role, resources := range c.Capabilities {
		if !roleNames[role] {
			return fmt.Errorf(errConfigCapabilityUnknownRoleFmt, role)
		}
		for res, actions := range resources {
			if !resSet[res] {
				return fmt.Errorf(errConfigCapabilityUnknownResourceFmt, role, res)
			}
			for _, act := range actions {
				if !actSet[act] {
					return fmt.Errorf(errConfigCapabilityUnknownActionFmt, role, res, act)
				}
			}
		}
	}

	mappedPerms := make(map[Permission]bool, len(c.PermissionToActionMap))
	mappedActions := make(map[Action]bool, len(c.PermissionToActionMap))
	for _, pm := range c.PermissionToActionMap {
		if !permSet[pm.Permission] {
			return fmt.Errorf(errConfigMappingUnknownPermissionFmt, pm.Permission)
		}
		if !actSet[pm.Action] {
			return fmt.Errorf(errConfigMappingUnknownActionFmt, pm.Action)
		}
		if mappedPerms[pm.Permission] {
			return fmt.Errorf(errConfigMappingDuplicatePermissionFmt, pm.Permission)
		}
		if mappedActions[pm.Action] {
			return fmt.Errorf(errConfigMappingDuplicateActionFmt, pm.Action)
		}
		mappedPerms[pm.Permission] = true
		mappedActions[pm.Action] = true
	}

	if c.APIKeyScope != nil {
		for _, res := range c.APIKeyScope.AllowedResources {
			if !resSet[res] {
				return fmt.Errorf(errConfigAPIKeyScopeUnknownResourceFmt, res)
			}
		}
	}

	return nil
}
