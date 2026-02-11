package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type OrderGroup struct {
	ID             uuid.UUID   `json:"id"`
	TenantID       uuid.UUID   `json:"tenant_id"`
	GroupType      string      `json:"group_type"`
	SourceOrderIDs []uuid.UUID `json:"source_order_ids"`
	TargetOrderIDs []uuid.UUID `json:"target_order_ids"`
	Notes          *string     `json:"notes,omitempty"`
	CreatedBy      *uuid.UUID  `json:"created_by,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

type MergeOrdersRequest struct {
	OrderIDs []uuid.UUID `json:"order_ids"`
	Notes    *string     `json:"notes,omitempty"`
}

func (r *MergeOrdersRequest) Validate() error {
	if len(r.OrderIDs) < 2 {
		return errors.New("at least 2 order_ids are required to merge")
	}
	if len(r.OrderIDs) > 50 {
		return errors.New("maximum 50 orders per merge operation")
	}
	return nil
}

type SplitSpec struct {
	Items           json.RawMessage `json:"items"`
	CustomerName    string          `json:"customer_name,omitempty"`
	ShippingAddress json.RawMessage `json:"shipping_address,omitempty"`
}

type SplitOrderRequest struct {
	Splits []SplitSpec `json:"splits"`
	Notes  *string     `json:"notes,omitempty"`
}

func (r *SplitOrderRequest) Validate() error {
	if len(r.Splits) < 2 {
		return errors.New("at least 2 splits are required")
	}
	if len(r.Splits) > 20 {
		return errors.New("maximum 20 splits per operation")
	}
	for i, split := range r.Splits {
		if len(split.Items) == 0 || string(split.Items) == "null" {
			return errors.New("split " + string(rune('1'+i)) + " must have items")
		}
	}
	return nil
}
