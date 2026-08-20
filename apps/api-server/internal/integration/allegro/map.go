package allegro

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ToOMSOrder maps a normalized Allegro checkout-form onto an OMS order row.
// Source is always allegro; ExternalID is the checkout-form id.
func ToOMSOrder(mo integration.MarketplaceOrder, tenantID, integrationID uuid.UUID) model.Order {
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", integrationID)
	order := model.Order{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ExternalID:    req.ExternalID,
		Source:        req.Source,
		IntegrationID: req.IntegrationID,
		Status:        "new",
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		TotalAmount:   req.TotalAmount,
		Currency:      req.Currency,
		OrderedAt:     req.OrderedAt,
		PaymentMethod: req.PaymentMethod,
	}

	if req.PaymentStatus != nil {
		order.PaymentStatus = *req.PaymentStatus
	} else {
		order.PaymentStatus = "pending"
	}

	if addrJSON, err := json.Marshal(mo.ShippingAddress); err == nil {
		order.ShippingAddress = addrJSON
	}
	if itemsJSON, err := json.Marshal(mo.Items); err == nil {
		order.Items = itemsJSON
	}

	if (order.CustomerPhone == nil || *order.CustomerPhone == "") && mo.ShippingAddress.Phone != "" {
		order.CustomerPhone = &mo.ShippingAddress.Phone
	}

	if mo.RawData != nil {
		if dmName, ok := mo.RawData["delivery_method_name"].(string); ok {
			order.DeliveryMethod = &dmName
		}
		if ppID, ok := mo.RawData["pickup_point_id"].(string); ok {
			order.PickupPointID = &ppID
		}
	}

	metadata := map[string]any{"external_id": mo.ExternalID}
	if mo.RawData != nil {
		if dmID, ok := mo.RawData["delivery_method_id"].(string); ok && dmID != "" {
			metadata["delivery_method_id"] = dmID
		}
		if dmName, ok := mo.RawData["delivery_method_name"].(string); ok && dmName != "" {
			metadata["delivery_method_name"] = dmName
		}
	}
	if metadataJSON, err := json.Marshal(metadata); err == nil {
		order.Metadata = metadataJSON
	}
	order.Tags = []string{}
	return order
}
