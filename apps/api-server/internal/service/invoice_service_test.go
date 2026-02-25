package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestInvoiceService_Create_ValidationError_MissingOrderID(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateInvoiceRequest{
		Provider: "fakturownia",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestInvoiceService_Create_ValidationError_MissingProvider(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateInvoiceRequest{
		OrderID: uuid.New(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}

func TestBuildInvoiceItems(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)
	orderID := uuid.New()

	tests := []struct {
		name        string
		order       *model.Order
		taxRate     int
		wantLen     int
		checkResult func(t *testing.T, items []integration.InvoiceItem)
	}{
		{
			name: "normal items with 23% tax",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 123.00,
				Items:       json.RawMessage(`[{"name":"Widget","sku":"W001","quantity":2,"price":49.99},{"name":"Gadget","sku":"G001","quantity":1,"price":23.02}]`),
			},
			taxRate: 23,
			wantLen: 2,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "Widget", items[0].Name)
				assert.Equal(t, 2, items[0].Quantity)
				assert.Equal(t, 23, items[0].TaxRate)
				assert.Equal(t, "szt.", items[0].Unit)
				expectedNet := 49.99 / 1.23
				assert.InDelta(t, expectedNet, items[0].NetPrice, 0.01)

				assert.Equal(t, "Gadget", items[1].Name)
				assert.Equal(t, 1, items[1].Quantity)
				expectedNet2 := 23.02 / 1.23
				assert.InDelta(t, expectedNet2, items[1].NetPrice, 0.01)
			},
		},
		{
			name: "nil items falls back to single line",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 100.00,
				Items:       nil,
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Contains(t, items[0].Name, "Zamówienie")
				assert.Equal(t, 1, items[0].Quantity)
				assert.Equal(t, 23, items[0].TaxRate)
				assert.Equal(t, "szt.", items[0].Unit)
				expectedNet := 100.00 / 1.23
				assert.InDelta(t, expectedNet, items[0].NetPrice, 0.01)
			},
		},
		{
			name: "empty array falls back to single line",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 50.00,
				Items:       json.RawMessage(`[]`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Contains(t, items[0].Name, "Zamówienie")
				assert.Equal(t, 1, items[0].Quantity)
				expectedNet := 50.00 / 1.23
				assert.InDelta(t, expectedNet, items[0].NetPrice, 0.01)
			},
		},
		{
			name: "null JSON falls back to single line",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 200.00,
				Items:       json.RawMessage(`null`),
			},
			taxRate: 8,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Contains(t, items[0].Name, "Zamówienie")
				expectedNet := 200.00 / 1.08
				assert.InDelta(t, expectedNet, items[0].NetPrice, 0.01)
				assert.Equal(t, 8, items[0].TaxRate)
			},
		},
		{
			name: "invalid JSON falls back to single line",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 75.00,
				Items:       json.RawMessage(`{not valid json`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Contains(t, items[0].Name, "Zamówienie")
				assert.Equal(t, 1, items[0].Quantity)
			},
		},
		{
			name: "zero tax rate",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 100.00,
				Items:       json.RawMessage(`[{"name":"Tax Free Item","sku":"TF01","quantity":1,"price":100.00}]`),
			},
			taxRate: 0,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "Tax Free Item", items[0].Name)
				assert.Equal(t, 0, items[0].TaxRate)
				// With 0% tax, net price equals gross price: 100 / (1 + 0/100) = 100
				assert.InDelta(t, 100.00, items[0].NetPrice, 0.01)
			},
		},
		{
			name: "8% tax rate",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 108.00,
				Items:       json.RawMessage(`[{"name":"Food Item","sku":"FD01","quantity":3,"price":36.00}]`),
			},
			taxRate: 8,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "Food Item", items[0].Name)
				assert.Equal(t, 3, items[0].Quantity)
				assert.Equal(t, 8, items[0].TaxRate)
				expectedNet := 36.00 / 1.08
				assert.InDelta(t, expectedNet, items[0].NetPrice, 0.01)
			},
		},
		{
			name: "item with zero quantity defaults to 1",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 50.00,
				Items:       json.RawMessage(`[{"name":"Zero Qty","sku":"ZQ01","quantity":0,"price":50.00}]`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, 1, items[0].Quantity)
			},
		},
		{
			name: "item with negative quantity defaults to 1",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 50.00,
				Items:       json.RawMessage(`[{"name":"Neg Qty","sku":"NQ01","quantity":-2,"price":50.00}]`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, 1, items[0].Quantity)
			},
		},
		{
			name: "items with missing name field",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 30.00,
				Items:       json.RawMessage(`[{"sku":"X01","quantity":1,"price":30.00}]`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "", items[0].Name)
				assert.Equal(t, 1, items[0].Quantity)
			},
		},
		{
			name: "items with missing price field (defaults to 0)",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 0,
				Items:       json.RawMessage(`[{"name":"Free Item","quantity":1}]`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "Free Item", items[0].Name)
				assert.InDelta(t, 0.0, items[0].NetPrice, 0.001)
			},
		},
		{
			name: "single item",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 99.99,
				Items:       json.RawMessage(`[{"name":"Single","sku":"S01","quantity":1,"price":99.99}]`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "Single", items[0].Name)
				expectedNet := 99.99 / 1.23
				assert.InDelta(t, expectedNet, items[0].NetPrice, 0.01)
			},
		},
		{
			name: "many items",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 300.00,
				Items:       json.RawMessage(`[{"name":"A","quantity":1,"price":100},{"name":"B","quantity":2,"price":50},{"name":"C","quantity":3,"price":33.33}]`),
			},
			taxRate: 23,
			wantLen: 3,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, "A", items[0].Name)
				assert.Equal(t, "B", items[1].Name)
				assert.Equal(t, "C", items[2].Name)
				assert.Equal(t, 1, items[0].Quantity)
				assert.Equal(t, 2, items[1].Quantity)
				assert.Equal(t, 3, items[2].Quantity)
				// All should have correct unit
				for _, item := range items {
					assert.Equal(t, "szt.", item.Unit)
					assert.Equal(t, 23, item.TaxRate)
				}
			},
		},
		{
			name: "fallback single line uses first 8 chars of order ID",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 123.45,
				Items:       nil,
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				expectedName := "Zamówienie " + orderID.String()[:8]
				assert.Equal(t, expectedName, items[0].Name)
			},
		},
		{
			name: "JSON array of wrong shape falls back to single line",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 100.00,
				Items:       json.RawMessage(`"just a string"`),
			},
			taxRate: 23,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Contains(t, items[0].Name, "Zamówienie")
			},
		},
		{
			name: "very high tax rate",
			order: &model.Order{
				ID:          orderID,
				TotalAmount: 200.00,
				Items:       json.RawMessage(`[{"name":"Luxury","quantity":1,"price":200.00}]`),
			},
			taxRate: 100,
			wantLen: 1,
			checkResult: func(t *testing.T, items []integration.InvoiceItem) {
				assert.Equal(t, 100, items[0].TaxRate)
				// 200 / (1 + 100/100) = 200 / 2 = 100
				assert.InDelta(t, 100.00, items[0].NetPrice, 0.01)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := svc.buildInvoiceItems(tt.order, tt.taxRate)
			require.Len(t, items, tt.wantLen)

			// Every item should have a non-negative net price and valid tax rate
			for _, item := range items {
				assert.False(t, math.IsNaN(item.NetPrice), "NetPrice should not be NaN")
				assert.False(t, math.IsInf(item.NetPrice, 0), "NetPrice should not be Inf")
				assert.GreaterOrEqual(t, item.Quantity, 1, "Quantity should be at least 1")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, items)
			}
		})
	}
}
