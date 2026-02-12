package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID              uuid.UUID        `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	ExternalID      *string          `json:"external_id,omitempty"`
	Source          string           `json:"source"`
	IntegrationID   *uuid.UUID       `json:"integration_id,omitempty"`
	Status          string           `json:"status"`
	CustomerName    string           `json:"customer_name"`
	CustomerEmail   *string          `json:"customer_email,omitempty"`
	CustomerPhone   *string          `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage  `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage  `json:"billing_address,omitempty"`
	Items           json.RawMessage  `json:"items,omitempty"`
	TotalAmount     float64          `json:"total_amount"`
	Currency        string           `json:"currency"`
	Notes           *string          `json:"notes,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
	Tags            []string         `json:"tags"`
	OrderedAt       *time.Time       `json:"ordered_at,omitempty"`
	ShippedAt       *time.Time       `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time       `json:"delivered_at,omitempty"`
	DeliveryMethod  *string          `json:"delivery_method,omitempty"`
	PickupPointID   *string          `json:"pickup_point_id,omitempty"`
	PaymentStatus   string           `json:"payment_status"`
	PaymentMethod   *string          `json:"payment_method,omitempty"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
	CustomerID      *uuid.UUID       `json:"customer_id,omitempty"`
	MergedInto      *uuid.UUID       `json:"merged_into,omitempty"`
	SplitFrom       *uuid.UUID       `json:"split_from,omitempty"`
	InternalNotes   string           `json:"internal_notes"`
	Priority        string           `json:"priority"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CreateOrderRequest struct {
	ExternalID      *string          `json:"external_id,omitempty"`
	Source          string           `json:"source"`
	IntegrationID   *uuid.UUID       `json:"integration_id,omitempty"`
	CustomerName    string           `json:"customer_name"`
	CustomerEmail   *string          `json:"customer_email,omitempty"`
	CustomerPhone   *string          `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage  `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage  `json:"billing_address,omitempty"`
	Items           json.RawMessage  `json:"items,omitempty"`
	TotalAmount     float64          `json:"total_amount"`
	Currency        string           `json:"currency"`
	Notes           *string          `json:"notes,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
	Tags            []string         `json:"tags,omitempty"`
	DeliveryMethod  *string          `json:"delivery_method,omitempty"`
	PickupPointID   *string          `json:"pickup_point_id,omitempty"`
	OrderedAt       *time.Time       `json:"ordered_at,omitempty"`
	PaymentStatus   *string          `json:"payment_status,omitempty"`
	PaymentMethod   *string          `json:"payment_method,omitempty"`
	InternalNotes   string           `json:"internal_notes,omitempty"`
	Priority        string           `json:"priority,omitempty"`
}

var validPriorities = map[string]bool{
	"low": true, "normal": true, "high": true, "urgent": true,
}

func IsValidPriority(p string) bool {
	return validPriorities[p]
}

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
	return nil
}

type UpdateOrderRequest struct {
	ExternalID      *string          `json:"external_id,omitempty"`
	CustomerName    *string          `json:"customer_name,omitempty"`
	CustomerEmail   *string          `json:"customer_email,omitempty"`
	CustomerPhone   *string          `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage  `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage  `json:"billing_address,omitempty"`
	Items           json.RawMessage  `json:"items,omitempty"`
	TotalAmount     *float64         `json:"total_amount,omitempty"`
	Currency        *string          `json:"currency,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
	Tags            *[]string        `json:"tags,omitempty"`
	DeliveryMethod  *string          `json:"delivery_method,omitempty"`
	PickupPointID   *string          `json:"pickup_point_id,omitempty"`
	PaymentStatus   *string          `json:"payment_status,omitempty"`
	PaymentMethod   *string          `json:"payment_method,omitempty"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
	InternalNotes   *string          `json:"internal_notes,omitempty"`
	Priority        *string          `json:"priority,omitempty"`
}

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

type StatusTransitionRequest struct {
	Status string `json:"status"`
	Force  bool   `json:"force,omitempty"`
}

func (r *StatusTransitionRequest) Validate() error {
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

type OrderListFilter struct {
	Status        *string
	Source        *string
	Search        *string
	PaymentStatus *string
	Tag           *string
	Priority      *string
	PaginationParams
}

type BulkStatusTransitionRequest struct {
	OrderIDs []uuid.UUID `json:"order_ids"`
	Status   string      `json:"status"`
	Force    bool        `json:"force,omitempty"`
}

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

type BulkStatusResult struct {
	OrderID uuid.UUID `json:"order_id"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

type BulkStatusTransitionResponse struct {
	Results   []BulkStatusResult `json:"results"`
	Succeeded int                `json:"succeeded"`
	Failed    int                `json:"failed"`
}

// --- Custom Order Statuses ---

type StatusDef struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

type OrderStatusConfig struct {
	Statuses    []StatusDef         `json:"statuses"`
	Transitions map[string][]string `json:"transitions"`
}

func (c *OrderStatusConfig) IsValidStatus(key string) bool {
	for _, s := range c.Statuses {
		if s.Key == key {
			return true
		}
	}
	return false
}

func (c *OrderStatusConfig) CanTransition(from, to string) bool {
	targets, ok := c.Transitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

func (c *OrderStatusConfig) GetStatusDef(key string) *StatusDef {
	for _, s := range c.Statuses {
		if s.Key == key {
			return &s
		}
	}
	return nil
}

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

type CustomFieldDef struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // "text", "number", "select", "date", "checkbox"
	Required bool     `json:"required"`
	Position int      `json:"position"`
	Options  []string `json:"options,omitempty"` // only for type="select"
}

type CustomFieldsConfig struct {
	Fields []CustomFieldDef `json:"fields"`
}

var validFieldTypes = map[string]bool{
	"text": true, "number": true, "select": true, "date": true, "checkbox": true,
}

func IsValidFieldType(t string) bool {
	return validFieldTypes[t]
}

// --- Product Categories ---

type CategoryDef struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

type ProductCategoriesConfig struct {
	Categories []CategoryDef `json:"categories"`
}

func DefaultProductCategoriesConfig() ProductCategoriesConfig {
	return ProductCategoriesConfig{
		Categories: []CategoryDef{},
	}
}

func DefaultOrderStatusConfig() OrderStatusConfig {
	return OrderStatusConfig{
		Statuses: []StatusDef{
			{Key: "new", Label: "Nowe", Color: "blue", Position: 1},
			{Key: "confirmed", Label: "Potwierdzone", Color: "indigo", Position: 2},
			{Key: "processing", Label: "W realizacji", Color: "yellow", Position: 3},
			{Key: "ready_to_ship", Label: "Gotowe do wysyłki", Color: "orange", Position: 4},
			{Key: "shipped", Label: "Wysłane", Color: "purple", Position: 5},
			{Key: "in_transit", Label: "W transporcie", Color: "purple", Position: 6},
			{Key: "out_for_delivery", Label: "W doręczeniu", Color: "teal", Position: 7},
			{Key: "delivered", Label: "Dostarczone", Color: "green", Position: 8},
			{Key: "completed", Label: "Zakończone", Color: "green-dark", Position: 9},
			{Key: "on_hold", Label: "Wstrzymane", Color: "gray", Position: 10},
			{Key: "cancelled", Label: "Anulowane", Color: "red", Position: 11},
			{Key: "refunded", Label: "Zwrócone", Color: "red-dark", Position: 12},
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
