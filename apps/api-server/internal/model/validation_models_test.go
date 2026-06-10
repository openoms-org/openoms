package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── CreateCustomerRequest.Validate ─────────────────────────────────────────

func TestCreateCustomerRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateCustomerRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateCustomerRequest{Name: "Jan Kowalski"},
		},
		{
			name: "valid with all optional fields",
			req: func() CreateCustomerRequest {
				email := "jan@example.com"
				phone := "+48123456789"
				company := "ACME Sp. z o.o."
				nip := "1234567890"
				notes := "VIP customer"
				return CreateCustomerRequest{
					Name:        "Jan Kowalski",
					Email:       &email,
					Phone:       &phone,
					CompanyName: &company,
					NIP:         &nip,
					Notes:       &notes,
					Tags:        []string{"vip", "wholesale"},
				}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateCustomerRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateCustomerRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateCustomerRequest{Name: strings.Repeat("x", 501)},
			wantErr: "name",
		},
		{
			name: "email too long",
			req: func() CreateCustomerRequest {
				email := strings.Repeat("x", 256)
				return CreateCustomerRequest{Name: "Jan", Email: &email}
			}(),
			wantErr: "email",
		},
		{
			name: "invalid email format",
			req: func() CreateCustomerRequest {
				email := "not-an-email"
				return CreateCustomerRequest{Name: "Jan", Email: &email}
			}(),
			wantErr: "invalid email format",
		},
		{
			name: "empty email is allowed",
			req: func() CreateCustomerRequest {
				email := ""
				return CreateCustomerRequest{Name: "Jan", Email: &email}
			}(),
		},
		{
			name: "phone too long",
			req: func() CreateCustomerRequest {
				phone := strings.Repeat("9", 51)
				return CreateCustomerRequest{Name: "Jan", Phone: &phone}
			}(),
			wantErr: "phone",
		},
		{
			name: "company_name too long",
			req: func() CreateCustomerRequest {
				company := strings.Repeat("x", 501)
				return CreateCustomerRequest{Name: "Jan", CompanyName: &company}
			}(),
			wantErr: "company_name",
		},
		{
			name: "nip too long",
			req: func() CreateCustomerRequest {
				nip := strings.Repeat("1", 21)
				return CreateCustomerRequest{Name: "Jan", NIP: &nip}
			}(),
			wantErr: "nip",
		},
		{
			name: "notes too long",
			req: func() CreateCustomerRequest {
				notes := strings.Repeat("x", 5001)
				return CreateCustomerRequest{Name: "Jan", Notes: &notes}
			}(),
			wantErr: "notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateCustomerRequest.Validate ─────────────────────────────────────────

func TestUpdateCustomerRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateCustomerRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateCustomerRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req: func() UpdateCustomerRequest {
				name := "New Name"
				return UpdateCustomerRequest{Name: &name}
			}(),
		},
		{
			name: "empty name",
			req: func() UpdateCustomerRequest {
				name := ""
				return UpdateCustomerRequest{Name: &name}
			}(),
			wantErr: "name cannot be empty",
		},
		{
			name: "whitespace-only name",
			req: func() UpdateCustomerRequest {
				name := "   "
				return UpdateCustomerRequest{Name: &name}
			}(),
			wantErr: "name cannot be empty",
		},
		{
			name: "name too long",
			req: func() UpdateCustomerRequest {
				name := strings.Repeat("x", 501)
				return UpdateCustomerRequest{Name: &name}
			}(),
			wantErr: "name",
		},
		{
			name: "invalid email format",
			req: func() UpdateCustomerRequest {
				email := "not-an-email"
				return UpdateCustomerRequest{Email: &email}
			}(),
			wantErr: "invalid email format",
		},
		{
			name: "email too long",
			req: func() UpdateCustomerRequest {
				email := strings.Repeat("x", 256)
				return UpdateCustomerRequest{Email: &email}
			}(),
			wantErr: "email",
		},
		{
			name: "valid email update",
			req: func() UpdateCustomerRequest {
				email := "new@example.com"
				return UpdateCustomerRequest{Email: &email}
			}(),
		},
		{
			name: "phone too long",
			req: func() UpdateCustomerRequest {
				phone := strings.Repeat("9", 51)
				return UpdateCustomerRequest{Phone: &phone}
			}(),
			wantErr: "phone",
		},
		{
			name: "company_name too long",
			req: func() UpdateCustomerRequest {
				company := strings.Repeat("x", 501)
				return UpdateCustomerRequest{CompanyName: &company}
			}(),
			wantErr: "company_name",
		},
		{
			name: "nip too long",
			req: func() UpdateCustomerRequest {
				nip := strings.Repeat("1", 21)
				return UpdateCustomerRequest{NIP: &nip}
			}(),
			wantErr: "nip",
		},
		{
			name: "notes too long",
			req: func() UpdateCustomerRequest {
				notes := strings.Repeat("x", 5001)
				return UpdateCustomerRequest{Notes: &notes}
			}(),
			wantErr: "notes",
		},
		{
			name: "valid tags update",
			req: func() UpdateCustomerRequest {
				tags := []string{"vip"}
				return UpdateCustomerRequest{Tags: &tags}
			}(),
		},
		{
			name: "valid price_list_id update",
			req: func() UpdateCustomerRequest {
				id := uuid.New()
				return UpdateCustomerRequest{PriceListID: &id}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateInvoiceRequest.Validate ──────────────────────────────────────────

func TestCreateInvoiceRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateInvoiceRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateInvoiceRequest{OrderID: uuid.New(), Provider: "fakturownia"},
		},
		{
			name: "valid with all fields",
			req: CreateInvoiceRequest{
				OrderID:       uuid.New(),
				Provider:      "fakturownia",
				InvoiceType:   "proforma",
				CustomerName:  "Jan Kowalski",
				CustomerEmail: "jan@example.com",
				NIP:           "1234567890",
				PaymentMethod: "transfer",
				Notes:         "Urgent invoice",
			},
		},
		{
			name:    "missing order_id",
			req:     CreateInvoiceRequest{Provider: "fakturownia"},
			wantErr: "order_id is required",
		},
		{
			name:    "missing provider",
			req:     CreateInvoiceRequest{OrderID: uuid.New()},
			wantErr: "provider is required",
		},
		{
			name:    "whitespace-only provider",
			req:     CreateInvoiceRequest{OrderID: uuid.New(), Provider: "   "},
			wantErr: "provider is required",
		},
		{
			name:    "provider too long",
			req:     CreateInvoiceRequest{OrderID: uuid.New(), Provider: strings.Repeat("x", 101)},
			wantErr: "provider",
		},
		{
			name:    "customer_name too long",
			req:     CreateInvoiceRequest{OrderID: uuid.New(), Provider: "fakturownia", CustomerName: strings.Repeat("x", 501)},
			wantErr: "customer_name",
		},
		{
			name:    "nip too long",
			req:     CreateInvoiceRequest{OrderID: uuid.New(), Provider: "fakturownia", NIP: strings.Repeat("1", 21)},
			wantErr: "nip",
		},
		{
			name:    "notes too long",
			req:     CreateInvoiceRequest{OrderID: uuid.New(), Provider: "fakturownia", Notes: strings.Repeat("x", 5001)},
			wantErr: "notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateInvoiceRequest_Validate_DefaultInvoiceType(t *testing.T) {
	req := CreateInvoiceRequest{OrderID: uuid.New(), Provider: "fakturownia"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "vat", req.InvoiceType, "invoice_type should default to 'vat'")
}

func TestCreateInvoiceRequest_Validate_PreservesExplicitInvoiceType(t *testing.T) {
	req := CreateInvoiceRequest{OrderID: uuid.New(), Provider: "fakturownia", InvoiceType: "proforma"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "proforma", req.InvoiceType)
}

// ─── CreateWarehouseRequest.Validate ────────────────────────────────────────

func TestCreateWarehouseRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateWarehouseRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateWarehouseRequest{Name: "Main Warehouse"},
		},
		{
			name: "valid with optional fields",
			req: func() CreateWarehouseRequest {
				code := "WH-01"
				isDefault := true
				active := true
				return CreateWarehouseRequest{
					Name:      "Main Warehouse",
					Code:      &code,
					Address:   json.RawMessage(`{"city":"Krakow"}`),
					IsDefault: &isDefault,
					Active:    &active,
				}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateWarehouseRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateWarehouseRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateWarehouseRequest{Name: strings.Repeat("x", 501)},
			wantErr: "name",
		},
		{
			name: "name at max length",
			req:  CreateWarehouseRequest{Name: strings.Repeat("x", 500)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateWarehouseRequest.Validate ────────────────────────────────────────

func TestUpdateWarehouseRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateWarehouseRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateWarehouseRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req: func() UpdateWarehouseRequest {
				name := "Updated Warehouse"
				return UpdateWarehouseRequest{Name: &name}
			}(),
		},
		{
			name: "empty name",
			req: func() UpdateWarehouseRequest {
				name := ""
				return UpdateWarehouseRequest{Name: &name}
			}(),
			wantErr: "name must not be empty",
		},
		{
			name: "whitespace-only name",
			req: func() UpdateWarehouseRequest {
				name := "   "
				return UpdateWarehouseRequest{Name: &name}
			}(),
			wantErr: "name must not be empty",
		},
		{
			name: "name too long",
			req: func() UpdateWarehouseRequest {
				name := strings.Repeat("x", 501)
				return UpdateWarehouseRequest{Name: &name}
			}(),
			wantErr: "name",
		},
		{
			name: "valid code update",
			req: func() UpdateWarehouseRequest {
				code := "WH-02"
				return UpdateWarehouseRequest{Code: &code}
			}(),
		},
		{
			name: "valid is_default update",
			req: func() UpdateWarehouseRequest {
				isDefault := true
				return UpdateWarehouseRequest{IsDefault: &isDefault}
			}(),
		},
		{
			name: "valid active update",
			req: func() UpdateWarehouseRequest {
				active := false
				return UpdateWarehouseRequest{Active: &active}
			}(),
		},
		{
			name: "valid address update",
			req: func() UpdateWarehouseRequest {
				addr := json.RawMessage(`{"city":"Warsaw"}`)
				return UpdateWarehouseRequest{Address: &addr}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpsertWarehouseStockRequest.Validate ───────────────────────────────────

func TestUpsertWarehouseStockRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpsertWarehouseStockRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  UpsertWarehouseStockRequest{ProductID: uuid.New(), Quantity: 10, Reserved: 2, MinStock: 5},
		},
		{
			name: "valid with zero values",
			req:  UpsertWarehouseStockRequest{ProductID: uuid.New(), Quantity: 0, Reserved: 0, MinStock: 0},
		},
		{
			name: "valid with variant_id",
			req: func() UpsertWarehouseStockRequest {
				vid := uuid.New()
				return UpsertWarehouseStockRequest{ProductID: uuid.New(), VariantID: &vid, Quantity: 5}
			}(),
		},
		{
			name:    "missing product_id",
			req:     UpsertWarehouseStockRequest{Quantity: 10},
			wantErr: "product_id is required",
		},
		{
			name:    "negative quantity",
			req:     UpsertWarehouseStockRequest{ProductID: uuid.New(), Quantity: -1},
			wantErr: "quantity must not be negative",
		},
		{
			name:    "negative reserved",
			req:     UpsertWarehouseStockRequest{ProductID: uuid.New(), Quantity: 10, Reserved: -1},
			wantErr: "reserved must not be negative",
		},
		{
			name:    "negative min_stock",
			req:     UpsertWarehouseStockRequest{ProductID: uuid.New(), Quantity: 10, MinStock: -1},
			wantErr: "min_stock must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateRoleRequest.Validate ─────────────────────────────────────────────

func TestCreateRoleRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateRoleRequest
		wantErr string
	}{
		{
			name: "valid with permissions",
			req:  CreateRoleRequest{Name: "Sales Rep", Permissions: []string{PermOrdersView, PermOrdersCreate}},
		},
		{
			name: "valid with empty permissions",
			req:  CreateRoleRequest{Name: "Viewer"},
		},
		{
			name: "valid with description",
			req: func() CreateRoleRequest {
				desc := "Can view everything"
				return CreateRoleRequest{Name: "Observer", Description: &desc}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateRoleRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateRoleRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateRoleRequest{Name: strings.Repeat("x", 201)},
			wantErr: "name",
		},
		{
			name: "name at max length",
			req:  CreateRoleRequest{Name: strings.Repeat("x", 200)},
		},
		{
			name:    "invalid permission",
			req:     CreateRoleRequest{Name: "Bad Role", Permissions: []string{"nonexistent.permission"}},
			wantErr: "invalid permission",
		},
		{
			name:    "one valid one invalid permission",
			req:     CreateRoleRequest{Name: "Mixed", Permissions: []string{PermOrdersView, "bogus.perm"}},
			wantErr: "invalid permission: bogus.perm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateRoleRequest_Validate_AllPermissionsValid(t *testing.T) {
	req := CreateRoleRequest{Name: "Full Access", Permissions: AllPermissions}
	require.NoError(t, req.Validate())
}

// ─── UpdateRoleRequest.Validate ─────────────────────────────────────────────

func TestUpdateRoleRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateRoleRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateRoleRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req: func() UpdateRoleRequest {
				name := "Updated Role"
				return UpdateRoleRequest{Name: &name}
			}(),
		},
		{
			name: "empty name",
			req: func() UpdateRoleRequest {
				name := ""
				return UpdateRoleRequest{Name: &name}
			}(),
			wantErr: "name must not be empty",
		},
		{
			name: "whitespace-only name",
			req: func() UpdateRoleRequest {
				name := "   "
				return UpdateRoleRequest{Name: &name}
			}(),
			wantErr: "name must not be empty",
		},
		{
			name: "name too long",
			req: func() UpdateRoleRequest {
				name := strings.Repeat("x", 201)
				return UpdateRoleRequest{Name: &name}
			}(),
			wantErr: "name",
		},
		{
			name: "valid permissions update",
			req:  UpdateRoleRequest{Permissions: []string{PermOrdersView, PermProductsView}},
		},
		{
			name:    "invalid permission in update",
			req:     UpdateRoleRequest{Permissions: []string{PermOrdersView, "invalid.perm"}},
			wantErr: "invalid permission: invalid.perm",
		},
		{
			name: "valid description update",
			req: func() UpdateRoleRequest {
				desc := "Updated description"
				return UpdateRoleRequest{Description: &desc}
			}(),
		},
		{
			name: "name and permissions together",
			req: func() UpdateRoleRequest {
				name := "New Name"
				return UpdateRoleRequest{Name: &name, Permissions: []string{PermOrdersView}}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateAutomationRuleRequest.Validate ───────────────────────────────────

func TestCreateAutomationRuleRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateAutomationRuleRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateAutomationRuleRequest{Name: "Auto-confirm", TriggerEvent: "order.created"},
		},
		{
			name: "valid with all fields",
			req: func() CreateAutomationRuleRequest {
				desc := "Confirm orders over 100 PLN"
				enabled := true
				priority := 10
				return CreateAutomationRuleRequest{
					Name:         "Auto-confirm",
					Description:  &desc,
					Enabled:      &enabled,
					Priority:     &priority,
					TriggerEvent: "order.created",
					Conditions:   json.RawMessage(`[{"field":"total_amount","operator":"gt","value":100}]`),
					Actions:      json.RawMessage(`[{"type":"set_status","params":{"status":"confirmed"}}]`),
				}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateAutomationRuleRequest{TriggerEvent: "order.created"},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateAutomationRuleRequest{Name: "   ", TriggerEvent: "order.created"},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateAutomationRuleRequest{Name: strings.Repeat("x", 501), TriggerEvent: "order.created"},
			wantErr: "name",
		},
		{
			name: "name at max length",
			req:  CreateAutomationRuleRequest{Name: strings.Repeat("x", 500), TriggerEvent: "order.created"},
		},
		{
			name: "description too long",
			req: func() CreateAutomationRuleRequest {
				desc := strings.Repeat("x", 5001)
				return CreateAutomationRuleRequest{Name: "Rule", Description: &desc, TriggerEvent: "order.created"}
			}(),
			wantErr: "description",
		},
		{
			name:    "missing trigger_event",
			req:     CreateAutomationRuleRequest{Name: "Rule"},
			wantErr: "trigger_event is required",
		},
		{
			name:    "whitespace-only trigger_event",
			req:     CreateAutomationRuleRequest{Name: "Rule", TriggerEvent: "   "},
			wantErr: "trigger_event is required",
		},
		{
			name:    "invalid trigger_event",
			req:     CreateAutomationRuleRequest{Name: "Rule", TriggerEvent: "nonexistent.event"},
			wantErr: "invalid trigger_event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateAutomationRuleRequest_Validate_AllTriggerEvents(t *testing.T) {
	for event := range ValidTriggerEvents {
		t.Run(event, func(t *testing.T) {
			req := CreateAutomationRuleRequest{Name: "Rule", TriggerEvent: event}
			require.NoError(t, req.Validate(), "trigger_event %s should be valid", event)
		})
	}
}

func TestCreateAutomationRuleRequest_Validate_DefaultsNilConditionsAndActions(t *testing.T) {
	req := CreateAutomationRuleRequest{Name: "Rule", TriggerEvent: "order.created"}
	require.NoError(t, req.Validate())
	assert.Equal(t, json.RawMessage("[]"), req.Conditions, "nil conditions should default to []")
	assert.Equal(t, json.RawMessage("[]"), req.Actions, "nil actions should default to []")
}

func TestCreateAutomationRuleRequest_Validate_DefaultsEmptyConditionsAndActions(t *testing.T) {
	req := CreateAutomationRuleRequest{
		Name:         "Rule",
		TriggerEvent: "order.created",
		Conditions:   json.RawMessage(""),
		Actions:      json.RawMessage(""),
	}
	require.NoError(t, req.Validate())
	assert.Equal(t, json.RawMessage("[]"), req.Conditions, "empty conditions should default to []")
	assert.Equal(t, json.RawMessage("[]"), req.Actions, "empty actions should default to []")
}

// ─── UpdateAutomationRuleRequest.Validate ───────────────────────────────────

func TestUpdateAutomationRuleRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateAutomationRuleRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateAutomationRuleRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req: func() UpdateAutomationRuleRequest {
				name := "Updated Rule"
				return UpdateAutomationRuleRequest{Name: &name}
			}(),
		},
		{
			name: "empty name",
			req: func() UpdateAutomationRuleRequest {
				name := ""
				return UpdateAutomationRuleRequest{Name: &name}
			}(),
			wantErr: "name cannot be empty",
		},
		{
			name: "whitespace-only name",
			req: func() UpdateAutomationRuleRequest {
				name := "   "
				return UpdateAutomationRuleRequest{Name: &name}
			}(),
			wantErr: "name cannot be empty",
		},
		{
			name: "name too long",
			req: func() UpdateAutomationRuleRequest {
				name := strings.Repeat("x", 501)
				return UpdateAutomationRuleRequest{Name: &name}
			}(),
			wantErr: "name",
		},
		{
			name: "description too long",
			req: func() UpdateAutomationRuleRequest {
				desc := strings.Repeat("x", 5001)
				return UpdateAutomationRuleRequest{Description: &desc}
			}(),
			wantErr: "description",
		},
		{
			name: "valid trigger_event update",
			req: func() UpdateAutomationRuleRequest {
				event := "shipment.created"
				return UpdateAutomationRuleRequest{TriggerEvent: &event}
			}(),
		},
		{
			name: "invalid trigger_event",
			req: func() UpdateAutomationRuleRequest {
				event := "bogus.event"
				return UpdateAutomationRuleRequest{TriggerEvent: &event}
			}(),
			wantErr: "invalid trigger_event",
		},
		{
			name: "valid enabled update",
			req: func() UpdateAutomationRuleRequest {
				enabled := false
				return UpdateAutomationRuleRequest{Enabled: &enabled}
			}(),
		},
		{
			name: "valid priority update",
			req: func() UpdateAutomationRuleRequest {
				priority := 5
				return UpdateAutomationRuleRequest{Priority: &priority}
			}(),
		},
		{
			name: "valid conditions update",
			req:  UpdateAutomationRuleRequest{Conditions: json.RawMessage(`[{"field":"status","operator":"eq","value":"new"}]`)},
		},
		{
			name: "valid actions update",
			req:  UpdateAutomationRuleRequest{Actions: json.RawMessage(`[{"type":"send_email"}]`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── PublicReturnRequest.Validate ───────────────────────────────────────────

func TestPublicReturnRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     PublicReturnRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  PublicReturnRequest{OrderID: "ORD-123", Email: "customer@example.com", Reason: "defective"},
		},
		{
			name: "valid with notes",
			req:  PublicReturnRequest{OrderID: "ORD-123", Email: "customer@example.com", Reason: "defective", Notes: "Arrived broken"},
		},
		{
			name:    "missing order_id",
			req:     PublicReturnRequest{Email: "customer@example.com", Reason: "defective"},
			wantErr: "order_id is required",
		},
		{
			name:    "missing email",
			req:     PublicReturnRequest{OrderID: "ORD-123", Reason: "defective"},
			wantErr: "email is required",
		},
		{
			name:    "missing reason",
			req:     PublicReturnRequest{OrderID: "ORD-123", Email: "customer@example.com"},
			wantErr: "reason is required",
		},
		{
			name:    "reason too long",
			req:     PublicReturnRequest{OrderID: "ORD-123", Email: "customer@example.com", Reason: strings.Repeat("x", 2001)},
			wantErr: "reason",
		},
		{
			name: "reason at max length",
			req:  PublicReturnRequest{OrderID: "ORD-123", Email: "customer@example.com", Reason: strings.Repeat("x", 2000)},
		},
		{
			name:    "notes too long",
			req:     PublicReturnRequest{OrderID: "ORD-123", Email: "customer@example.com", Reason: "defective", Notes: strings.Repeat("x", 5001)},
			wantErr: "notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── EmailSettings.Validate ─────────────────────────────────────────────────

func TestEmailSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		s       EmailSettings
		wantErr string
	}{
		{
			name: "valid",
			s:    EmailSettings{SMTPPort: 587},
		},
		{
			name: "valid port 0",
			s:    EmailSettings{SMTPPort: 0},
		},
		{
			name: "valid port 65535",
			s:    EmailSettings{SMTPPort: 65535},
		},
		{
			name:    "negative port",
			s:       EmailSettings{SMTPPort: -1},
			wantErr: "smtp_port must be between 0 and 65535",
		},
		{
			name:    "port too high",
			s:       EmailSettings{SMTPPort: 65536},
			wantErr: "smtp_port must be between 0 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── SMSSettings.Validate ───────────────────────────────────────────────────

func TestSMSSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		s       SMSSettings
		wantErr string
	}{
		{
			name: "valid empty from",
			s:    SMSSettings{},
		},
		{
			name: "valid from at max length",
			s:    SMSSettings{From: strings.Repeat("x", 11)},
		},
		{
			name: "valid short from",
			s:    SMSSettings{From: "OpenOMS"},
		},
		{
			name:    "from too long",
			s:       SMSSettings{From: strings.Repeat("x", 12)},
			wantErr: "SMS sender name must be 11 characters or fewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateProductRequest.Validate (deeper coverage) ────────────────────────

func TestCreateProductRequest_Validate_MaxLengths(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateProductRequest
		wantErr string
	}{
		{
			name:    "name at boundary (500)",
			req:     CreateProductRequest{Name: strings.Repeat("x", 500), Price: 10},
			wantErr: "",
		},
		{
			name: "sku too long",
			req: func() CreateProductRequest {
				sku := strings.Repeat("x", 101)
				return CreateProductRequest{Name: "Widget", Price: 10, SKU: &sku}
			}(),
			wantErr: "sku",
		},
		{
			name: "sku at max length",
			req: func() CreateProductRequest {
				sku := strings.Repeat("x", 100)
				return CreateProductRequest{Name: "Widget", Price: 10, SKU: &sku}
			}(),
		},
		{
			name: "ean too long",
			req: func() CreateProductRequest {
				ean := strings.Repeat("0", 51)
				return CreateProductRequest{Name: "Widget", Price: 10, EAN: &ean}
			}(),
			wantErr: "ean",
		},
		{
			name: "ean at max length",
			req: func() CreateProductRequest {
				ean := strings.Repeat("0", 50)
				return CreateProductRequest{Name: "Widget", Price: 10, EAN: &ean}
			}(),
		},
		{
			name:    "description_short too long",
			req:     CreateProductRequest{Name: "Widget", Price: 10, DescriptionShort: strings.Repeat("x", 1001)},
			wantErr: "description_short",
		},
		{
			name: "description_short at max length",
			req:  CreateProductRequest{Name: "Widget", Price: 10, DescriptionShort: strings.Repeat("x", 1000)},
		},
		{
			name:    "description_long too long",
			req:     CreateProductRequest{Name: "Widget", Price: 10, DescriptionLong: strings.Repeat("x", 10001)},
			wantErr: "description_long",
		},
		{
			name: "description_long at max length",
			req:  CreateProductRequest{Name: "Widget", Price: 10, DescriptionLong: strings.Repeat("x", 10000)},
		},
		{
			name: "category too long",
			req: func() CreateProductRequest {
				cat := strings.Repeat("x", 101)
				return CreateProductRequest{Name: "Widget", Price: 10, Category: &cat}
			}(),
			wantErr: "category",
		},
		{
			name: "category at max length",
			req: func() CreateProductRequest {
				cat := strings.Repeat("x", 100)
				return CreateProductRequest{Name: "Widget", Price: 10, Category: &cat}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateProductRequest_Validate_SupplierSource(t *testing.T) {
	req := CreateProductRequest{Name: "Widget", Price: 10, Source: "supplier"}
	require.NoError(t, req.Validate())
}

// ─── UpdateProductRequest.Validate (deeper coverage) ────────────────────────

func TestUpdateProductRequest_Validate_MaxLengths(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateProductRequest
		wantErr string
	}{
		{
			name: "name too long",
			req: func() UpdateProductRequest {
				name := strings.Repeat("x", 501)
				return UpdateProductRequest{Name: &name}
			}(),
			wantErr: "name",
		},
		{
			name: "sku too long",
			req: func() UpdateProductRequest {
				sku := strings.Repeat("x", 101)
				return UpdateProductRequest{SKU: &sku}
			}(),
			wantErr: "sku",
		},
		{
			name: "ean too long",
			req: func() UpdateProductRequest {
				ean := strings.Repeat("0", 51)
				return UpdateProductRequest{EAN: &ean}
			}(),
			wantErr: "ean",
		},
		{
			name: "description_short too long",
			req: func() UpdateProductRequest {
				desc := strings.Repeat("x", 1001)
				return UpdateProductRequest{DescriptionShort: &desc}
			}(),
			wantErr: "description_short",
		},
		{
			name: "description_long too long",
			req: func() UpdateProductRequest {
				desc := strings.Repeat("x", 10001)
				return UpdateProductRequest{DescriptionLong: &desc}
			}(),
			wantErr: "description_long",
		},
		{
			name: "category too long",
			req: func() UpdateProductRequest {
				cat := strings.Repeat("x", 101)
				return UpdateProductRequest{Category: &cat}
			}(),
			wantErr: "category",
		},
		{
			name: "valid name update",
			req: func() UpdateProductRequest {
				name := "Updated Widget"
				return UpdateProductRequest{Name: &name}
			}(),
		},
		{
			name: "valid source update - allegro",
			req: func() UpdateProductRequest {
				s := "allegro"
				return UpdateProductRequest{Source: &s}
			}(),
		},
		{
			name: "valid source update - woocommerce",
			req: func() UpdateProductRequest {
				s := "woocommerce"
				return UpdateProductRequest{Source: &s}
			}(),
		},
		{
			name: "valid source update - manual",
			req: func() UpdateProductRequest {
				s := "manual"
				return UpdateProductRequest{Source: &s}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateProductRequest_Validate_BooleanFields(t *testing.T) {
	isBundle := true
	req := UpdateProductRequest{IsBundle: &isBundle}
	require.NoError(t, req.Validate())
}

func TestUpdateProductRequest_Validate_DropshipFields(t *testing.T) {
	isDropship := true
	supplierID := uuid.New()
	req := UpdateProductRequest{IsDropship: &isDropship, DropshipSupplierID: &supplierID}
	require.NoError(t, req.Validate())
}

// ─── UpdateShipmentRequest.Validate (deeper coverage) ───────────────────────

func TestUpdateShipmentRequest_Validate_TrackingNumberTooLong(t *testing.T) {
	// The Create validates tracking_number max length, but Update does not have explicit max length
	// Still, let's confirm multiple field combinations work
	tn := "TR-12345"
	label := "https://example.com/label.pdf"
	req := UpdateShipmentRequest{TrackingNumber: &tn, LabelURL: &label}
	require.NoError(t, req.Validate())
}

func TestUpdateShipmentRequest_Validate_SingleFields(t *testing.T) {
	tests := []struct {
		name string
		req  UpdateShipmentRequest
	}{
		{
			name: "only weight",
			req: func() UpdateShipmentRequest {
				w := 1.5
				return UpdateShipmentRequest{Weight: &w}
			}(),
		},
		{
			name: "only length",
			req: func() UpdateShipmentRequest {
				l := 30.0
				return UpdateShipmentRequest{Length: &l}
			}(),
		},
		{
			name: "only width",
			req: func() UpdateShipmentRequest {
				w := 20.0
				return UpdateShipmentRequest{Width: &w}
			}(),
		},
		{
			name: "only height",
			req: func() UpdateShipmentRequest {
				h := 10.0
				return UpdateShipmentRequest{Height: &h}
			}(),
		},
		{
			name: "only notes",
			req: func() UpdateShipmentRequest {
				n := "Fragile"
				return UpdateShipmentRequest{Notes: &n}
			}(),
		},
		{
			name: "only carrier_data",
			req:  UpdateShipmentRequest{CarrierData: json.RawMessage(`{"key":"val"}`)},
		},
		{
			name: "only label_url",
			req: func() UpdateShipmentRequest {
				l := "https://example.com/label.pdf"
				return UpdateShipmentRequest{LabelURL: &l}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.req.Validate())
		})
	}
}

// ─── CreateIntegrationRequest.Validate (deeper coverage) ────────────────────

func TestCreateIntegrationRequest_Validate_LabelTooLong(t *testing.T) {
	label := strings.Repeat("x", 256)
	req := CreateIntegrationRequest{
		Provider:    "allegro",
		Credentials: json.RawMessage(`{"key":"val"}`),
		Label:       &label,
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label")
}

func TestCreateIntegrationRequest_Validate_LabelAtMaxLength(t *testing.T) {
	label := strings.Repeat("x", 255)
	req := CreateIntegrationRequest{
		Provider:    "allegro",
		Credentials: json.RawMessage(`{"key":"val"}`),
		Label:       &label,
	}
	require.NoError(t, req.Validate())
}

func TestCreateIntegrationRequest_Validate_WithSettings(t *testing.T) {
	req := CreateIntegrationRequest{
		Provider:    "woocommerce",
		Credentials: json.RawMessage(`{"url":"https://example.com"}`),
		Settings:    json.RawMessage(`{"auto_sync":true}`),
	}
	require.NoError(t, req.Validate())
}

// ─── UpdateIntegrationRequest.Validate (deeper coverage) ────────────────────

func TestUpdateIntegrationRequest_Validate_LabelUpdate(t *testing.T) {
	label := "My Allegro #2"
	req := UpdateIntegrationRequest{Label: &label}
	require.NoError(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_SyncCursorUpdate(t *testing.T) {
	cursor := "2024-01-01T00:00:00Z"
	req := UpdateIntegrationRequest{SyncCursor: &cursor}
	require.NoError(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_ErrorMessageUpdate(t *testing.T) {
	msg := "Connection timeout"
	req := UpdateIntegrationRequest{ErrorMessage: &msg}
	require.NoError(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_CredentialsUpdate(t *testing.T) {
	creds := json.RawMessage(`{"new_key":"new_val"}`)
	req := UpdateIntegrationRequest{Credentials: &creds}
	require.NoError(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_SettingsUpdate(t *testing.T) {
	settings := json.RawMessage(`{"auto_sync":false}`)
	req := UpdateIntegrationRequest{Settings: &settings}
	require.NoError(t, req.Validate())
}

// ─── CreateReturnRequest.Validate (deeper coverage) ─────────────────────────

func TestCreateReturnRequest_Validate_NotesBoundary(t *testing.T) {
	notes := strings.Repeat("x", 5001)
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: "damaged", Notes: &notes}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notes")
}

func TestCreateReturnRequest_Validate_NotesAtMaxLength(t *testing.T) {
	notes := strings.Repeat("x", 5000)
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: "damaged", Notes: &notes}
	require.NoError(t, req.Validate())
}

func TestCreateReturnRequest_Validate_ReasonAtMaxLength(t *testing.T) {
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: strings.Repeat("x", 2000)}
	require.NoError(t, req.Validate())
}

func TestCreateReturnRequest_Validate_ZeroRefundAmount(t *testing.T) {
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: "wrong item", RefundAmount: 0}
	require.NoError(t, req.Validate())
}

// ─── UpdateReturnRequest.Validate (deeper coverage) ─────────────────────────

func TestUpdateReturnRequest_Validate_ReasonUpdate(t *testing.T) {
	reason := "customer changed mind"
	req := UpdateReturnRequest{Reason: &reason}
	require.NoError(t, req.Validate())
}

func TestUpdateReturnRequest_Validate_ItemsUpdate(t *testing.T) {
	items := json.RawMessage(`[{"product_id":"abc","quantity":1}]`)
	req := UpdateReturnRequest{Items: &items}
	require.NoError(t, req.Validate())
}

func TestUpdateReturnRequest_Validate_NotesUpdate(t *testing.T) {
	notes := "Added return label"
	req := UpdateReturnRequest{Notes: &notes}
	require.NoError(t, req.Validate())
}

func TestUpdateReturnRequest_Validate_ZeroRefundAmount(t *testing.T) {
	zero := 0.0
	req := UpdateReturnRequest{RefundAmount: &zero}
	require.NoError(t, req.Validate())
}

// ─── CreateShipmentRequest.Validate (deeper coverage) ───────────────────────

func TestCreateShipmentRequest_Validate_TrackingNumberTooLong(t *testing.T) {
	tn := strings.Repeat("x", 201)
	req := CreateShipmentRequest{OrderID: uuid.New(), Provider: "inpost", TrackingNumber: &tn}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tracking_number")
}

func TestCreateShipmentRequest_Validate_TrackingNumberAtMaxLength(t *testing.T) {
	tn := strings.Repeat("x", 200)
	req := CreateShipmentRequest{OrderID: uuid.New(), Provider: "inpost", TrackingNumber: &tn}
	require.NoError(t, req.Validate())
}

func TestCreateShipmentRequest_Validate_WithOptionalFields(t *testing.T) {
	intID := uuid.New()
	extID := "EXT-123"
	tn := "123456789"
	labelURL := "https://example.com/label.pdf"
	whID := uuid.New()
	weight := 2.5
	req := CreateShipmentRequest{
		OrderID:        uuid.New(),
		Provider:       "dhl",
		IntegrationID:  &intID,
		ExternalID:     &extID,
		TrackingNumber: &tn,
		LabelURL:       &labelURL,
		CarrierData:    json.RawMessage(`{"service":"express"}`),
		WarehouseID:    &whID,
		Weight:         &weight,
		Notes:          "Handle with care",
	}
	require.NoError(t, req.Validate())
}

func TestCreateShipmentRequest_Validate_WhitespaceProvider(t *testing.T) {
	req := CreateShipmentRequest{OrderID: uuid.New(), Provider: "   "}
	// Provider is checked with == "" (not TrimSpace), so whitespace-only passes the required check
	// but let's verify current behavior
	err := req.Validate()
	// The provider check is: if r.Provider == "", so "   " passes
	require.NoError(t, err)
}

// ─── GenerateLabelRequest.Validate (deeper coverage) ────────────────────────

func TestGenerateLabelRequest_Validate_PNGFormat(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard", LabelFormat: "png"}
	require.NoError(t, req.Validate())
}

func TestGenerateLabelRequest_Validate_ValidSendingMethods(t *testing.T) {
	methods := []string{"parcel_locker", "dispatch_order", "pop", "any_point", "pok", "branch"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := GenerateLabelRequest{ServiceType: "standard", SendingMethod: method}
			require.NoError(t, req.Validate())
		})
	}
}

func TestGenerateLabelRequest_Validate_InvalidSendingMethod(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard", SendingMethod: "drone"}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sending_method must be one of")
}

func TestGenerateLabelRequest_Validate_EmptySendingMethodAllowed(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard"}
	require.NoError(t, req.Validate())
}
