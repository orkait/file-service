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
		return fmt.Errorf("rbac config: Roles must not be empty")
	}
	if len(c.Permissions) == 0 {
		return fmt.Errorf("rbac config: Permissions must not be empty")
	}
	if len(c.Resources) == 0 {
		return fmt.Errorf("rbac config: Resources must not be empty")
	}
	if len(c.Actions) == 0 {
		return fmt.Errorf("rbac config: Actions must not be empty")
	}
	if len(c.Capabilities) == 0 {
		return fmt.Errorf("rbac config: Capabilities must not be empty")
	}
	if len(c.PermissionToActionMap) == 0 {
		return fmt.Errorf("rbac config: PermissionToActionMap must not be empty")
	}

	roleNames := make(map[Role]bool, len(c.Roles))
	roleLevels := make(map[int]Role, len(c.Roles))
	for _, rd := range c.Roles {
		if rd.Name == "" {
			return fmt.Errorf("rbac config: role name must not be empty")
		}
		if roleNames[rd.Name] {
			return fmt.Errorf("rbac config: duplicate role name: %s", rd.Name)
		}
		if existing, dup := roleLevels[rd.Level]; dup {
			return fmt.Errorf("rbac config: duplicate role level %d (roles %s and %s)", rd.Level, existing, rd.Name)
		}
		roleNames[rd.Name] = true
		roleLevels[rd.Level] = rd.Name
	}

	permSet := make(map[Permission]bool, len(c.Permissions))
	for _, p := range c.Permissions {
		if p == "" {
			return fmt.Errorf("rbac config: permission must not be empty")
		}
		if permSet[p] {
			return fmt.Errorf("rbac config: duplicate permission: %s", p)
		}
		permSet[p] = true
	}

	resSet := make(map[Resource]bool, len(c.Resources))
	for _, r := range c.Resources {
		if r == "" {
			return fmt.Errorf("rbac config: resource must not be empty")
		}
		if resSet[r] {
			return fmt.Errorf("rbac config: duplicate resource: %s", r)
		}
		resSet[r] = true
	}

	actSet := make(map[Action]bool, len(c.Actions))
	for _, a := range c.Actions {
		if a == "" {
			return fmt.Errorf("rbac config: action must not be empty")
		}
		if actSet[a] {
			return fmt.Errorf("rbac config: duplicate action: %s", a)
		}
		actSet[a] = true
	}

	for role, resources := range c.Capabilities {
		if !roleNames[role] {
			return fmt.Errorf("rbac config: capability references unknown role: %s", role)
		}
		for res, actions := range resources {
			if !resSet[res] {
				return fmt.Errorf("rbac config: capability for role %s references unknown resource: %s", role, res)
			}
			for _, act := range actions {
				if !actSet[act] {
					return fmt.Errorf("rbac config: capability for role %s on resource %s references unknown action: %s", role, res, act)
				}
			}
		}
	}

	mappedPerms := make(map[Permission]bool, len(c.PermissionToActionMap))
	mappedActions := make(map[Action]bool, len(c.PermissionToActionMap))
	for _, pm := range c.PermissionToActionMap {
		if !permSet[pm.Permission] {
			return fmt.Errorf("rbac config: permission mapping references unknown permission: %s", pm.Permission)
		}
		if !actSet[pm.Action] {
			return fmt.Errorf("rbac config: permission mapping references unknown action: %s", pm.Action)
		}
		if mappedPerms[pm.Permission] {
			return fmt.Errorf("rbac config: duplicate permission in mapping: %s", pm.Permission)
		}
		if mappedActions[pm.Action] {
			return fmt.Errorf("rbac config: duplicate action in mapping: %s", pm.Action)
		}
		mappedPerms[pm.Permission] = true
		mappedActions[pm.Action] = true
	}

	if c.APIKeyScope != nil {
		for _, res := range c.APIKeyScope.AllowedResources {
			if !resSet[res] {
				return fmt.Errorf("rbac config: API key scope references unknown resource: %s", res)
			}
		}
	}

	return nil
}
