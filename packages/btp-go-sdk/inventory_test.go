package btp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInventoryGetReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/InventoryReportGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvReportDocument{
			Header: InvReportHeader{
				InventoryReportNumber: "INV-001",
				Currency:              "PLN",
			},
			Lines: []InvReportLine{
				{
					LineNumber:      1,
					ItemID:          "ITEM001",
					EAN:             "5901234567890",
					ItemName:        "Test Product",
					UnitNetPrice:    100.50,
					UnitRetailPrice: 123.62,
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	doc, err := c.Inventory.GetReport(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Header.InventoryReportNumber != "INV-001" {
		t.Fatalf("expected INV-001, got %s", doc.Header.InventoryReportNumber)
	}
	if doc.Header.Currency != "PLN" {
		t.Fatalf("expected PLN, got %s", doc.Header.Currency)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(doc.Lines))
	}
	if doc.Lines[0].EAN != "5901234567890" {
		t.Fatalf("expected EAN 5901234567890, got %s", doc.Lines[0].EAN)
	}
	if doc.Lines[0].UnitNetPrice != 100.50 {
		t.Fatalf("expected price 100.50, got %f", doc.Lines[0].UnitNetPrice)
	}
}

func TestInventoryGetReportWithOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("useItemCategoryFilter") != "false" {
			t.Fatalf("expected useItemCategoryFilter=false, got %s", q.Get("useItemCategoryFilter"))
		}
		if q.Get("langId") != "en" {
			t.Fatalf("expected langId=en, got %s", q.Get("langId"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvReportDocument{})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	filterFalse := false
	_, err := c.Inventory.GetReport(context.Background(), &InventoryReportOptions{
		UseItemCategoryFilter: &filterFalse,
		LangID:                "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInventoryGetReportWithStockSchedules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvReportDocument{
			Lines: []InvReportLine{
				{
					LineNumber: 1,
					ItemID:     "ITEM002",
					Availability: &Availability{
						StockSchedules: []StockSchedule{
							{ScheduleDate: "2026-03-01", Quantity: 50, WarehouseName: "WH-Main"},
							{ScheduleDate: "2026-03-15", Quantity: 100, WarehouseName: "WH-Main"},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	doc, err := c.Inventory.GetReport(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Lines[0].Availability == nil {
		t.Fatal("expected availability")
	}
	if len(doc.Lines[0].Availability.StockSchedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(doc.Lines[0].Availability.StockSchedules))
	}
	if doc.Lines[0].Availability.StockSchedules[0].Quantity != 50 {
		t.Fatalf("expected quantity 50, got %f", doc.Lines[0].Availability.StockSchedules[0].Quantity)
	}
}
