package model

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Permission constants define all available granular permissions.
const (
	PermOrdersView   = "orders.view"
	PermOrdersCreate = "orders.create"
	PermOrdersEdit   = "orders.edit"
	PermOrdersDelete = "orders.delete"
	PermOrdersExport = "orders.export"

	PermProductsView   = "products.view"
	PermProductsCreate = "products.create"
	PermProductsEdit   = "products.edit"
	PermProductsDelete = "products.delete"

	PermShipmentsView   = "shipments.view"
	PermShipmentsCreate = "shipments.create"
	PermShipmentsEdit   = "shipments.edit"
	PermShipmentsDelete = "shipments.delete"

	PermReturnsView   = "returns.view"
	PermReturnsCreate = "returns.create"
	PermReturnsEdit   = "returns.edit"
	PermReturnsDelete = "returns.delete"

	PermCustomersView   = "customers.view"
	PermCustomersCreate = "customers.create"
	PermCustomersEdit   = "customers.edit"
	PermCustomersDelete = "customers.delete"

	PermInvoicesView   = "invoices.view"
	PermInvoicesCreate = "invoices.create"
	PermInvoicesDelete = "invoices.delete"

	PermIntegrationsManage = "integrations.manage"
	PermSettingsManage     = "settings.manage"
	PermUsersManage        = "users.manage"
	PermReportsView        = "reports.view"
	PermAuditView          = "audit.view"
	PermAutomationManage   = "automation.manage"
	PermWarehousesManage   = "warehouses.manage"
)

// AllPermissions is the complete list of valid permission strings.
var AllPermissions = []string{
	PermOrdersView, PermOrdersCreate, PermOrdersEdit, PermOrdersDelete, PermOrdersExport,
	PermProductsView, PermProductsCreate, PermProductsEdit, PermProductsDelete,
	PermShipmentsView, PermShipmentsCreate, PermShipmentsEdit, PermShipmentsDelete,
	PermReturnsView, PermReturnsCreate, PermReturnsEdit, PermReturnsDelete,
	PermCustomersView, PermCustomersCreate, PermCustomersEdit, PermCustomersDelete,
	PermInvoicesView, PermInvoicesCreate, PermInvoicesDelete,
	PermIntegrationsManage, PermSettingsManage, PermUsersManage,
	PermReportsView, PermAuditView, PermAutomationManage, PermWarehousesManage,
}

// PermissionGroups maps display group names to their permissions (for frontend).
var PermissionGroups = map[string][]string{
	"Zamówienia":    {PermOrdersView, PermOrdersCreate, PermOrdersEdit, PermOrdersDelete, PermOrdersExport},
	"Produkty":      {PermProductsView, PermProductsCreate, PermProductsEdit, PermProductsDelete},
	"Przesyłki":     {PermShipmentsView, PermShipmentsCreate, PermShipmentsEdit, PermShipmentsDelete},
	"Zwroty":        {PermReturnsView, PermReturnsCreate, PermReturnsEdit, PermReturnsDelete},
	"Klienci":       {PermCustomersView, PermCustomersCreate, PermCustomersEdit, PermCustomersDelete},
	"Faktury":       {PermInvoicesView, PermInvoicesCreate, PermInvoicesDelete},
	"Administracja": {PermIntegrationsManage, PermSettingsManage, PermUsersManage, PermReportsView, PermAuditView, PermAutomationManage, PermWarehousesManage},
}

// SystemRoleOwnerPermissions — full access.
var SystemRoleOwnerPermissions = AllPermissions

// SystemRoleAdminPermissions — full access except users.manage.
var SystemRoleAdminPermissions = func() []string {
	perms := make([]string, 0, len(AllPermissions))
	perms = append(perms, AllPermissions...)
	return perms
}()

// SystemRoleMemberPermissions — view + create permissions.
var SystemRoleMemberPermissions = []string{
	PermOrdersView, PermOrdersCreate, PermOrdersEdit, PermOrdersExport,
	PermProductsView,
	PermShipmentsView, PermShipmentsCreate, PermShipmentsEdit,
	PermReturnsView, PermReturnsCreate, PermReturnsEdit,
	PermCustomersView, PermCustomersCreate, PermCustomersEdit,
	PermInvoicesView, PermInvoicesCreate,
	PermReportsView,
}

// validPermissionSet is a lookup set for fast validation.
var validPermissionSet = func() map[string]bool {
	m := make(map[string]bool, len(AllPermissions))
	for _, p := range AllPermissions {
		m[p] = true
	}
	return m
}()

// IsValidPermission checks if a permission string is valid.
func IsValidPermission(p string) bool {
	return validPermissionSet[p]
}

// Role represents a custom role with granular permissions.
type Role struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HasPermission checks if this role has a given permission.
func (r *Role) HasPermission(permission string) bool {
	return slices.Contains(r.Permissions, permission)
}

// CreateRoleRequest is the payload for creating a new role.
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

// Validate validates the create role request.
func (r *CreateRoleRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 200); err != nil {
		return err
	}
	for _, p := range r.Permissions {
		if !IsValidPermission(p) {
			return errors.New("invalid permission: " + p)
		}
	}
	return nil
}

// UpdateRoleRequest is the payload for updating a role.
type UpdateRoleRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// Validate validates the update role request.
func (r *UpdateRoleRequest) Validate() error {
	if r.Name == nil && r.Description == nil && r.Permissions == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name must not be empty")
	}
	if r.Name != nil {
		if err := validateMaxLength("name", *r.Name, 200); err != nil {
			return err
		}
	}
	if r.Permissions != nil {
		for _, p := range r.Permissions {
			if !IsValidPermission(p) {
				return errors.New("invalid permission: " + p)
			}
		}
	}
	return nil
}

// RoleListFilter holds the filtering/pagination for listing roles.
type RoleListFilter struct {
	PaginationParams
}
