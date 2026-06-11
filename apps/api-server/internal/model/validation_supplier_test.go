package model

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── CreateSupplierRequest.Validate ─────────────────────────────────────────

func TestCreateSupplierRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateSupplierRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateSupplierRequest{Name: "Supplier A"},
		},
		{
			name: "valid with all fields",
			req: func() CreateSupplierRequest {
				code := "SUP-01"
				feedURL := "https://example.com/feed.xml"
				interval := 60
				intID := uuid.New()
				return CreateSupplierRequest{
					Name:                "Supplier A",
					Code:                &code,
					FeedURL:             &feedURL,
					FeedFormat:          "csv",
					SyncIntervalMinutes: &interval,
					IntegrationID:       &intID,
				}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateSupplierRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateSupplierRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateSupplierRequest{Name: strings.Repeat("x", 501)},
			wantErr: "name",
		},
		{
			name: "name at max length",
			req:  CreateSupplierRequest{Name: strings.Repeat("x", 500)},
		},
		{
			name:    "invalid feed_format",
			req:     CreateSupplierRequest{Name: "S", FeedFormat: "yaml"},
			wantErr: "feed_format must be one of",
		},
		{
			name: "sync_interval_minutes too low",
			req: func() CreateSupplierRequest {
				i := 4
				return CreateSupplierRequest{Name: "S", SyncIntervalMinutes: &i}
			}(),
			wantErr: "sync_interval_minutes must be between 5 and 1440",
		},
		{
			name: "sync_interval_minutes too high",
			req: func() CreateSupplierRequest {
				i := 1441
				return CreateSupplierRequest{Name: "S", SyncIntervalMinutes: &i}
			}(),
			wantErr: "sync_interval_minutes must be between 5 and 1440",
		},
		{
			name: "sync_interval_minutes at lower bound",
			req: func() CreateSupplierRequest {
				i := 5
				return CreateSupplierRequest{Name: "S", SyncIntervalMinutes: &i}
			}(),
		},
		{
			name: "sync_interval_minutes at upper bound",
			req: func() CreateSupplierRequest {
				i := 1440
				return CreateSupplierRequest{Name: "S", SyncIntervalMinutes: &i}
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

func TestCreateSupplierRequest_Validate_DefaultFeedFormat(t *testing.T) {
	req := CreateSupplierRequest{Name: "Supplier"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "iof", req.FeedFormat, "feed_format should default to 'iof'")
}

func TestCreateSupplierRequest_Validate_AllFeedFormats(t *testing.T) {
	for _, format := range []string{"iof", "csv", "custom", "btp", "xml"} {
		t.Run(format, func(t *testing.T) {
			req := CreateSupplierRequest{Name: "S", FeedFormat: format}
			require.NoError(t, req.Validate())
		})
	}
}

// ─── UpdateSupplierRequest.Validate ─────────────────────────────────────────

func TestUpdateSupplierRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateSupplierRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateSupplierRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req: func() UpdateSupplierRequest {
				name := "Updated"
				return UpdateSupplierRequest{Name: &name}
			}(),
		},
		{
			name: "valid feed_format update",
			req: func() UpdateSupplierRequest {
				f := "csv"
				return UpdateSupplierRequest{FeedFormat: &f}
			}(),
		},
		{
			name: "invalid feed_format",
			req: func() UpdateSupplierRequest {
				f := "yaml"
				return UpdateSupplierRequest{FeedFormat: &f}
			}(),
			wantErr: "feed_format must be one of",
		},
		{
			name: "valid status active",
			req: func() UpdateSupplierRequest {
				s := "active"
				return UpdateSupplierRequest{Status: &s}
			}(),
		},
		{
			name: "valid status inactive",
			req: func() UpdateSupplierRequest {
				s := "inactive"
				return UpdateSupplierRequest{Status: &s}
			}(),
		},
		{
			name: "invalid status",
			req: func() UpdateSupplierRequest {
				s := "archived"
				return UpdateSupplierRequest{Status: &s}
			}(),
			wantErr: "status must be one of",
		},
		{
			name: "sync_interval_minutes too low",
			req: func() UpdateSupplierRequest {
				i := 4
				return UpdateSupplierRequest{SyncIntervalMinutes: &i}
			}(),
			wantErr: "sync_interval_minutes must be between 5 and 1440",
		},
		{
			name: "sync_interval_minutes too high",
			req: func() UpdateSupplierRequest {
				i := 1441
				return UpdateSupplierRequest{SyncIntervalMinutes: &i}
			}(),
			wantErr: "sync_interval_minutes must be between 5 and 1440",
		},
		{
			name: "valid portal_enabled update",
			req: func() UpdateSupplierRequest {
				b := true
				return UpdateSupplierRequest{PortalEnabled: &b}
			}(),
		},
		{
			name: "valid code update",
			req: func() UpdateSupplierRequest {
				c := "SUP-02"
				return UpdateSupplierRequest{Code: &c}
			}(),
		},
		{
			name: "valid feed_url update",
			req: func() UpdateSupplierRequest {
				u := "https://example.com/new-feed.xml"
				return UpdateSupplierRequest{FeedURL: &u}
			}(),
		},
		{
			name: "valid error_message update",
			req: func() UpdateSupplierRequest {
				e := "timeout"
				return UpdateSupplierRequest{ErrorMessage: &e}
			}(),
		},
		{
			name: "valid integration_id update",
			req: func() UpdateSupplierRequest {
				id := uuid.New()
				return UpdateSupplierRequest{IntegrationID: &id}
			}(),
		},
		{
			name: "valid default_category_id update",
			req: func() UpdateSupplierRequest {
				id := uuid.New()
				return UpdateSupplierRequest{DefaultCategoryID: &id}
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

func TestUpdateSupplierRequest_Validate_AllFeedFormats(t *testing.T) {
	for _, format := range []string{"iof", "csv", "custom", "btp", "xml"} {
		t.Run(format, func(t *testing.T) {
			f := format
			req := UpdateSupplierRequest{FeedFormat: &f}
			require.NoError(t, req.Validate())
		})
	}
}

// ─── ImportSupplierProductsRequest.Validate ──────────────────────────────────

func TestImportSupplierProductsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ImportSupplierProductsRequest
		wantErr string
	}{
		{
			name: "valid single ID",
			req:  ImportSupplierProductsRequest{SupplierProductIDs: []uuid.UUID{uuid.New()}},
		},
		{
			name: "valid 500 IDs",
			req: func() ImportSupplierProductsRequest {
				ids := make([]uuid.UUID, 500)
				for i := range ids {
					ids[i] = uuid.New()
				}
				return ImportSupplierProductsRequest{SupplierProductIDs: ids}
			}(),
		},
		{
			name:    "empty IDs",
			req:     ImportSupplierProductsRequest{},
			wantErr: "supplier_product_ids is required",
		},
		{
			name:    "nil IDs",
			req:     ImportSupplierProductsRequest{SupplierProductIDs: nil},
			wantErr: "supplier_product_ids is required",
		},
		{
			name: "too many IDs",
			req: func() ImportSupplierProductsRequest {
				ids := make([]uuid.UUID, 501)
				for i := range ids {
					ids[i] = uuid.New()
				}
				return ImportSupplierProductsRequest{SupplierProductIDs: ids}
			}(),
			wantErr: "max 500 products per import",
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

// ─── UpsertCategoryMappingRequest.Validate ──────────────────────────────────

func TestUpsertCategoryMappingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpsertCategoryMappingRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  UpsertCategoryMappingRequest{SourceCategory: "Electronics"},
		},
		{
			name: "valid with category_id",
			req: func() UpsertCategoryMappingRequest {
				id := uuid.New()
				return UpsertCategoryMappingRequest{SourceCategory: "Electronics", CategoryID: &id, Confirmed: true}
			}(),
		},
		{
			name:    "missing source_category",
			req:     UpsertCategoryMappingRequest{},
			wantErr: "source_category is required",
		},
		{
			name:    "whitespace-only source_category",
			req:     UpsertCategoryMappingRequest{SourceCategory: "   "},
			wantErr: "source_category is required",
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

// ─── BulkDeleteSupplierProductsRequest.Validate ─────────────────────────────

func TestBulkDeleteSupplierProductsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     BulkDeleteSupplierProductsRequest
		wantErr string
	}{
		{
			name: "valid single ID",
			req:  BulkDeleteSupplierProductsRequest{SupplierProductIDs: []uuid.UUID{uuid.New()}},
		},
		{
			name: "valid 500 IDs",
			req: func() BulkDeleteSupplierProductsRequest {
				ids := make([]uuid.UUID, 500)
				for i := range ids {
					ids[i] = uuid.New()
				}
				return BulkDeleteSupplierProductsRequest{SupplierProductIDs: ids}
			}(),
		},
		{
			name:    "empty IDs",
			req:     BulkDeleteSupplierProductsRequest{},
			wantErr: "supplier_product_ids is required",
		},
		{
			name: "too many IDs",
			req: func() BulkDeleteSupplierProductsRequest {
				ids := make([]uuid.UUID, 501)
				for i := range ids {
					ids[i] = uuid.New()
				}
				return BulkDeleteSupplierProductsRequest{SupplierProductIDs: ids}
			}(),
			wantErr: "max 500 products per bulk delete",
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

// ─── LinkProductRequest.Validate ────────────────────────────────────────────

func TestLinkProductRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     LinkProductRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  LinkProductRequest{ProductID: uuid.New()},
		},
		{
			name:    "missing product_id",
			req:     LinkProductRequest{},
			wantErr: "product_id is required",
		},
		{
			name:    "nil product_id",
			req:     LinkProductRequest{ProductID: uuid.Nil},
			wantErr: "product_id is required",
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

// ─── BTPWizardStartImportRequest.Validate ───────────────────────────────────

func TestBTPWizardStartImportRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     BTPWizardStartImportRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  BTPWizardStartImportRequest{Name: "BTP Supplier", FeedURL: "https://example.com/btp.xml"},
		},
		{
			name:    "missing name",
			req:     BTPWizardStartImportRequest{FeedURL: "https://example.com/btp.xml"},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     BTPWizardStartImportRequest{Name: "   ", FeedURL: "https://example.com/btp.xml"},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     BTPWizardStartImportRequest{Name: strings.Repeat("x", 501), FeedURL: "https://example.com/btp.xml"},
			wantErr: "name",
		},
		{
			name: "name at max length",
			req:  BTPWizardStartImportRequest{Name: strings.Repeat("x", 500), FeedURL: "https://example.com/btp.xml"},
		},
		{
			name:    "missing feed_url",
			req:     BTPWizardStartImportRequest{Name: "BTP Supplier"},
			wantErr: "feed_url is required",
		},
		{
			name:    "whitespace-only feed_url",
			req:     BTPWizardStartImportRequest{Name: "BTP Supplier", FeedURL: "   "},
			wantErr: "feed_url is required",
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

// ─── BTPWizardSetAPIKeysRequest.Validate ────────────────────────────────────

func TestBTPWizardSetAPIKeysRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     BTPWizardSetAPIKeysRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  BTPWizardSetAPIKeysRequest{PublicKey: "pk_test_123", PrivateKey: "sk_test_456"},
		},
		{
			name: "valid with base_url",
			req:  BTPWizardSetAPIKeysRequest{PublicKey: "pk_test_123", PrivateKey: "sk_test_456", BaseURL: "https://api.btp.com"},
		},
		{
			name:    "missing public_key",
			req:     BTPWizardSetAPIKeysRequest{PrivateKey: "sk_test_456"},
			wantErr: "public_key is required",
		},
		{
			name:    "whitespace-only public_key",
			req:     BTPWizardSetAPIKeysRequest{PublicKey: "   ", PrivateKey: "sk_test_456"},
			wantErr: "public_key is required",
		},
		{
			name:    "missing private_key",
			req:     BTPWizardSetAPIKeysRequest{PublicKey: "pk_test_123"},
			wantErr: "private_key is required",
		},
		{
			name:    "whitespace-only private_key",
			req:     BTPWizardSetAPIKeysRequest{PublicKey: "pk_test_123", PrivateKey: "   "},
			wantErr: "private_key is required",
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

// ─── BTPWizardSyncSettingsRequest.Validate ──────────────────────────────────

func TestBTPWizardSyncSettingsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     BTPWizardSyncSettingsRequest
		wantErr string
	}{
		{
			name: "valid 12h / 15min",
			req:  BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 12, APISyncIntervalMin: 15},
		},
		{
			name: "valid 24h / 30min",
			req:  BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 24, APISyncIntervalMin: 30},
		},
		{
			name: "valid 48h / 60min",
			req:  BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 48, APISyncIntervalMin: 60},
		},
		{
			name:    "invalid xml_sync_interval_hours",
			req:     BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 6, APISyncIntervalMin: 15},
			wantErr: "xml_sync_interval_hours must be 12, 24, or 48",
		},
		{
			name:    "zero xml_sync_interval_hours",
			req:     BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 0, APISyncIntervalMin: 15},
			wantErr: "xml_sync_interval_hours must be 12, 24, or 48",
		},
		{
			name:    "invalid api_sync_interval_minutes",
			req:     BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 12, APISyncIntervalMin: 45},
			wantErr: "api_sync_interval_minutes must be 15, 30, or 60",
		},
		{
			name:    "zero api_sync_interval_minutes",
			req:     BTPWizardSyncSettingsRequest{XMLSyncIntervalHours: 12, APISyncIntervalMin: 0},
			wantErr: "api_sync_interval_minutes must be 15, 30, or 60",
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

// ─── BulkUpsertAllegroMappingsRequest.Validate ──────────────────────────────

func TestBulkUpsertAllegroMappingsRequest_Validate(t *testing.T) {
	validMapping := UpsertAllegroParameterMappingRequest{
		AllegroParamID: "12345",
		SourceType:     "attribute",
		SourceKey:      "color",
	}

	tests := []struct {
		name    string
		req     BulkUpsertAllegroMappingsRequest
		wantErr string
	}{
		{
			name: "valid single mapping",
			req: BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings:          []UpsertAllegroParameterMappingRequest{validMapping},
			},
		},
		{
			name:    "missing allegro_category_id",
			req:     BulkUpsertAllegroMappingsRequest{Mappings: []UpsertAllegroParameterMappingRequest{validMapping}},
			wantErr: "allegro_category_id is required",
		},
		{
			name:    "whitespace-only allegro_category_id",
			req:     BulkUpsertAllegroMappingsRequest{AllegroCategoryID: "   ", Mappings: []UpsertAllegroParameterMappingRequest{validMapping}},
			wantErr: "allegro_category_id is required",
		},
		{
			name:    "empty mappings",
			req:     BulkUpsertAllegroMappingsRequest{AllegroCategoryID: "cat-123"},
			wantErr: "at least one mapping is required",
		},
		{
			name: "too many mappings",
			req: func() BulkUpsertAllegroMappingsRequest {
				mappings := make([]UpsertAllegroParameterMappingRequest, 201)
				for i := range mappings {
					mappings[i] = validMapping
				}
				return BulkUpsertAllegroMappingsRequest{AllegroCategoryID: "cat-123", Mappings: mappings}
			}(),
			wantErr: "max 200 mappings per request",
		},
		{
			name: "200 mappings (at limit)",
			req: func() BulkUpsertAllegroMappingsRequest {
				mappings := make([]UpsertAllegroParameterMappingRequest, 200)
				for i := range mappings {
					mappings[i] = validMapping
				}
				return BulkUpsertAllegroMappingsRequest{AllegroCategoryID: "cat-123", Mappings: mappings}
			}(),
		},
		{
			name: "mapping missing allegro_param_id",
			req: BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings: []UpsertAllegroParameterMappingRequest{
					{SourceType: "attribute", SourceKey: "color"},
				},
			},
			wantErr: "allegro_param_id is required for each mapping",
		},
		{
			name: "mapping missing source_type",
			req: BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings: []UpsertAllegroParameterMappingRequest{
					{AllegroParamID: "12345", SourceKey: "color"},
				},
			},
			wantErr: "source_type is required",
		},
		{
			name: "mapping invalid source_type",
			req: BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings: []UpsertAllegroParameterMappingRequest{
					{AllegroParamID: "12345", SourceType: "dynamic", SourceKey: "color"},
				},
			},
			wantErr: "source_type must be one of",
		},
		{
			name: "mapping missing source_key",
			req: BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings: []UpsertAllegroParameterMappingRequest{
					{AllegroParamID: "12345", SourceType: "attribute"},
				},
			},
			wantErr: "source_key is required",
		},
		{
			name: "mapping whitespace-only source_key",
			req: BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings: []UpsertAllegroParameterMappingRequest{
					{AllegroParamID: "12345", SourceType: "field", SourceKey: "   "},
				},
			},
			wantErr: "source_key is required",
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

func TestBulkUpsertAllegroMappingsRequest_Validate_AllSourceTypes(t *testing.T) {
	for _, st := range []string{"attribute", "field", "static"} {
		t.Run(st, func(t *testing.T) {
			req := BulkUpsertAllegroMappingsRequest{
				AllegroCategoryID: "cat-123",
				Mappings: []UpsertAllegroParameterMappingRequest{
					{AllegroParamID: "12345", SourceType: st, SourceKey: "k"},
				},
			}
			require.NoError(t, req.Validate())
		})
	}
}

// ─── CreateDropshipOrderRequest.Validate ────────────────────────────────────

func TestCreateDropshipOrderRequest_Validate(t *testing.T) {
	validItem := CreateDropshipOrderItemReq{
		ProductName: "Widget",
		Quantity:    1,
		UnitCost:    10.0,
	}

	tests := []struct {
		name    string
		req     CreateDropshipOrderRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req: CreateDropshipOrderRequest{
				OrderID:    uuid.New(),
				SupplierID: uuid.New(),
				Items:      []CreateDropshipOrderItemReq{validItem},
			},
		},
		{
			name: "valid with currency",
			req: CreateDropshipOrderRequest{
				OrderID:    uuid.New(),
				SupplierID: uuid.New(),
				Currency:   "EUR",
				Items:      []CreateDropshipOrderItemReq{validItem},
			},
		},
		{
			name: "missing order_id",
			req: CreateDropshipOrderRequest{
				SupplierID: uuid.New(),
				Items:      []CreateDropshipOrderItemReq{validItem},
			},
			wantErr: "order_id is required",
		},
		{
			name: "missing supplier_id",
			req: CreateDropshipOrderRequest{
				OrderID: uuid.New(),
				Items:   []CreateDropshipOrderItemReq{validItem},
			},
			wantErr: "supplier_id is required",
		},
		{
			name: "empty items",
			req: CreateDropshipOrderRequest{
				OrderID:    uuid.New(),
				SupplierID: uuid.New(),
			},
			wantErr: "at least one item is required",
		},
		{
			name: "item with invalid product_name",
			req: CreateDropshipOrderRequest{
				OrderID:    uuid.New(),
				SupplierID: uuid.New(),
				Items:      []CreateDropshipOrderItemReq{{ProductName: "", Quantity: 1}},
			},
			wantErr: "product_name is required",
		},
		{
			name: "item with zero quantity",
			req: CreateDropshipOrderRequest{
				OrderID:    uuid.New(),
				SupplierID: uuid.New(),
				Items:      []CreateDropshipOrderItemReq{{ProductName: "Widget", Quantity: 0}},
			},
			wantErr: "quantity must be greater than 0",
		},
		{
			name: "item with negative unit_cost",
			req: CreateDropshipOrderRequest{
				OrderID:    uuid.New(),
				SupplierID: uuid.New(),
				Items:      []CreateDropshipOrderItemReq{{ProductName: "Widget", Quantity: 1, UnitCost: -1}},
			},
			wantErr: "unit_cost must not be negative",
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

func TestCreateDropshipOrderRequest_Validate_DefaultCurrency(t *testing.T) {
	req := CreateDropshipOrderRequest{
		OrderID:    uuid.New(),
		SupplierID: uuid.New(),
		Items:      []CreateDropshipOrderItemReq{{ProductName: "Widget", Quantity: 1}},
	}
	require.NoError(t, req.Validate())
	assert.Equal(t, "PLN", req.Currency, "currency should default to 'PLN'")
}

// ─── CreateDropshipOrderItemReq.Validate ────────────────────────────────────

func TestCreateDropshipOrderItemReq_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateDropshipOrderItemReq
		wantErr string
	}{
		{
			name: "valid",
			req:  CreateDropshipOrderItemReq{ProductName: "Widget", Quantity: 5, UnitCost: 12.50},
		},
		{
			name: "valid with product_id",
			req: func() CreateDropshipOrderItemReq {
				pid := uuid.New()
				return CreateDropshipOrderItemReq{ProductID: &pid, ProductName: "Widget", Quantity: 1, UnitCost: 0}
			}(),
		},
		{
			name: "valid zero unit_cost",
			req:  CreateDropshipOrderItemReq{ProductName: "Free sample", Quantity: 1, UnitCost: 0},
		},
		{
			name:    "missing product_name",
			req:     CreateDropshipOrderItemReq{Quantity: 1},
			wantErr: "product_name is required",
		},
		{
			name:    "whitespace-only product_name",
			req:     CreateDropshipOrderItemReq{ProductName: "   ", Quantity: 1},
			wantErr: "product_name is required",
		},
		{
			name:    "zero quantity",
			req:     CreateDropshipOrderItemReq{ProductName: "Widget", Quantity: 0},
			wantErr: "quantity must be greater than 0",
		},
		{
			name:    "negative quantity",
			req:     CreateDropshipOrderItemReq{ProductName: "Widget", Quantity: -1},
			wantErr: "quantity must be greater than 0",
		},
		{
			name:    "negative unit_cost",
			req:     CreateDropshipOrderItemReq{ProductName: "Widget", Quantity: 1, UnitCost: -0.01},
			wantErr: "unit_cost must not be negative",
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

// ─── UpdateDropshipStatusRequest.Validate ───────────────────────────────────

func TestUpdateDropshipStatusRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateDropshipStatusRequest
		wantErr string
	}{
		{
			name: "valid pending",
			req:  UpdateDropshipStatusRequest{Status: "pending"},
		},
		{
			name: "valid sent",
			req:  UpdateDropshipStatusRequest{Status: "sent"},
		},
		{
			name: "valid confirmed",
			req:  UpdateDropshipStatusRequest{Status: "confirmed"},
		},
		{
			name: "valid shipped",
			req:  UpdateDropshipStatusRequest{Status: "shipped"},
		},
		{
			name: "valid delivered",
			req:  UpdateDropshipStatusRequest{Status: "delivered"},
		},
		{
			name: "valid cancelled",
			req:  UpdateDropshipStatusRequest{Status: "cancelled"},
		},
		{
			name: "valid with optional tracking",
			req: func() UpdateDropshipStatusRequest {
				tn := "TN-123"
				carrier := "DHL"
				return UpdateDropshipStatusRequest{Status: "shipped", TrackingNumber: &tn, Carrier: &carrier}
			}(),
		},
		{
			name:    "missing status",
			req:     UpdateDropshipStatusRequest{},
			wantErr: "status is required",
		},
		{
			name:    "whitespace-only status",
			req:     UpdateDropshipStatusRequest{Status: "   "},
			wantErr: "status is required",
		},
		{
			name:    "invalid status",
			req:     UpdateDropshipStatusRequest{Status: "processing"},
			wantErr: "status must be one of",
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

// ─── CreateListingSyncConfigRequest.Validate ────────────────────────────────

func TestCreateListingSyncConfigRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateListingSyncConfigRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateListingSyncConfigRequest{IntegrationID: uuid.New()},
		},
		{
			name: "valid with all fields",
			req: func() CreateListingSyncConfigRequest {
				autoSync := true
				interval := 30
				priceModifier := 1.15
				stockBuffer := 5
				return CreateListingSyncConfigRequest{
					IntegrationID:       uuid.New(),
					SyncDirection:       "push",
					AutoSync:            &autoSync,
					SyncIntervalMinutes: &interval,
					PriceRule:           "markup_pct",
					PriceModifier:       &priceModifier,
					StockBuffer:         &stockBuffer,
				}
			}(),
		},
		{
			name:    "missing integration_id",
			req:     CreateListingSyncConfigRequest{},
			wantErr: "integration_id is required",
		},
		{
			name:    "nil integration_id",
			req:     CreateListingSyncConfigRequest{IntegrationID: uuid.Nil},
			wantErr: "integration_id is required",
		},
		{
			name:    "invalid sync_direction",
			req:     CreateListingSyncConfigRequest{IntegrationID: uuid.New(), SyncDirection: "both"},
			wantErr: "sync_direction must be one of",
		},
		{
			name:    "invalid price_rule",
			req:     CreateListingSyncConfigRequest{IntegrationID: uuid.New(), PriceRule: "discount"},
			wantErr: "price_rule must be one of",
		},
		{
			name: "sync_interval_minutes too low",
			req: func() CreateListingSyncConfigRequest {
				i := 4
				return CreateListingSyncConfigRequest{IntegrationID: uuid.New(), SyncIntervalMinutes: &i}
			}(),
			wantErr: "sync_interval_minutes must be at least 5",
		},
		{
			name: "sync_interval_minutes at lower bound",
			req: func() CreateListingSyncConfigRequest {
				i := 5
				return CreateListingSyncConfigRequest{IntegrationID: uuid.New(), SyncIntervalMinutes: &i}
			}(),
		},
		{
			name: "negative stock_buffer",
			req: func() CreateListingSyncConfigRequest {
				sb := -1
				return CreateListingSyncConfigRequest{IntegrationID: uuid.New(), StockBuffer: &sb}
			}(),
			wantErr: "stock_buffer must be non-negative",
		},
		{
			name: "zero stock_buffer",
			req: func() CreateListingSyncConfigRequest {
				sb := 0
				return CreateListingSyncConfigRequest{IntegrationID: uuid.New(), StockBuffer: &sb}
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

func TestCreateListingSyncConfigRequest_Validate_DefaultSyncDirection(t *testing.T) {
	req := CreateListingSyncConfigRequest{IntegrationID: uuid.New()}
	require.NoError(t, req.Validate())
	assert.Equal(t, "bidirectional", req.SyncDirection, "sync_direction should default to 'bidirectional'")
}

func TestCreateListingSyncConfigRequest_Validate_DefaultPriceRule(t *testing.T) {
	req := CreateListingSyncConfigRequest{IntegrationID: uuid.New()}
	require.NoError(t, req.Validate())
	assert.Equal(t, "same", req.PriceRule, "price_rule should default to 'same'")
}

func TestCreateListingSyncConfigRequest_Validate_AllSyncDirections(t *testing.T) {
	for _, dir := range []string{"push", "pull", "bidirectional"} {
		t.Run(dir, func(t *testing.T) {
			req := CreateListingSyncConfigRequest{IntegrationID: uuid.New(), SyncDirection: dir}
			require.NoError(t, req.Validate())
		})
	}
}

func TestCreateListingSyncConfigRequest_Validate_AllPriceRules(t *testing.T) {
	for _, rule := range []string{"same", "markup_pct", "markup_fixed", "custom"} {
		t.Run(rule, func(t *testing.T) {
			req := CreateListingSyncConfigRequest{IntegrationID: uuid.New(), PriceRule: rule}
			require.NoError(t, req.Validate())
		})
	}
}

// ─── UpdateListingSyncConfigRequest.Validate ────────────────────────────────

func TestUpdateListingSyncConfigRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateListingSyncConfigRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateListingSyncConfigRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid sync_direction update",
			req: func() UpdateListingSyncConfigRequest {
				d := "push"
				return UpdateListingSyncConfigRequest{SyncDirection: &d}
			}(),
		},
		{
			name: "invalid sync_direction",
			req: func() UpdateListingSyncConfigRequest {
				d := "both"
				return UpdateListingSyncConfigRequest{SyncDirection: &d}
			}(),
			wantErr: "sync_direction must be one of",
		},
		{
			name: "valid price_rule update",
			req: func() UpdateListingSyncConfigRequest {
				r := "markup_pct"
				return UpdateListingSyncConfigRequest{PriceRule: &r}
			}(),
		},
		{
			name: "invalid price_rule",
			req: func() UpdateListingSyncConfigRequest {
				r := "discount"
				return UpdateListingSyncConfigRequest{PriceRule: &r}
			}(),
			wantErr: "price_rule must be one of",
		},
		{
			name: "valid status active",
			req: func() UpdateListingSyncConfigRequest {
				s := "active"
				return UpdateListingSyncConfigRequest{Status: &s}
			}(),
		},
		{
			name: "valid status paused",
			req: func() UpdateListingSyncConfigRequest {
				s := "paused"
				return UpdateListingSyncConfigRequest{Status: &s}
			}(),
		},
		{
			name: "valid status error",
			req: func() UpdateListingSyncConfigRequest {
				s := "error"
				return UpdateListingSyncConfigRequest{Status: &s}
			}(),
		},
		{
			name: "invalid status",
			req: func() UpdateListingSyncConfigRequest {
				s := "disabled"
				return UpdateListingSyncConfigRequest{Status: &s}
			}(),
			wantErr: "status must be one of",
		},
		{
			name: "sync_interval_minutes too low",
			req: func() UpdateListingSyncConfigRequest {
				i := 4
				return UpdateListingSyncConfigRequest{SyncIntervalMinutes: &i}
			}(),
			wantErr: "sync_interval_minutes must be at least 5",
		},
		{
			name: "negative stock_buffer",
			req: func() UpdateListingSyncConfigRequest {
				sb := -1
				return UpdateListingSyncConfigRequest{StockBuffer: &sb}
			}(),
			wantErr: "stock_buffer must be non-negative",
		},
		{
			name: "valid auto_sync update",
			req: func() UpdateListingSyncConfigRequest {
				b := false
				return UpdateListingSyncConfigRequest{AutoSync: &b}
			}(),
		},
		{
			name: "valid price_modifier update",
			req: func() UpdateListingSyncConfigRequest {
				pm := 1.25
				return UpdateListingSyncConfigRequest{PriceModifier: &pm}
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

// ─── CreateWarehouseDocumentRequest.Validate ────────────────────────────────

func TestCreateWarehouseDocumentRequest_Validate(t *testing.T) {
	validItem := CreateWarehouseDocItemRequest{
		ProductID: uuid.New(),
		Quantity:  10,
	}
	warehouseID := uuid.New()
	targetWarehouseID := uuid.New()

	tests := []struct {
		name    string
		req     CreateWarehouseDocumentRequest
		wantErr string
	}{
		{
			name: "valid PZ",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "PZ",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{validItem},
			},
		},
		{
			name: "valid WZ",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "WZ",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{validItem},
			},
		},
		{
			name: "valid MM",
			req: CreateWarehouseDocumentRequest{
				DocumentType:      "MM",
				WarehouseID:       warehouseID,
				TargetWarehouseID: &targetWarehouseID,
				Items:             []CreateWarehouseDocItemRequest{validItem},
			},
		},
		{
			name: "valid lowercase document_type normalized to upper",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "pz",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{validItem},
			},
		},
		{
			name: "invalid document_type",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "RW",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{validItem},
			},
			wantErr: "document_type must be PZ, WZ, or MM",
		},
		{
			name: "empty document_type",
			req: CreateWarehouseDocumentRequest{
				WarehouseID: warehouseID,
				Items:       []CreateWarehouseDocItemRequest{validItem},
			},
			wantErr: "document_type must be PZ, WZ, or MM",
		},
		{
			name: "missing warehouse_id",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "PZ",
				Items:        []CreateWarehouseDocItemRequest{validItem},
			},
			wantErr: "warehouse_id is required",
		},
		{
			name: "MM without target_warehouse_id",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "MM",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{validItem},
			},
			wantErr: "target_warehouse_id is required for MM documents",
		},
		{
			name: "MM with nil target_warehouse_id",
			req: func() CreateWarehouseDocumentRequest {
				nilID := uuid.Nil
				return CreateWarehouseDocumentRequest{
					DocumentType:      "MM",
					WarehouseID:       warehouseID,
					TargetWarehouseID: &nilID,
					Items:             []CreateWarehouseDocItemRequest{validItem},
				}
			}(),
			wantErr: "target_warehouse_id is required for MM documents",
		},
		{
			name: "MM with same source and target warehouse",
			req: CreateWarehouseDocumentRequest{
				DocumentType:      "MM",
				WarehouseID:       warehouseID,
				TargetWarehouseID: &warehouseID,
				Items:             []CreateWarehouseDocItemRequest{validItem},
			},
			wantErr: "target_warehouse_id must differ from warehouse_id",
		},
		{
			name: "empty items",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "PZ",
				WarehouseID:  warehouseID,
			},
			wantErr: "at least one item is required",
		},
		{
			name: "item missing product_id",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "PZ",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{{Quantity: 5}},
			},
			wantErr: "product_id is required for each item",
		},
		{
			name: "item with zero quantity",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "PZ",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{{ProductID: uuid.New(), Quantity: 0}},
			},
			wantErr: "quantity must be positive for each item",
		},
		{
			name: "item with negative quantity",
			req: CreateWarehouseDocumentRequest{
				DocumentType: "PZ",
				WarehouseID:  warehouseID,
				Items:        []CreateWarehouseDocItemRequest{{ProductID: uuid.New(), Quantity: -1}},
			},
			wantErr: "quantity must be positive for each item",
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

// ─── UpdateWarehouseDocumentRequest.Validate ────────────────────────────────

func TestUpdateWarehouseDocumentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateWarehouseDocumentRequest
		wantErr string
	}{
		{
			name: "valid notes update",
			req: func() UpdateWarehouseDocumentRequest {
				notes := "Updated notes"
				return UpdateWarehouseDocumentRequest{Notes: &notes}
			}(),
		},
		{
			name:    "empty update",
			req:     UpdateWarehouseDocumentRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name:    "nil notes",
			req:     UpdateWarehouseDocumentRequest{Notes: nil},
			wantErr: "at least one field must be provided",
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

// ─── SupplierShipRequest.Validate ───────────────────────────────────────────

func TestSupplierShipRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     SupplierShipRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  SupplierShipRequest{TrackingNumber: "TN-12345", Carrier: "DHL"},
		},
		{
			name:    "missing tracking_number",
			req:     SupplierShipRequest{Carrier: "DHL"},
			wantErr: "tracking_number is required",
		},
		{
			name:    "whitespace-only tracking_number",
			req:     SupplierShipRequest{TrackingNumber: "   ", Carrier: "DHL"},
			wantErr: "tracking_number is required",
		},
		{
			name:    "missing carrier",
			req:     SupplierShipRequest{TrackingNumber: "TN-12345"},
			wantErr: "carrier is required",
		},
		{
			name:    "whitespace-only carrier",
			req:     SupplierShipRequest{TrackingNumber: "TN-12345", Carrier: "   "},
			wantErr: "carrier is required",
		},
		{
			name:    "tracking_number too long",
			req:     SupplierShipRequest{TrackingNumber: strings.Repeat("x", 201), Carrier: "DHL"},
			wantErr: "tracking_number",
		},
		{
			name: "tracking_number at max length",
			req:  SupplierShipRequest{TrackingNumber: strings.Repeat("x", 200), Carrier: "DHL"},
		},
		{
			name:    "carrier too long",
			req:     SupplierShipRequest{TrackingNumber: "TN-12345", Carrier: strings.Repeat("x", 201)},
			wantErr: "carrier",
		},
		{
			name: "carrier at max length",
			req:  SupplierShipRequest{TrackingNumber: "TN-12345", Carrier: strings.Repeat("x", 200)},
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

// ─── CreateSupplierMessageRequest.Validate ──────────────────────────────────

func TestCreateSupplierMessageRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateSupplierMessageRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  CreateSupplierMessageRequest{Message: "Hello, order received."},
		},
		{
			name:    "missing message",
			req:     CreateSupplierMessageRequest{},
			wantErr: "message is required",
		},
		{
			name:    "whitespace-only message",
			req:     CreateSupplierMessageRequest{Message: "   "},
			wantErr: "message is required",
		},
		{
			name:    "message too long",
			req:     CreateSupplierMessageRequest{Message: strings.Repeat("x", 5001)},
			wantErr: "message",
		},
		{
			name: "message at max length",
			req:  CreateSupplierMessageRequest{Message: strings.Repeat("x", 5000)},
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

// ─── UpdateProductListingRequest.Validate ───────────────────────────────────

func TestUpdateProductListingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateProductListingRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateProductListingRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid external_id update",
			req: func() UpdateProductListingRequest {
				extID := "NEW-EXT-123"
				return UpdateProductListingRequest{ExternalID: &extID}
			}(),
		},
		{
			name: "valid status update",
			req: func() UpdateProductListingRequest {
				s := "active"
				return UpdateProductListingRequest{Status: &s}
			}(),
		},
		{
			name: "valid url update",
			req: func() UpdateProductListingRequest {
				u := "https://allegro.pl/offer/456"
				return UpdateProductListingRequest{URL: &u}
			}(),
		},
		{
			name: "valid price_override update",
			req: func() UpdateProductListingRequest {
				p := 149.99
				return UpdateProductListingRequest{PriceOverride: &p}
			}(),
		},
		{
			name: "valid stock_override update",
			req: func() UpdateProductListingRequest {
				s := 50
				return UpdateProductListingRequest{StockOverride: &s}
			}(),
		},
		{
			name: "valid stock_sync_mode auto",
			req: func() UpdateProductListingRequest {
				m := "auto"
				return UpdateProductListingRequest{StockSyncMode: &m}
			}(),
		},
		{
			name: "valid stock_sync_mode manual",
			req: func() UpdateProductListingRequest {
				m := "manual"
				return UpdateProductListingRequest{StockSyncMode: &m}
			}(),
		},
		{
			name: "invalid stock_sync_mode",
			req: func() UpdateProductListingRequest {
				m := "semi-auto"
				return UpdateProductListingRequest{StockSyncMode: &m}
			}(),
			wantErr: "stock_sync_mode must be 'auto' or 'manual'",
		},
		{
			name: "valid sync_status update",
			req: func() UpdateProductListingRequest {
				s := "synced"
				return UpdateProductListingRequest{SyncStatus: &s}
			}(),
		},
		{
			name: "valid error_message update",
			req: func() UpdateProductListingRequest {
				e := "API timeout"
				return UpdateProductListingRequest{ErrorMessage: &e}
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

// ─── CreateCategoryRequest.Validate ─────────────────────────────────────────

func TestCreateCategoryRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateCategoryRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateCategoryRequest{Name: "Electronics"},
		},
		{
			name: "valid with all fields",
			req: func() CreateCategoryRequest {
				parentID := uuid.New()
				icon := "laptop"
				return CreateCategoryRequest{
					Name:     "Laptops",
					ParentID: &parentID,
					Color:    "#ff5733",
					Icon:     &icon,
				}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateCategoryRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateCategoryRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateCategoryRequest{Name: strings.Repeat("x", 201)},
			wantErr: "name",
		},
		{
			name: "name at max length",
			req:  CreateCategoryRequest{Name: strings.Repeat("x", 200)},
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

func TestCreateCategoryRequest_Validate_DefaultColor(t *testing.T) {
	req := CreateCategoryRequest{Name: "Electronics"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "#6b7280", req.Color, "color should default to '#6b7280'")
}

func TestCreateCategoryRequest_Validate_PreservesExplicitColor(t *testing.T) {
	req := CreateCategoryRequest{Name: "Electronics", Color: "#ff0000"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "#ff0000", req.Color)
}

// ─── UpdateCategoryRequest.Validate ─────────────────────────────────────────

func TestUpdateCategoryRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateCategoryRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateCategoryRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req: func() UpdateCategoryRequest {
				name := "Updated Category"
				return UpdateCategoryRequest{Name: &name}
			}(),
		},
		{
			name: "empty name",
			req: func() UpdateCategoryRequest {
				name := ""
				return UpdateCategoryRequest{Name: &name}
			}(),
			wantErr: "name must not be empty",
		},
		{
			name: "whitespace-only name",
			req: func() UpdateCategoryRequest {
				name := "   "
				return UpdateCategoryRequest{Name: &name}
			}(),
			wantErr: "name must not be empty",
		},
		{
			name: "name too long",
			req: func() UpdateCategoryRequest {
				name := strings.Repeat("x", 201)
				return UpdateCategoryRequest{Name: &name}
			}(),
			wantErr: "name",
		},
		{
			name: "name at max length",
			req: func() UpdateCategoryRequest {
				name := strings.Repeat("x", 200)
				return UpdateCategoryRequest{Name: &name}
			}(),
		},
		{
			name: "valid parent_id update",
			req: func() UpdateCategoryRequest {
				parentID := uuid.New()
				return UpdateCategoryRequest{ParentID: &parentID}
			}(),
		},
		{
			name: "valid color update",
			req: func() UpdateCategoryRequest {
				color := "#abcdef"
				return UpdateCategoryRequest{Color: &color}
			}(),
		},
		{
			name: "valid icon update",
			req: func() UpdateCategoryRequest {
				icon := "phone"
				return UpdateCategoryRequest{Icon: &icon}
			}(),
		},
		{
			name: "valid position update",
			req: func() UpdateCategoryRequest {
				pos := 5
				return UpdateCategoryRequest{Position: &pos}
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
