package presets

import "file-management-service/pkg/rbac"

// Preset is a named RBAC configuration constructor.
type Preset struct {
	Name   string
	Config func() rbac.Config
}

// All returns every registered preset. Add new presets here so they are
// automatically included in validation.
func All() []Preset {
	return []Preset{
		{Name: "FileManagement", Config: FileManagement},
		{Name: "CMS", Config: CMS},
		{Name: "Ecommerce", Config: Ecommerce},
		{Name: "ProjectManagement", Config: ProjectManagement},
		{Name: "SaaS", Config: SaaS},
	}
}
