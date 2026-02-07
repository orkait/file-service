package presets

import "file-management-service/pkg/rbac"

// E-commerce roles.
const (
	EcomRoleOwner   rbac.Role = "owner"
	EcomRoleManager rbac.Role = "manager"
	EcomRoleStaff   rbac.Role = "staff"
)

// E-commerce permissions (API key scoped).
const (
	EcomPermissionRead   rbac.Permission = "read"
	EcomPermissionWrite  rbac.Permission = "write"
	EcomPermissionDelete rbac.Permission = "delete"
)

// E-commerce resources.
const (
	EcomResourceProduct   rbac.Resource = "product"
	EcomResourceOrder     rbac.Resource = "order"
	EcomResourceCustomer  rbac.Resource = "customer"
	EcomResourceInventory rbac.Resource = "inventory"
	EcomResourceDiscount  rbac.Resource = "discount"
	EcomResourceReport    rbac.Resource = "report"
	EcomResourceSetting   rbac.Resource = "setting"
)

// E-commerce actions.
const (
	EcomActionRead    rbac.Action = "read"
	EcomActionWrite   rbac.Action = "write"
	EcomActionDelete  rbac.Action = "delete"
	EcomActionManage  rbac.Action = "manage"
	EcomActionRefund  rbac.Action = "refund"
)

// Ecommerce returns an RBAC configuration for an e-commerce platform.
//
// Role hierarchy:
//
//	owner   (3) — full access including settings and reports
//	manager (2) — manage products, orders, customers, inventory, discounts; refund orders
//	staff   (1) — read catalog, read/write orders, read customers
//
// API keys are scoped to products and orders (for storefront/integrations).
func Ecommerce() rbac.Config {
	return rbac.Config{
		Roles: []rbac.RoleDefinition{
			{Name: EcomRoleOwner, Level: 3},
			{Name: EcomRoleManager, Level: 2},
			{Name: EcomRoleStaff, Level: 1},
		},
		Permissions: []rbac.Permission{
			EcomPermissionRead,
			EcomPermissionWrite,
			EcomPermissionDelete,
		},
		Resources: []rbac.Resource{
			EcomResourceProduct,
			EcomResourceOrder,
			EcomResourceCustomer,
			EcomResourceInventory,
			EcomResourceDiscount,
			EcomResourceReport,
			EcomResourceSetting,
		},
		Actions: []rbac.Action{
			EcomActionRead,
			EcomActionWrite,
			EcomActionDelete,
			EcomActionManage,
			EcomActionRefund,
		},
		Capabilities: map[rbac.Role]map[rbac.Resource][]rbac.Action{
			EcomRoleOwner: {
				EcomResourceProduct:   {EcomActionRead, EcomActionWrite, EcomActionDelete, EcomActionManage},
				EcomResourceOrder:     {EcomActionRead, EcomActionWrite, EcomActionDelete, EcomActionRefund, EcomActionManage},
				EcomResourceCustomer:  {EcomActionRead, EcomActionWrite, EcomActionDelete, EcomActionManage},
				EcomResourceInventory: {EcomActionRead, EcomActionWrite, EcomActionDelete, EcomActionManage},
				EcomResourceDiscount:  {EcomActionRead, EcomActionWrite, EcomActionDelete, EcomActionManage},
				EcomResourceReport:    {EcomActionRead, EcomActionManage},
				EcomResourceSetting:   {EcomActionRead, EcomActionWrite, EcomActionManage},
			},
			EcomRoleManager: {
				EcomResourceProduct:   {EcomActionRead, EcomActionWrite, EcomActionDelete},
				EcomResourceOrder:     {EcomActionRead, EcomActionWrite, EcomActionRefund},
				EcomResourceCustomer:  {EcomActionRead, EcomActionWrite},
				EcomResourceInventory: {EcomActionRead, EcomActionWrite},
				EcomResourceDiscount:  {EcomActionRead, EcomActionWrite, EcomActionDelete},
				EcomResourceReport:    {EcomActionRead},
				EcomResourceSetting:   {EcomActionRead},
			},
			EcomRoleStaff: {
				EcomResourceProduct:   {EcomActionRead},
				EcomResourceOrder:     {EcomActionRead, EcomActionWrite},
				EcomResourceCustomer:  {EcomActionRead},
				EcomResourceInventory: {EcomActionRead},
				EcomResourceDiscount:  {EcomActionRead},
				EcomResourceReport:    {EcomActionRead},
				EcomResourceSetting:   {EcomActionRead},
			},
		},
		PermissionToActionMap: []rbac.PermissionMapping{
			{Permission: EcomPermissionRead, Action: EcomActionRead},
			{Permission: EcomPermissionWrite, Action: EcomActionWrite},
			{Permission: EcomPermissionDelete, Action: EcomActionDelete},
		},
		APIKeyScope: &rbac.APIKeyResourceScope{
			AllowedResources: []rbac.Resource{EcomResourceProduct, EcomResourceOrder},
		},
	}
}
