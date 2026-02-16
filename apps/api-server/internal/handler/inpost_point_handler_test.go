package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	inpost "github.com/openoms-org/openoms/packages/inpost-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInPostPointHandler_Search_MissingQuery(t *testing.T) {
	h := NewInPostPointHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "query parameter is required", resp["error"])
}

func TestInPostPointHandler_Search_QueryTooShort(t *testing.T) {
	h := NewInPostPointHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=a", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "query must be at least 2 characters", resp["error"])
}

func TestInPostPointHandler_Search_EmptyQuery(t *testing.T) {
	h := NewInPostPointHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "query parameter is required", resp["error"])
}

func TestInPostPointHandler_Search_ExactlyTwoCharsAccepted(t *testing.T) {
	// A query with exactly 2 characters should pass validation and reach the API call.
	// We set up a fake InPost Points API server to handle the request.
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items:      []inpost.Point{},
			Count:      0,
			Page:       1,
			PerPage:    10,
			TotalPages: 0,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=ab", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInPostPointHandler_Search_SuccessWithResults(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "city=Krakow")
		assert.Contains(t, r.URL.String(), "type=parcel_locker")
		assert.Contains(t, r.URL.String(), "per_page=10")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items: []inpost.Point{
				{
					Name:   "KRA01M",
					Type:   []string{"parcel_locker"},
					Status: "Operating",
					Address: inpost.PointAddress{
						Line1: "ul. Krakowska 1",
						Line2: "30-001 Krakow",
					},
					Location: inpost.PointLocation{
						Latitude:  50.0614,
						Longitude: 19.9383,
					},
					LocationDescription: "Przy wejsciu do galerii",
					OpeningHours:        "24/7",
				},
				{
					Name:   "KRA02M",
					Type:   []string{"parcel_locker"},
					Status: "Operating",
					Address: inpost.PointAddress{
						Line1: "ul. Wielicka 28",
						Line2: "30-552 Krakow",
					},
					Location: inpost.PointLocation{
						Latitude:  50.0452,
						Longitude: 19.9612,
					},
				},
			},
			Count:      2,
			Page:       1,
			PerPage:    10,
			TotalPages: 1,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Krakow", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Count)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "KRA01M", resp.Items[0].Name)
	assert.Equal(t, "KRA02M", resp.Items[1].Name)
}

func TestInPostPointHandler_Search_EmptyResults(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items:      []inpost.Point{},
			Count:      0,
			Page:       1,
			PerPage:    10,
			TotalPages: 0,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=NonexistentCity", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Count)
	assert.Empty(t, resp.Items)
}

func TestInPostPointHandler_Search_ServiceError(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Krakow", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed to search InPost points", resp["error"])
}

func TestInPostPointHandler_Search_ServiceUnauthorized(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid token"})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("bad-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Warszawa", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	// Any API error from InPost is mapped to 502 BadGateway by the handler
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed to search InPost points", resp["error"])
}

func TestInPostPointHandler_Search_PointCodeQuery(t *testing.T) {
	// When the query looks like a point code (e.g. "KRA01M"), the SDK searches by name
	// instead of by city. Verify the correct parameter is sent.
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "name=KRA01M")
		assert.NotContains(t, r.URL.String(), "city=")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items: []inpost.Point{
				{
					Name:   "KRA01M",
					Type:   []string{"parcel_locker"},
					Status: "Operating",
				},
			},
			Count:      1,
			Page:       1,
			PerPage:    10,
			TotalPages: 1,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=KRA01M", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Count)
	assert.Equal(t, "KRA01M", resp.Items[0].Name)
}

func TestInPostPointHandler_Search_POPPointType(t *testing.T) {
	// The handler currently hardcodes PointTypeParcelLocker, but verify the response
	// can include points of different types from the API.
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items: []inpost.Point{
				{
					Name:   "POP-KRA001",
					Type:   []string{"pop"},
					Status: "Operating",
					Address: inpost.PointAddress{
						Line1: "Sklep Zabka, ul. Nowa 5",
						Line2: "30-100 Krakow",
					},
				},
			},
			Count:      1,
			Page:       1,
			PerPage:    10,
			TotalPages: 1,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Krakow", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Contains(t, resp.Items[0].Type, "pop")
}

func TestInPostPointHandler_Search_MixedPointTypes(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items: []inpost.Point{
				{
					Name: "KRA45A",
					Type: []string{"parcel_locker"},
				},
				{
					Name: "POP-KRA100",
					Type: []string{"pop"},
				},
				{
					Name: "KRA99M",
					Type: []string{"parcel_locker", "pop"},
				},
			},
			Count:      3,
			Page:       1,
			PerPage:    10,
			TotalPages: 1,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Krakow", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Count)
	assert.Len(t, resp.Items, 3)
}

func TestInPostPointHandler_Search_ConnectionError(t *testing.T) {
	// Create a client pointing to a closed server to simulate a connection error.
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	pointsSrvURL := pointsSrv.URL
	pointsSrv.Close() // Close immediately to cause connection refused

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrvURL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Krakow", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed to search InPost points", resp["error"])
}

func TestInPostPointHandler_Search_LargeResultSet(t *testing.T) {
	points := make([]inpost.Point, 10)
	for i := range points {
		points[i] = inpost.Point{
			Name:   "KRA" + string(rune('A'+i)) + "1M",
			Type:   []string{"parcel_locker"},
			Status: "Operating",
		}
	}

	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items:      points,
			Count:      10,
			Page:       1,
			PerPage:    10,
			TotalPages: 3,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Krakow", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 10, resp.Count)
	assert.Len(t, resp.Items, 10)
	assert.Equal(t, 3, resp.TotalPages)
}

func TestInPostPointHandler_Search_PointWithAddressDetails(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items: []inpost.Point{
				{
					Name:   "WAW123M",
					Type:   []string{"parcel_locker"},
					Status: "Operating",
					Address: inpost.PointAddress{
						Line1: "ul. Marszalkowska 1",
						Line2: "00-001 Warszawa",
					},
					AddressDetails: &inpost.PointAddressDetails{
						City:           "Warszawa",
						Province:       "Mazowieckie",
						PostCode:       "00-001",
						Street:         "Marszalkowska",
						BuildingNumber: "1",
					},
					Location: inpost.PointLocation{
						Latitude:  52.2297,
						Longitude: 21.0122,
					},
				},
			},
			Count:      1,
			Page:       1,
			PerPage:    10,
			TotalPages: 1,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Warszawa", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp inpost.PointSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "WAW123M", resp.Items[0].Name)
	require.NotNil(t, resp.Items[0].AddressDetails)
	assert.Equal(t, "Warszawa", resp.Items[0].AddressDetails.City)
	assert.Equal(t, "00-001", resp.Items[0].AddressDetails.PostCode)
}

func TestInPostPointHandler_Search_ResponseContentType(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inpost.PointSearchResponse{
			Items:      []inpost.Point{},
			Count:      0,
			Page:       1,
			PerPage:    10,
			TotalPages: 0,
		})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=Gdansk", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
}

func TestInPostPointHandler_Search_ApiReturns422Validation(t *testing.T) {
	pointsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "validation failed"})
	}))
	defer pointsSrv.Close()

	client := inpost.NewClient("test-token", "org-1", inpost.WithPointsBaseURL(pointsSrv.URL))
	h := NewInPostPointHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=test", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed to search InPost points", resp["error"])
}
