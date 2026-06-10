package model

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Order represents a customer order in the system.
type Order struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	ExternalID      *string         `json:"external_id,omitempty"`
	Source          string          `json:"source"`
	IntegrationID   *uuid.UUID      `json:"integration_id,omitempty"`
	Status          string          `json:"status"`
	CustomerName    string          `json:"customer_name"`
	CustomerEmail   *string         `json:"customer_email,omitempty"`
	CustomerPhone   *string         `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage `json:"billing_address,omitempty"`
	Items           json.RawMessage `json:"items,omitempty"`
	TotalAmount     float64         `json:"total_amount"`
	Currency        string          `json:"currency"`
	Notes           *string         `json:"notes,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Tags            []string        `json:"tags"`
	OrderedAt       *time.Time      `json:"ordered_at,omitempty"`
	ShippedAt       *time.Time      `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time      `json:"delivered_at,omitempty"`
	DeliveryMethod  *string         `json:"delivery_method,omitempty"`
	PickupPointID   *string         `json:"pickup_point_id,omitempty"`
	PaymentStatus   string          `json:"payment_status"`
	PaymentMethod   *string         `json:"payment_method,omitempty"`
	PaidAt          *time.Time      `json:"paid_at,omitempty"`
	CustomerID      *uuid.UUID      `json:"customer_id,omitempty"`
	MergedInto      *uuid.UUID      `json:"merged_into,omitempty"`
	SplitFrom       *uuid.UUID      `json:"split_from,omitempty"`
	InternalNotes   string          `json:"internal_notes"`
	Priority        string          `json:"priority"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// CreateOrderRequest is the payload for creating a new order.
type CreateOrderRequest struct {
	ExternalID      *string         `json:"external_id,omitempty"`
	Source          string          `json:"source"`
	IntegrationID   *uuid.UUID      `json:"integration_id,omitempty"`
	CustomerName    string          `json:"customer_name"`
	CustomerEmail   *string         `json:"customer_email,omitempty"`
	CustomerPhone   *string         `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage `json:"billing_address,omitempty"`
	Items           json.RawMessage `json:"items,omitempty"`
	TotalAmount     float64         `json:"total_amount"`
	Currency        string          `json:"currency"`
	Notes           *string         `json:"notes,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	DeliveryMethod  *string         `json:"delivery_method,omitempty"`
	PickupPointID   *string         `json:"pickup_point_id,omitempty"`
	OrderedAt       *time.Time      `json:"ordered_at,omitempty"`
	PaymentStatus   *string         `json:"payment_status,omitempty"`
	PaymentMethod   *string         `json:"payment_method,omitempty"`
	InternalNotes   string          `json:"internal_notes,omitempty"`
	Priority        string          `json:"priority,omitempty"`

	// Transient: trigger shipment auto-creation (not persisted on order)
	ShipmentProvider   *string `json:"shipment_provider,omitempty"`
	AutoCreateShipment bool    `json:"auto_create_shipment,omitempty"`

	// Transient: plan limit injected by handler, not from JSON
	MaxOrdersMonthly int `json:"-"`
}

var validPriorities = map[string]bool{
	"low": true, "normal": true, "high": true, "urgent": true,
}

// Order status keys for the built-in lifecycle (see DefaultOrderStatusConfig).
const (
	OrderStatusNew            = "new"
	OrderStatusConfirmed      = "confirmed"
	OrderStatusProcessing     = "processing"
	OrderStatusReadyToShip    = "ready_to_ship"
	OrderStatusShipped        = "shipped"
	OrderStatusInTransit      = "in_transit"
	OrderStatusOutForDelivery = "out_for_delivery"
	OrderStatusDelivered      = "delivered"
	OrderStatusCompleted      = "completed"
	OrderStatusOnHold         = "on_hold"
	OrderStatusCancelled      = "cancelled"
	OrderStatusRefunded       = "refunded"
)

// TerminalOrderStatuses are the order statuses from which the lifecycle no longer
// advances: the order is finished (completed) or closed out (cancelled/refunded).
// Non-terminal orders are the ones still flowing through fulfillment, so they are
// the set eligible for fulfillment-process backfill (OPE-423a). Per
// DefaultOrderStatusConfig, completed and cancelled each have a single onward edge
// to refunded and refunded is fully terminal; all three are excluded from backfill
// because none represents an order still flowing through fulfillment.
var TerminalOrderStatuses = []string{
	OrderStatusCompleted,
	OrderStatusCancelled,
	OrderStatusRefunded,
}

// IsTerminalOrderStatus reports whether status is a terminal order status (the
// lifecycle does not advance further). Used to exclude finished/closed orders from
// fulfillment-process backfill.
func IsTerminalOrderStatus(status string) bool {
	return slices.Contains(TerminalOrderStatuses, status)
}

// IsValidPriority reports whether p is a recognised order priority string.
func IsValidPriority(p string) bool {
	return validPriorities[p]
}

// Validate validates the create order request.
func (r *CreateOrderRequest) Validate() error {
	if strings.TrimSpace(r.CustomerName) == "" {
		return errors.New("customer_name is required")
	}
	if r.Source == "" {
		r.Source = "manual"
	}
	if r.TotalAmount < 0 {
		return errors.New("total_amount must be non-negative")
	}
	if r.Currency == "" {
		r.Currency = "PLN"
	}
	if err := validateMaxLength("customer_name", r.CustomerName, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("customer_email", r.CustomerEmail, 255); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("customer_phone", r.CustomerPhone, 50); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	if err := validateMaxLength("source", r.Source, 100); err != nil {
		return err
	}
	if err := validateMaxLength("currency", r.Currency, 10); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("payment_status", r.PaymentStatus, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("payment_method", r.PaymentMethod, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("delivery_method", r.DeliveryMethod, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("pickup_point_id", r.PickupPointID, 100); err != nil {
		return err
	}
	if r.Priority == "" {
		r.Priority = "normal"
	}
	if !IsValidPriority(r.Priority) {
		return errors.New("priority must be one of: low, normal, high, urgent")
	}
	if r.AutoCreateShipment && (r.ShipmentProvider == nil || *r.ShipmentProvider == "") {
		return errors.New("shipment_provider is required when auto_create_shipment is true")
	}
	return nil
}

// UpdateOrderRequest is the payload for updating an existing order.
type UpdateOrderRequest struct {
	ExternalID      *string         `json:"external_id,omitempty"`
	CustomerName    *string         `json:"customer_name,omitempty"`
	CustomerEmail   *string         `json:"customer_email,omitempty"`
	CustomerPhone   *string         `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage `json:"billing_address,omitempty"`
	Items           json.RawMessage `json:"items,omitempty"`
	TotalAmount     *float64        `json:"total_amount,omitempty"`
	Currency        *string         `json:"currency,omitempty"`
	Notes           *string         `json:"notes,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Tags            *[]string       `json:"tags,omitempty"`
	DeliveryMethod  *string         `json:"delivery_method,omitempty"`
	PickupPointID   *string         `json:"pickup_point_id,omitempty"`
	PaymentStatus   *string         `json:"payment_status,omitempty"`
	PaymentMethod   *string         `json:"payment_method,omitempty"`
	PaidAt          *time.Time      `json:"paid_at,omitempty"`
	InternalNotes   *string         `json:"internal_notes,omitempty"`
	Priority        *string         `json:"priority,omitempty"`
}

// Validate validates the update order request.
func (r *UpdateOrderRequest) Validate() error {
	if r.ExternalID == nil && r.CustomerName == nil && r.CustomerEmail == nil &&
		r.CustomerPhone == nil && r.ShippingAddress == nil && r.BillingAddress == nil &&
		r.Items == nil && r.TotalAmount == nil && r.Currency == nil &&
		r.Notes == nil && r.Metadata == nil && r.Tags == nil &&
		r.DeliveryMethod == nil && r.PickupPointID == nil &&
		r.PaymentStatus == nil && r.PaymentMethod == nil && r.PaidAt == nil &&
		r.InternalNotes == nil && r.Priority == nil {
		return errors.New("at least one field must be provided")
	}
	if r.TotalAmount != nil && *r.TotalAmount < 0 {
		return errors.New("total_amount must be non-negative")
	}
	if err := validateMaxLengthPtr("customer_name", r.CustomerName, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("customer_email", r.CustomerEmail, 255); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("customer_phone", r.CustomerPhone, 50); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("currency", r.Currency, 10); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("payment_status", r.PaymentStatus, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("payment_method", r.PaymentMethod, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("delivery_method", r.DeliveryMethod, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("pickup_point_id", r.PickupPointID, 100); err != nil {
		return err
	}
	if r.Priority != nil && !IsValidPriority(*r.Priority) {
		return errors.New("priority must be one of: low, normal, high, urgent")
	}
	return nil
}

// StatusTransitionRequest is the payload for transitioning a single order status.
type StatusTransitionRequest struct {
	Status string `json:"status"`
	Force  bool   `json:"force,omitempty"`
}

// Validate validates the status transition request.
func (r *StatusTransitionRequest) Validate() error {
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

// OrderListFilter holds query parameters for listing orders.
type OrderListFilter struct {
	Status        *string
	Source        *string
	Search        *string
	PaymentStatus *string
	Tag           *string
	Priority      *string
	PaginationParams
}

// BulkStatusTransitionRequest is the payload for bulk-transitioning multiple order statuses.
type BulkStatusTransitionRequest struct {
	OrderIDs []uuid.UUID `json:"order_ids"`
	Status   string      `json:"status"`
	Force    bool        `json:"force,omitempty"`
}

// Validate validates the bulk status transition request.
func (r *BulkStatusTransitionRequest) Validate() error {
	if len(r.OrderIDs) == 0 {
		return errors.New("at least one order_id is required")
	}
	if len(r.OrderIDs) > 100 {
		return errors.New("maximum 100 orders per bulk operation")
	}
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

// BulkStatusResult holds the outcome of a single order in a bulk status transition.
type BulkStatusResult struct {
	OrderID uuid.UUID `json:"order_id"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

// BulkStatusTransitionResponse is returned after a bulk status transition operation.
type BulkStatusTransitionResponse struct {
	Results       []BulkStatusResult `json:"results"`
	Succeeded     int                `json:"succeeded"`
	Failed        int                `json:"failed"`
	AuditFailures []string           `json:"audit_failures,omitempty"`
}

// --- Custom Order Statuses ---

// StatusDef defines a single order status with its display properties.
type StatusDef struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

// OrderStatusConfig holds the custom order statuses and allowed transitions for a tenant.
type OrderStatusConfig struct {
	Statuses    []StatusDef         `json:"statuses"`
	Transitions map[string][]string `json:"transitions"`
}

// IsValidStatus reports whether the given status key exists in the config.
func (c *OrderStatusConfig) IsValidStatus(key string) bool {
	for _, s := range c.Statuses {
		if s.Key == key {
			return true
		}
	}
	return false
}

// CanTransition reports whether a transition from one status to another is allowed.
func (c *OrderStatusConfig) CanTransition(from, to string) bool {
	targets, ok := c.Transitions[from]
	if !ok {
		return false
	}
	return slices.Contains(targets, to)
}

// GetStatusDef returns the StatusDef for the given key, or nil if not found.
func (c *OrderStatusConfig) GetStatusDef(key string) *StatusDef {
	for _, s := range c.Statuses {
		if s.Key == key {
			return &s
		}
	}
	return nil
}

// ColorPresetHex maps color preset names to their hex values.
var ColorPresetHex = map[string]string{
	"blue":       "#3b82f6",
	"indigo":     "#6366f1",
	"yellow":     "#eab308",
	"orange":     "#f97316",
	"purple":     "#a855f7",
	"teal":       "#14b8a6",
	"green":      "#22c55e",
	"green-dark": "#16a34a",
	"gray":       "#6b7280",
	"red":        "#ef4444",
	"red-dark":   "#dc2626",
}

// --- Custom Fields on Orders ---

// CustomFieldDef defines a single custom field on orders.
type CustomFieldDef struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // "text", "number", "select", "date", "checkbox"
	Required bool     `json:"required"`
	Position int      `json:"position"`
	Options  []string `json:"options,omitempty"` // only for type="select"
}

// CustomFieldsConfig holds the tenant-level custom fields configuration for orders.
type CustomFieldsConfig struct {
	Fields []CustomFieldDef `json:"fields"`
}

var validFieldTypes = map[string]bool{
	"text": true, "number": true, "select": true, "date": true, "checkbox": true,
}

// IsValidFieldType reports whether t is a valid custom field type.
func IsValidFieldType(t string) bool {
	return validFieldTypes[t]
}

// --- Product Categories ---

// CategoryDef defines a single product category with display properties.
type CategoryDef struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

// ProductCategoriesConfig holds the tenant-level product categories configuration.
type ProductCategoriesConfig struct {
	Categories []CategoryDef `json:"categories"`
}

// DefaultProductCategoriesConfig returns an empty product categories configuration.
func DefaultProductCategoriesConfig() ProductCategoriesConfig {
	return ProductCategoriesConfig{
		Categories: []CategoryDef{},
	}
}

// DefaultOrderStatusConfig returns the default order statuses and transitions.
func DefaultOrderStatusConfig() OrderStatusConfig {
	return OrderStatusConfig{
		Statuses: []StatusDef{
			{Key: "new", Label: "New", Color: "blue", Position: 1},
			{Key: "confirmed", Label: "Confirmed", Color: "indigo", Position: 2},
			{Key: "processing", Label: "Processing", Color: "yellow", Position: 3},
			{Key: "ready_to_ship", Label: "Ready to Ship", Color: "orange", Position: 4},
			{Key: "shipped", Label: "Shipped", Color: "purple", Position: 5},
			{Key: "in_transit", Label: "In Transit", Color: "purple", Position: 6},
			{Key: "out_for_delivery", Label: "Out for Delivery", Color: "teal", Position: 7},
			{Key: "delivered", Label: "Delivered", Color: "green", Position: 8},
			{Key: "completed", Label: "Completed", Color: "green-dark", Position: 9},
			{Key: "on_hold", Label: "On Hold", Color: "gray", Position: 10},
			{Key: "cancelled", Label: "Cancelled", Color: "red", Position: 11},
			{Key: "refunded", Label: "Refunded", Color: "red-dark", Position: 12},
		},
		Transitions: map[string][]string{
			"new":              {"confirmed", "cancelled", "on_hold"},
			"confirmed":        {"processing", "cancelled", "on_hold"},
			"processing":       {"ready_to_ship", "cancelled", "on_hold"},
			"ready_to_ship":    {"shipped", "cancelled", "on_hold"},
			"shipped":          {"in_transit", "delivered", "refunded"},
			"in_transit":       {"out_for_delivery", "delivered", "refunded"},
			"out_for_delivery": {"delivered", "refunded"},
			"delivered":        {"completed", "refunded"},
			"completed":        {"refunded"},
			"on_hold":          {"confirmed", "processing", "cancelled"},
			"cancelled":        {"refunded"},
			"refunded":         {},
		},
	}
}
