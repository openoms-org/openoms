package amazon

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClientWithAPI(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"test-at","refresh_token":"test-rt","expires_in":3600}`)
	}))
	client := NewClient("test_id", "test_secret", //nolint:gosec // test credentials
		WithBaseURL(srv.URL),
		WithTokenEndpoint(tokenSrv.URL),
		WithTokens("test-at", "test-rt", time.Now().Add(time.Hour)),
	)
	return client, srv
}

func TestCreateDocument(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/feeds/2021-06-30/documents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "test-at", r.Header.Get("x-amz-access-token"))

		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"contentType":"text/xml; charset=UTF-8"`)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"feedDocumentId":"doc-123","url":"https://s3.example.com/upload"}`)
	})

	client, srv := newTestClientWithAPI(mux)
	defer srv.Close()

	resp, err := client.Feeds.CreateDocument(context.Background(), "text/xml; charset=UTF-8")
	require.NoError(t, err)
	assert.Equal(t, "doc-123", resp.FeedDocumentID)
	assert.Equal(t, "https://s3.example.com/upload", resp.URL)
}

func TestUpload(t *testing.T) {
	var receivedBody string
	var receivedContentType string
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		// Upload should NOT have x-amz-access-token header
		assert.Empty(t, r.Header.Get("x-amz-access-token"))
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadSrv.Close()

	client := NewClient("test_id", "test_secret") //nolint:gosec // test credentials
	err := client.Feeds.Upload(context.Background(), uploadSrv.URL, []byte("<xml>test</xml>"), "text/xml; charset=UTF-8")
	require.NoError(t, err)
	assert.Equal(t, "<xml>test</xml>", receivedBody)
	assert.Equal(t, "text/xml; charset=UTF-8", receivedContentType)
}

func TestUploadError(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Access Denied")
	}))
	defer uploadSrv.Close()

	client := NewClient("test_id", "test_secret") //nolint:gosec // test credentials
	err := client.Feeds.Upload(context.Background(), uploadSrv.URL, []byte("<xml/>"), "text/xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestSubmitFeed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/feeds/2021-06-30/feeds", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"feedType":"POST_INVENTORY_AVAILABILITY_DATA"`)
		assert.Contains(t, string(body), `"inputFeedDocumentId":"doc-123"`)
		assert.Contains(t, string(body), `"marketplaceIds":["A1C3SOZRARQ6R3"]`)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"feedId":"feed-456"}`)
	})

	client, srv := newTestClientWithAPI(mux)
	defer srv.Close()

	resp, err := client.Feeds.SubmitFeed(context.Background(), "POST_INVENTORY_AVAILABILITY_DATA", []string{"A1C3SOZRARQ6R3"}, "doc-123")
	require.NoError(t, err)
	assert.Equal(t, "feed-456", resp.FeedID)
}

func TestGetFeed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/feeds/2021-06-30/feeds/feed-456", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"feedId":"feed-456","feedType":"POST_INVENTORY_AVAILABILITY_DATA","processingStatus":"DONE"}`)
	})

	client, srv := newTestClientWithAPI(mux)
	defer srv.Close()

	feed, err := client.Feeds.GetFeed(context.Background(), "feed-456")
	require.NoError(t, err)
	assert.Equal(t, "feed-456", feed.FeedID)
	assert.Equal(t, "DONE", feed.ProcessingStatus)
}

func TestSubmitInventoryFeed(t *testing.T) {
	var uploadedXML string

	// Upload server (simulates S3 presigned URL)
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		uploadedXML = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadSrv.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/feeds/2021-06-30/documents", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"feedDocumentId":"doc-789","url":"%s"}`, uploadSrv.URL)
	})
	mux.HandleFunc("/feeds/2021-06-30/feeds", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"inputFeedDocumentId":"doc-789"`)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"feedId":"feed-final"}`)
	})

	client, srv := newTestClientWithAPI(mux)
	defer srv.Close()

	feedID, err := client.Feeds.SubmitInventoryFeed(context.Background(), "A1C3SOZRARQ6R3", "MERCHANT_001", map[string]int{
		"SKU-A": 10,
		"SKU-B": 0,
	})
	require.NoError(t, err)
	assert.Equal(t, "feed-final", feedID)

	// Verify XML structure
	assert.True(t, strings.HasPrefix(uploadedXML, `<?xml version="1.0" encoding="UTF-8"?>`))
	assert.Contains(t, uploadedXML, "<MerchantIdentifier>MERCHANT_001</MerchantIdentifier>")
	assert.Contains(t, uploadedXML, "<MessageType>Inventory</MessageType>")
	assert.Contains(t, uploadedXML, "<SKU>SKU-A</SKU>")
	assert.Contains(t, uploadedXML, "<Quantity>10</Quantity>")
	assert.Contains(t, uploadedXML, "<SKU>SKU-B</SKU>")
	assert.Contains(t, uploadedXML, "<Quantity>0</Quantity>")
}

func TestBuildInventoryXML(t *testing.T) {
	data := buildInventoryXML("MERCH_123", map[string]int{
		"B-SKU": 5,
		"A-SKU": 15,
	})

	xmlStr := string(data)
	assert.Contains(t, xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, xmlStr, "<MerchantIdentifier>MERCH_123</MerchantIdentifier>")

	// Verify sorted order: A-SKU (MessageID 1) before B-SKU (MessageID 2)
	var env amazonEnvelope
	// Strip xml header for parsing
	xmlBody := xmlStr[strings.Index(xmlStr, "<AmazonEnvelope"):]
	err := xml.Unmarshal([]byte(xmlBody), &env)
	require.NoError(t, err)

	assert.Equal(t, 2, len(env.Messages))
	assert.Equal(t, 1, env.Messages[0].MessageID)
	assert.Equal(t, "A-SKU", env.Messages[0].Inventory.SKU)
	assert.Equal(t, 15, env.Messages[0].Inventory.Quantity)
	assert.Equal(t, 2, env.Messages[1].MessageID)
	assert.Equal(t, "B-SKU", env.Messages[1].Inventory.SKU)
	assert.Equal(t, 5, env.Messages[1].Inventory.Quantity)
}
