package olx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("client_id", "client_secret", "access_tok")

	if c.clientID != "client_id" {
		t.Errorf("clientID = %q, want %q", c.clientID, "client_id")
	}
	if c.clientSecret != "client_secret" {
		t.Errorf("clientSecret = %q, want %q", c.clientSecret, "client_secret")
	}
	if c.accessToken != "access_tok" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "access_tok")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Adverts == nil {
		t.Error("Adverts service is nil")
	}
	if c.Transactions == nil {
		t.Error("Transactions service is nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("id", "secret", "tok", WithBaseURL("https://custom.api/"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithAccessToken(t *testing.T) {
	c := NewClient("id", "secret", "", WithAccessToken("override_tok"))

	if c.accessToken != "override_tok" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "override_tok")
	}
}

func TestDoSetsAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test_token")
		}
		version := r.Header.Get("Version")
		if version != "2.0" {
			t.Errorf("Version = %q, want %q", version, "2.0")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoHandlesErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"access_denied","message":"Insufficient permissions"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/forbidden", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
	if apiErr.ErrorType != "access_denied" {
		t.Errorf("ErrorType = %q, want %q", apiErr.ErrorType, "access_denied")
	}
}

func TestListAdverts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adverts" {
			t.Errorf("path = %q, want /adverts", r.URL.Path)
		}
		if s := r.URL.Query().Get("status"); s != "active" {
			t.Errorf("status = %q, want active", s)
		}

		resp := AdvertListResponse{
			Data: []Advert{
				{
					ID:    12345,
					Title: "Test Product",
					Status: "active",
					Price: &AdvertPrice{Value: 99.99, Currency: "PLN"},
					Contact: &Contact{Name: "Jan Kowalski", Email: "jan@test.pl"},
				},
			},
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

	result, err := c.Adverts.ListAdverts(context.Background(), AdvertListParams{Status: "active"})
	if err != nil {
		t.Fatalf("ListAdverts error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].Title != "Test Product" {
		t.Errorf("Title = %q, want %q", result.Data[0].Title, "Test Product")
	}
	if result.Data[0].Price.Value != 99.99 {
		t.Errorf("Price.Value = %f, want 99.99", result.Data[0].Price.Value)
	}
}

func TestGetAdvert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adverts/12345" {
			t.Errorf("path = %q, want /adverts/12345", r.URL.Path)
		}
		resp := struct {
			Data Advert `json:"data"`
		}{
			Data: Advert{
				ID:    12345,
				Title: "Test Product",
				Status: "active",
			},
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

	advert, err := c.Adverts.GetAdvert(context.Background(), 12345)
	if err != nil {
		t.Fatalf("GetAdvert error: %v", err)
	}
	if advert.ID != 12345 {
		t.Errorf("ID = %d, want 12345", advert.ID)
	}
	if advert.Title != "Test Product" {
		t.Errorf("Title = %q, want %q", advert.Title, "Test Product")
	}
}

func TestListTransactions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transactions" {
			t.Errorf("path = %q, want /transactions", r.URL.Path)
		}
		if ca := r.URL.Query().Get("created_after"); ca != "2024-01-15T00:00:00Z" {
			t.Errorf("created_after = %q, want 2024-01-15T00:00:00Z", ca)
		}

		resp := TransactionListResponse{
			Data: []Transaction{
				{
					ID:          "tx-001",
					AdvertID:    12345,
					Status:      "completed",
					Amount:      199.99,
					Currency:    "PLN",
					CreatedAt:   "2024-01-15T10:30:00Z",
					BuyerName:   "Anna Nowak",
					BuyerEmail:  "anna@test.pl",
					AdvertTitle: "Test Product",
					Quantity:    1,
					ShippingAddr: &ShippingAddr{
						Name:       "Anna Nowak",
						Street:     "Krakowska 5",
						City:       "Krakow",
						PostalCode: "30-001",
						Country:    "PL",
					},
				},
			},
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

	result, err := c.Transactions.ListTransactions(context.Background(), TransactionListParams{
		CreatedAfter: "2024-01-15T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("ListTransactions error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].ID != "tx-001" {
		t.Errorf("ID = %q, want %q", result.Data[0].ID, "tx-001")
	}
	if result.Data[0].BuyerName != "Anna Nowak" {
		t.Errorf("BuyerName = %q, want %q", result.Data[0].BuyerName, "Anna Nowak")
	}
	if result.Data[0].ShippingAddr.City != "Krakow" {
		t.Errorf("ShippingAddr.City = %q, want %q", result.Data[0].ShippingAddr.City, "Krakow")
	}
}

func TestAPIErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with error type and message",
			err:  APIError{StatusCode: 403, ErrorType: "access_denied", Message: "Forbidden"},
			want: "olx: HTTP 403 [access_denied]: Forbidden",
		},
		{
			name: "message only",
			err:  APIError{StatusCode: 500, Message: "Internal error"},
			want: "olx: HTTP 500: Internal error",
		},
		{
			name: "status only",
			err:  APIError{StatusCode: 400},
			want: "olx: HTTP 400",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAPIErrorUnwrap(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{401, ErrUnauthorized},
		{403, ErrForbidden},
		{404, ErrNotFound},
		{429, ErrRateLimited},
		{500, ErrServerError},
		{503, ErrServerError},
		{400, nil},
	}

	for _, tc := range tests {
		apiErr := &APIError{StatusCode: tc.status}
		got := apiErr.Unwrap()
		if got != tc.want {
			t.Errorf("Unwrap() for status %d = %v, want %v", tc.status, got, tc.want)
		}
	}
}
