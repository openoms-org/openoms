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
		_, _ = w.Write([]byte(`{}`))
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"access_denied","message":"Insufficient permissions"}`))
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
					ID:      12345,
					Title:   "Test Product",
					Status:  "active",
					Price:   &AdvertPrice{Value: 99.99, Currency: "PLN"},
					Contact: &Contact{Name: "Jan Kowalski", Email: "jan@test.pl"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
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
				ID:     12345,
				Title:  "Test Product",
				Status: "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
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
		_ = json.NewEncoder(w).Encode(resp)
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

func TestCreateAdvert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/adverts" {
			t.Errorf("path = %q, want /adverts", r.URL.Path)
		}

		var body struct {
			Title       string `json:"title"`
			CategoryID  int    `json:"category_id"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Title != "Test" {
			t.Errorf("body.Title = %q, want %q", body.Title, "Test")
		}
		if body.CategoryID != 42 {
			t.Errorf("body.CategoryID = %d, want 42", body.CategoryID)
		}
		if body.Description != "A test advert" {
			t.Errorf("body.Description = %q, want %q", body.Description, "A test advert")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":99999,"title":"Test","status":"new"}}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	advert, err := c.Adverts.CreateAdvert(context.Background(), CreateAdvertRequest{
		Title:       "Test",
		CategoryID:  42,
		Description: "A test advert",
	})
	if err != nil {
		t.Fatalf("CreateAdvert error: %v", err)
	}
	if advert.ID != 99999 {
		t.Errorf("ID = %d, want 99999", advert.ID)
	}
	if advert.Title != "Test" {
		t.Errorf("Title = %q, want %q", advert.Title, "Test")
	}
	if advert.Status != "new" {
		t.Errorf("Status = %q, want %q", advert.Status, "new")
	}
}

func TestUpdateAdvert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/adverts/12345" {
			t.Errorf("path = %q, want /adverts/12345", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":12345,"title":"Updated","status":"active"}}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	advert, err := c.Adverts.UpdateAdvert(context.Background(), 12345, CreateAdvertRequest{
		Title:       "Updated",
		Description: "Updated description",
		CategoryID:  42,
	})
	if err != nil {
		t.Fatalf("UpdateAdvert error: %v", err)
	}
	if advert.ID != 12345 {
		t.Errorf("ID = %d, want 12345", advert.ID)
	}
	if advert.Title != "Updated" {
		t.Errorf("Title = %q, want %q", advert.Title, "Updated")
	}
	if advert.Status != "active" {
		t.Errorf("Status = %q, want %q", advert.Status, "active")
	}
}

func TestDeleteAdvert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/adverts/12345" {
			t.Errorf("path = %q, want /adverts/12345", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Adverts.DeleteAdvert(context.Background(), 12345)
	if err != nil {
		t.Fatalf("DeleteAdvert error: %v", err)
	}
}

func TestRunCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/adverts/12345/commands" {
			t.Errorf("path = %q, want /adverts/12345/commands", r.URL.Path)
		}

		var body struct {
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Command != "activate" {
			t.Errorf("body.Command = %q, want %q", body.Command, "activate")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Adverts.RunCommand(context.Background(), 12345, AdvertCommandRequest{
		Command: "activate",
	})
	if err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
}

func TestRunCommandDeactivate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adverts/12345/commands" {
			t.Errorf("path = %q, want /adverts/12345/commands", r.URL.Path)
		}

		var body struct {
			Command   string `json:"command"`
			IsSuccess bool   `json:"is_success"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Command != "deactivate" {
			t.Errorf("body.Command = %q, want %q", body.Command, "deactivate")
		}
		if !body.IsSuccess {
			t.Error("body.IsSuccess = false, want true")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Adverts.RunCommand(context.Background(), 12345, AdvertCommandRequest{
		Command:   "deactivate",
		IsSuccess: true,
	})
	if err != nil {
		t.Fatalf("RunCommand deactivate error: %v", err)
	}
}

func TestListCategories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/categories" {
			t.Errorf("path = %q, want /categories", r.URL.Path)
		}
		if pid := r.URL.Query().Get("parent_id"); pid != "5" {
			t.Errorf("parent_id = %q, want %q", pid, "5")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":100,"name":"Elektronika","is_leaf":false}]}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Categories.ListCategories(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListCategories error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].ID != 100 {
		t.Errorf("ID = %d, want 100", result.Data[0].ID)
	}
	if result.Data[0].Name != "Elektronika" {
		t.Errorf("Name = %q, want %q", result.Data[0].Name, "Elektronika")
	}
	if result.Data[0].IsLeaf != false {
		t.Errorf("IsLeaf = %v, want false", result.Data[0].IsLeaf)
	}
}

func TestGetCategory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/categories/100" {
			t.Errorf("path = %q, want /categories/100", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":100,"name":"Elektronika","is_leaf":false,"photos_limit":8}}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	cat, err := c.Categories.GetCategory(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetCategory error: %v", err)
	}
	if cat.ID != 100 {
		t.Errorf("ID = %d, want 100", cat.ID)
	}
	if cat.Name != "Elektronika" {
		t.Errorf("Name = %q, want %q", cat.Name, "Elektronika")
	}
	if cat.IsLeaf != false {
		t.Errorf("IsLeaf = %v, want false", cat.IsLeaf)
	}
	if cat.PhotosLimit != 8 {
		t.Errorf("PhotosLimit = %d, want 8", cat.PhotosLimit)
	}
}

func TestGetCategoryAttributes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/categories/100/attributes" {
			t.Errorf("path = %q, want /categories/100/attributes", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"code":"state","label":"Stan","validation":{"type":"attribute","required":true}}]}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Categories.GetAttributes(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetAttributes error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].Code != "state" {
		t.Errorf("Code = %q, want %q", result.Data[0].Code, "state")
	}
	if result.Data[0].Label != "Stan" {
		t.Errorf("Label = %q, want %q", result.Data[0].Label, "Stan")
	}
	if result.Data[0].Validation.Type != "attribute" {
		t.Errorf("Validation.Type = %q, want %q", result.Data[0].Validation.Type, "attribute")
	}
	if !result.Data[0].Validation.Required {
		t.Error("Validation.Required = false, want true")
	}
}

func TestSuggestCategory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if !containsPath(r.URL.Path, "/categories/suggestion") {
			t.Errorf("path = %q, want /categories/suggestion", r.URL.Path)
		}
		if q := r.URL.Query().Get("q"); q != "telefon" {
			t.Errorf("q = %q, want %q", q, "telefon")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":200,"name":"Smartfony","path":["Elektronika","Telefony","Smartfony"]}]}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Categories.SuggestCategory(context.Background(), "telefon")
	if err != nil {
		t.Fatalf("SuggestCategory error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].ID != 200 {
		t.Errorf("ID = %d, want 200", result.Data[0].ID)
	}
	if result.Data[0].Name != "Smartfony" {
		t.Errorf("Name = %q, want %q", result.Data[0].Name, "Smartfony")
	}
	if len(result.Data[0].Path) != 3 {
		t.Fatalf("len(Path) = %d, want 3", len(result.Data[0].Path))
	}
	if result.Data[0].Path[0] != "Elektronika" {
		t.Errorf("Path[0] = %q, want %q", result.Data[0].Path[0], "Elektronika")
	}
}

// containsPath is a helper to check URL path when the mock might receive
// path with or without query string components.
func containsPath(urlPath, expected string) bool {
	return urlPath == expected
}

func TestMapTransactionStatus(t *testing.T) {
	tests := []struct {
		olx     string
		wantOMS string
		wantOK  bool
	}{
		{"pending", "new", true},
		{"completed", "confirmed", true},
		{"paid", "confirmed", true},
		{"cancelled", "cancelled", true},
		{"unknown", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.olx, func(t *testing.T) {
			got, ok := MapTransactionStatus(tc.olx)
			if ok != tc.wantOK {
				t.Errorf("MapTransactionStatus(%q) ok = %v, want %v", tc.olx, ok, tc.wantOK)
			}
			if got != tc.wantOMS {
				t.Errorf("MapTransactionStatus(%q) = %q, want %q", tc.olx, got, tc.wantOMS)
			}
		})
	}
}

func TestMapAdvertStatus(t *testing.T) {
	tests := []struct {
		olx      string
		wantSync string
		wantOK   bool
	}{
		{"active", "synced", true},
		{"new", "pending", true},
		{"limited", "error", true},
		{"outdated", "inactive", true},
		{"removed_by_user", "inactive", true},
		{"blocked", "error", true},
		{"unknown", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.olx, func(t *testing.T) {
			got, ok := MapAdvertStatus(tc.olx)
			if ok != tc.wantOK {
				t.Errorf("MapAdvertStatus(%q) ok = %v, want %v", tc.olx, ok, tc.wantOK)
			}
			if got != tc.wantSync {
				t.Errorf("MapAdvertStatus(%q) = %q, want %q", tc.olx, got, tc.wantSync)
			}
		})
	}
}
