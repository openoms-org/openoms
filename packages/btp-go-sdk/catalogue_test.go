package btp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalogueGetCatalogue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/ProductCatalogueGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ProdCatalogueDocument{
			Header: ProdCatalogueHeader{
				CatalogueNumber: "CAT-001",
				Currency:        "PLN",
			},
			Lines: []ProdCatalogueLine{
				{
					LineNumber:           1,
					ItemID:               "PROD001",
					EAN:                  "5901234567890",
					ManufacturerItemCode: "MFR-001",
					ItemName:             "Test Product",
					ItemDescription:      "A test product description",
					UnitOfMeasure:        "pce",
					UnitNetPrice:         99.99,
					UnitRetailPrice:      122.99,
					CategoryNodes: []CategoryNode{
						{CategoryNodeID: "cat1", CategoryNodeName: "Electronics"},
					},
					BrandNodes: []BrandNode{
						{BrandNodeID: "brand1", BrandNodeName: "TestBrand"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	doc, err := c.Catalogue.GetCatalogue(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Header.CatalogueNumber != "CAT-001" {
		t.Fatalf("expected CAT-001, got %s", doc.Header.CatalogueNumber)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(doc.Lines))
	}
	line := doc.Lines[0]
	if line.EAN != "5901234567890" {
		t.Fatalf("expected EAN 5901234567890, got %s", line.EAN)
	}
	if line.ItemName != "Test Product" {
		t.Fatalf("expected 'Test Product', got %s", line.ItemName)
	}
	if len(line.CategoryNodes) != 1 || line.CategoryNodes[0].CategoryNodeName != "Electronics" {
		t.Fatal("expected category Electronics")
	}
	if len(line.BrandNodes) != 1 || line.BrandNodes[0].BrandNodeName != "TestBrand" {
		t.Fatal("expected brand TestBrand")
	}
}

func TestCatalogueGetCatalogueWithOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("langId") != "pl" {
			t.Fatalf("expected langId=pl, got %s", q.Get("langId"))
		}
		if q.Get("useItemCategoryFilter") != "true" {
			t.Fatalf("expected useItemCategoryFilter=true, got %s", q.Get("useItemCategoryFilter"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ProdCatalogueDocument{})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	filterTrue := true
	_, err := c.Catalogue.GetCatalogue(context.Background(), &CatalogueOptions{
		UseItemCategoryFilter: &filterTrue,
		LangID:                "pl",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
