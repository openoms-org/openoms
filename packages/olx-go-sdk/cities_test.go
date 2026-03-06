package olx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/cities" {
			t.Errorf("path = %q, want /cities", r.URL.Path)
		}
		if o := r.URL.Query().Get("offset"); o != "0" {
			t.Errorf("offset = %q, want %q", o, "0")
		}
		if l := r.URL.Query().Get("limit"); l != "50" {
			t.Errorf("limit = %q, want %q", l, "50")
		}

		resp := CityListResponse{
			Data: []City{
				{ID: 1, Name: "Warszawa", County: "Warszawa", Municipality: "Warszawa"},
				{ID: 2, Name: "Kraków", County: "krakowski", Municipality: "Kraków"},
			},
			Total: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Cities.ListCities(context.Background(), "", 0, 50)
	if err != nil {
		t.Fatalf("ListCities error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(result.Data))
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Data[0].Name != "Warszawa" {
		t.Errorf("Data[0].Name = %q, want %q", result.Data[0].Name, "Warszawa")
	}
	if result.Data[1].Name != "Kraków" {
		t.Errorf("Data[1].Name = %q, want %q", result.Data[1].Name, "Kraków")
	}
}

func TestListCitiesWithQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("query"); q != "Warsz" {
			t.Errorf("query = %q, want %q", q, "Warsz")
		}

		resp := CityListResponse{
			Data:  []City{{ID: 1, Name: "Warszawa", County: "Warszawa"}},
			Total: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Cities.ListCities(context.Background(), "Warsz", 0, 50)
	if err != nil {
		t.Fatalf("ListCities error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].ID != 1 {
		t.Errorf("Data[0].ID = %d, want 1", result.Data[0].ID)
	}
}

func TestGetCity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/cities/12345" {
			t.Errorf("path = %q, want /cities/12345", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"id":12345,"name":"Warszawa","county":"Warszawa","municipality":"Warszawa","latitude":52.2297,"longitude":21.0122}}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	city, err := c.Cities.GetCity(context.Background(), 12345)
	if err != nil {
		t.Fatalf("GetCity error: %v", err)
	}
	if city.ID != 12345 {
		t.Errorf("ID = %d, want 12345", city.ID)
	}
	if city.Name != "Warszawa" {
		t.Errorf("Name = %q, want %q", city.Name, "Warszawa")
	}
	if city.County != "Warszawa" {
		t.Errorf("County = %q, want %q", city.County, "Warszawa")
	}
	if city.Latitude != 52.2297 {
		t.Errorf("Latitude = %f, want 52.2297", city.Latitude)
	}
}

func TestListCities_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized","message":"Invalid token"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	_, err := c.Cities.ListCities(context.Background(), "", 0, 50)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
