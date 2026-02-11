package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- validateMaxLength / validateMaxLengthPtr ---

func TestValidateMaxLength_OK(t *testing.T) {
	assert.NoError(t, validateMaxLength("name", "short", 100))
}

func TestValidateMaxLength_ExactBoundary(t *testing.T) {
	assert.NoError(t, validateMaxLength("name", strings.Repeat("x", 100), 100))
}

func TestValidateMaxLength_TooLong(t *testing.T) {
	err := validateMaxLength("name", strings.Repeat("x", 101), 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "100")
}

func TestValidateMaxLengthPtr_Nil(t *testing.T) {
	assert.NoError(t, validateMaxLengthPtr("field", nil, 10))
}

func TestValidateMaxLengthPtr_OK(t *testing.T) {
	s := "short"
	assert.NoError(t, validateMaxLengthPtr("field", &s, 100))
}

func TestValidateMaxLengthPtr_TooLong(t *testing.T) {
	s := strings.Repeat("x", 101)
	err := validateMaxLengthPtr("field", &s, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field")
}

// --- StripHTMLTags ---

func TestStripHTMLTags_NoTags(t *testing.T) {
	assert.Equal(t, "hello world", StripHTMLTags("hello world"))
}

func TestStripHTMLTags_SimpleTags(t *testing.T) {
	assert.Equal(t, "hello", StripHTMLTags("<b>hello</b>"))
}

func TestStripHTMLTags_Script(t *testing.T) {
	result := StripHTMLTags(`<script>alert("xss")</script>`)
	assert.Equal(t, `alert("xss")`, result)
}

func TestStripHTMLTags_NestedTags(t *testing.T) {
	assert.Equal(t, "bold italic", StripHTMLTags("<div><b>bold</b> <i>italic</i></div>"))
}

func TestStripHTMLTags_Empty(t *testing.T) {
	assert.Equal(t, "", StripHTMLTags(""))
}

func TestStripHTMLTags_OnlyTags(t *testing.T) {
	assert.Equal(t, "", StripHTMLTags("<br/><hr>"))
}

// --- CreateOrderRequest additional validation ---

func TestCreateOrderRequest_Validate_CustomerNameTooLong(t *testing.T) {
	req := CreateOrderRequest{
		CustomerName: strings.Repeat("x", 501),
		TotalAmount:  10,
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer_name")
}

func TestCreateOrderRequest_Validate_EmailTooLong(t *testing.T) {
	email := strings.Repeat("x", 256)
	req := CreateOrderRequest{
		CustomerName:  "John",
		TotalAmount:   10,
		CustomerEmail: &email,
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer_email")
}

func TestCreateOrderRequest_Validate_PhoneTooLong(t *testing.T) {
	phone := strings.Repeat("x", 51)
	req := CreateOrderRequest{
		CustomerName:  "John",
		TotalAmount:   10,
		CustomerPhone: &phone,
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer_phone")
}

func TestCreateOrderRequest_Validate_NotesTooLong(t *testing.T) {
	notes := strings.Repeat("x", 5001)
	req := CreateOrderRequest{
		CustomerName: "John",
		TotalAmount:  10,
		Notes:        &notes,
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notes")
}

// --- UpdateOrderRequest additional validation ---

func TestUpdateOrderRequest_Validate_CustomerNameTooLong(t *testing.T) {
	longName := strings.Repeat("x", 501)
	req := UpdateOrderRequest{CustomerName: &longName}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer_name")
}

// --- CreateShipmentRequest.Validate ---

func TestCreateShipmentRequest_Validate_Success(t *testing.T) {
	req := CreateShipmentRequest{
		OrderID:  uuid.New(),
		Provider: "inpost",
	}
	assert.NoError(t, req.Validate())
}

func TestCreateShipmentRequest_Validate_MissingOrderID(t *testing.T) {
	req := CreateShipmentRequest{Provider: "inpost"}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order_id")
}

func TestCreateShipmentRequest_Validate_MissingProvider(t *testing.T) {
	req := CreateShipmentRequest{OrderID: uuid.New()}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
}

func TestCreateShipmentRequest_Validate_ProviderTooLong(t *testing.T) {
	req := CreateShipmentRequest{OrderID: uuid.New(), Provider: strings.Repeat("x", 101)}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
}

// --- UpdateShipmentRequest.Validate ---

func TestUpdateShipmentRequest_Validate_Success(t *testing.T) {
	tn := "12345"
	req := UpdateShipmentRequest{TrackingNumber: &tn}
	assert.NoError(t, req.Validate())
}

func TestUpdateShipmentRequest_Validate_NoFields(t *testing.T) {
	req := UpdateShipmentRequest{}
	assert.Error(t, req.Validate())
}

// --- ShipmentStatusTransitionRequest.Validate ---

func TestShipmentStatusTransitionRequest_Validate_Success(t *testing.T) {
	req := ShipmentStatusTransitionRequest{Status: "in_transit"}
	assert.NoError(t, req.Validate())
}

func TestShipmentStatusTransitionRequest_Validate_Empty(t *testing.T) {
	req := ShipmentStatusTransitionRequest{Status: "  "}
	assert.Error(t, req.Validate())
}

// --- GenerateLabelRequest.Validate ---

func TestGenerateLabelRequest_Validate_Success(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "inpost_locker_standard", TargetPoint: "KRA01A", LabelFormat: "pdf"}
	assert.NoError(t, req.Validate())
}

func TestGenerateLabelRequest_Validate_MissingServiceType(t *testing.T) {
	req := GenerateLabelRequest{}
	assert.Error(t, req.Validate())
}

func TestGenerateLabelRequest_Validate_LockerRequiresTargetPoint(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "inpost_locker_standard", LabelFormat: "pdf"}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target_point")
}

func TestGenerateLabelRequest_Validate_InvalidFormat(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard", LabelFormat: "png"}
	assert.Error(t, req.Validate())
}

func TestGenerateLabelRequest_Validate_DefaultFormat(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "pdf", req.LabelFormat)
}

func TestGenerateLabelRequest_Validate_ZPLFormat(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard", LabelFormat: "zpl"}
	assert.NoError(t, req.Validate())
}

func TestGenerateLabelRequest_Validate_EPLFormat(t *testing.T) {
	req := GenerateLabelRequest{ServiceType: "standard", LabelFormat: "epl"}
	assert.NoError(t, req.Validate())
}

// --- CreateIntegrationRequest.Validate ---

func TestCreateIntegrationRequest_Validate_Success(t *testing.T) {
	req := CreateIntegrationRequest{
		Provider:    "allegro",
		Credentials: json.RawMessage(`{"key":"val"}`),
	}
	assert.NoError(t, req.Validate())
}

func TestCreateIntegrationRequest_Validate_MissingProvider(t *testing.T) {
	req := CreateIntegrationRequest{Credentials: json.RawMessage(`{}`)}
	assert.Error(t, req.Validate())
}

func TestCreateIntegrationRequest_Validate_MissingCredentials(t *testing.T) {
	req := CreateIntegrationRequest{Provider: "allegro"}
	assert.Error(t, req.Validate())
}

func TestCreateIntegrationRequest_Validate_ProviderTooLong(t *testing.T) {
	req := CreateIntegrationRequest{
		Provider:    strings.Repeat("x", 101),
		Credentials: json.RawMessage(`{}`),
	}
	assert.Error(t, req.Validate())
}

// --- UpdateIntegrationRequest.Validate ---

func TestUpdateIntegrationRequest_Validate_Success(t *testing.T) {
	status := "active"
	req := UpdateIntegrationRequest{Status: &status}
	assert.NoError(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_NoFields(t *testing.T) {
	req := UpdateIntegrationRequest{}
	assert.Error(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_InvalidStatus(t *testing.T) {
	status := "bogus"
	req := UpdateIntegrationRequest{Status: &status}
	assert.Error(t, req.Validate())
}

func TestUpdateIntegrationRequest_Validate_AllStatuses(t *testing.T) {
	for _, s := range []string{"active", "inactive", "error"} {
		status := s
		req := UpdateIntegrationRequest{Status: &status}
		assert.NoError(t, req.Validate(), "status %s should be valid", s)
	}
}

// --- CreateReturnRequest.Validate ---

func TestCreateReturnRequest_Validate_Success(t *testing.T) {
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: "defective", RefundAmount: 50.0}
	assert.NoError(t, req.Validate())
}

func TestCreateReturnRequest_Validate_MissingOrderID(t *testing.T) {
	req := CreateReturnRequest{Reason: "defective"}
	assert.Error(t, req.Validate())
}

func TestCreateReturnRequest_Validate_MissingReason(t *testing.T) {
	req := CreateReturnRequest{OrderID: uuid.New()}
	assert.Error(t, req.Validate())
}

func TestCreateReturnRequest_Validate_NegativeRefund(t *testing.T) {
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: "defective", RefundAmount: -1}
	assert.Error(t, req.Validate())
}

func TestCreateReturnRequest_Validate_ReasonTooLong(t *testing.T) {
	req := CreateReturnRequest{OrderID: uuid.New(), Reason: strings.Repeat("x", 2001)}
	assert.Error(t, req.Validate())
}

// --- UpdateReturnRequest.Validate ---

func TestUpdateReturnRequest_Validate_Success(t *testing.T) {
	reason := "changed mind"
	req := UpdateReturnRequest{Reason: &reason}
	assert.NoError(t, req.Validate())
}

func TestUpdateReturnRequest_Validate_NoFields(t *testing.T) {
	req := UpdateReturnRequest{}
	assert.Error(t, req.Validate())
}

func TestUpdateReturnRequest_Validate_NegativeRefund(t *testing.T) {
	neg := -1.0
	req := UpdateReturnRequest{RefundAmount: &neg}
	assert.Error(t, req.Validate())
}

// --- IsValidReturnTransition ---

func TestIsValidReturnTransition(t *testing.T) {
	tests := []struct {
		from, to string
		valid    bool
	}{
		{"requested", "approved", true},
		{"requested", "rejected", true},
		{"requested", "cancelled", true},
		{"approved", "received", true},
		{"approved", "cancelled", true},
		{"received", "refunded", true},
		{"received", "cancelled", true},
		{"refunded", "requested", false},
		{"rejected", "approved", false},
		{"nonexistent", "approved", false},
		{"cancelled", "approved", false},
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidReturnTransition(tt.from, tt.to))
		})
	}
}

// --- IsValidFieldType ---

func TestIsValidFieldType(t *testing.T) {
	assert.True(t, IsValidFieldType("text"))
	assert.True(t, IsValidFieldType("number"))
	assert.True(t, IsValidFieldType("select"))
	assert.True(t, IsValidFieldType("date"))
	assert.True(t, IsValidFieldType("checkbox"))
	assert.False(t, IsValidFieldType("invalid"))
	assert.False(t, IsValidFieldType(""))
}

// --- DefaultWebhookConfig ---

func TestDefaultWebhookConfig(t *testing.T) {
	cfg := DefaultWebhookConfig()
	assert.NotNil(t, cfg.Endpoints)
	assert.Empty(t, cfg.Endpoints)
}

// --- DefaultProductCategoriesConfig ---

func TestDefaultProductCategoriesConfig(t *testing.T) {
	cfg := DefaultProductCategoriesConfig()
	assert.NotNil(t, cfg.Categories)
	assert.Empty(t, cfg.Categories)
}

// --- DefaultOrderStatusConfig ---

func TestDefaultOrderStatusConfig_HasStatuses(t *testing.T) {
	cfg := DefaultOrderStatusConfig()
	assert.True(t, len(cfg.Statuses) >= 10)
	assert.True(t, len(cfg.Transitions) >= 10)
}

// --- ColorPresetHex ---

func TestColorPresetHex(t *testing.T) {
	assert.Equal(t, "#3b82f6", ColorPresetHex["blue"])
	assert.Equal(t, "#ef4444", ColorPresetHex["red"])
	_, ok := ColorPresetHex["nonexistent"]
	assert.False(t, ok)
}

// --- CreateProductRequest.Validate ---

func TestCreateProductRequest_Validate_Success(t *testing.T) {
	req := CreateProductRequest{Name: "Widget", Price: 10, StockQty: 5}
	assert.NoError(t, req.Validate())
	assert.Equal(t, "manual", req.Source)
}

func TestCreateProductRequest_Validate_MissingName(t *testing.T) {
	req := CreateProductRequest{Price: 10}
	assert.Error(t, req.Validate())
}

func TestCreateProductRequest_Validate_NegativePrice(t *testing.T) {
	req := CreateProductRequest{Name: "Widget", Price: -1}
	assert.Error(t, req.Validate())
}

func TestCreateProductRequest_Validate_NegativeStock(t *testing.T) {
	req := CreateProductRequest{Name: "Widget", Price: 10, StockQty: -1}
	assert.Error(t, req.Validate())
}

func TestCreateProductRequest_Validate_InvalidSource(t *testing.T) {
	req := CreateProductRequest{Name: "Widget", Price: 10, Source: "ebay"}
	assert.Error(t, req.Validate())
}

func TestCreateProductRequest_Validate_ValidSources(t *testing.T) {
	for _, source := range []string{"allegro", "woocommerce", "manual"} {
		req := CreateProductRequest{Name: "Widget", Price: 10, Source: source}
		assert.NoError(t, req.Validate(), "source %s should be valid", source)
	}
}

// --- UpdateProductRequest.Validate ---

func TestUpdateProductRequest_Validate_NoFields(t *testing.T) {
	req := UpdateProductRequest{}
	assert.Error(t, req.Validate())
}

func TestUpdateProductRequest_Validate_NegativePrice(t *testing.T) {
	neg := -1.0
	req := UpdateProductRequest{Price: &neg}
	assert.Error(t, req.Validate())
}

func TestUpdateProductRequest_Validate_NegativeStock(t *testing.T) {
	neg := -1
	req := UpdateProductRequest{StockQuantity: &neg}
	assert.Error(t, req.Validate())
}

func TestUpdateProductRequest_Validate_InvalidSource(t *testing.T) {
	s := "ebay"
	req := UpdateProductRequest{Source: &s}
	assert.Error(t, req.Validate())
}

// --- CreateSupplierRequest.Validate ---

func TestCreateSupplierRequest_Validate_Success(t *testing.T) {
	req := CreateSupplierRequest{Name: "Supplier A"}
	assert.NoError(t, req.Validate())
	assert.Equal(t, "iof", req.FeedFormat)
}

func TestCreateSupplierRequest_Validate_MissingName(t *testing.T) {
	req := CreateSupplierRequest{}
	assert.Error(t, req.Validate())
}

func TestCreateSupplierRequest_Validate_InvalidFormat(t *testing.T) {
	req := CreateSupplierRequest{Name: "S", FeedFormat: "xml"}
	assert.Error(t, req.Validate())
}

func TestCreateSupplierRequest_Validate_ValidFormats(t *testing.T) {
	for _, fmt := range []string{"iof", "csv", "custom"} {
		req := CreateSupplierRequest{Name: "S", FeedFormat: fmt}
		assert.NoError(t, req.Validate(), "format %s should be valid", fmt)
	}
}

// --- UpdateSupplierRequest.Validate ---

func TestUpdateSupplierRequest_Validate_NoFields(t *testing.T) {
	req := UpdateSupplierRequest{}
	assert.Error(t, req.Validate())
}

func TestUpdateSupplierRequest_Validate_InvalidFormat(t *testing.T) {
	ff := "xml"
	req := UpdateSupplierRequest{FeedFormat: &ff}
	assert.Error(t, req.Validate())
}

func TestUpdateSupplierRequest_Validate_InvalidStatus(t *testing.T) {
	s := "deleted"
	req := UpdateSupplierRequest{Status: &s}
	assert.Error(t, req.Validate())
}

func TestUpdateSupplierRequest_Validate_ValidStatuses(t *testing.T) {
	for _, s := range []string{"active", "inactive"} {
		status := s
		req := UpdateSupplierRequest{Status: &status}
		assert.NoError(t, req.Validate(), "status %s should be valid", s)
	}
}

// --- CreateProductListingRequest.Validate ---

func TestCreateProductListingRequest_Validate_Success(t *testing.T) {
	req := CreateProductListingRequest{ProductID: uuid.New(), IntegrationID: uuid.New()}
	assert.NoError(t, req.Validate())
	assert.Equal(t, "pending", req.Status)
}

func TestCreateProductListingRequest_Validate_MissingProductID(t *testing.T) {
	req := CreateProductListingRequest{IntegrationID: uuid.New()}
	assert.Error(t, req.Validate())
}

func TestCreateProductListingRequest_Validate_MissingIntegrationID(t *testing.T) {
	req := CreateProductListingRequest{ProductID: uuid.New()}
	assert.Error(t, req.Validate())
}

// --- UpdateProductListingRequest.Validate ---

func TestUpdateProductListingRequest_Validate_NoFields(t *testing.T) {
	req := UpdateProductListingRequest{}
	assert.Error(t, req.Validate())
}

func TestUpdateProductListingRequest_Validate_Success(t *testing.T) {
	s := "active"
	req := UpdateProductListingRequest{Status: &s}
	assert.NoError(t, req.Validate())
}

// --- LinkProductRequest.Validate ---

func TestLinkProductRequest_Validate_Success(t *testing.T) {
	req := LinkProductRequest{ProductID: uuid.New()}
	assert.NoError(t, req.Validate())
}

func TestLinkProductRequest_Validate_MissingProductID(t *testing.T) {
	req := LinkProductRequest{}
	assert.Error(t, req.Validate())
}
