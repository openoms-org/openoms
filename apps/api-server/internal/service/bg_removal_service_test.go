package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBGRemovalServiceDefaultsToRemoveBG(t *testing.T) {
	assert.Equal(t, DefaultBGRemovalAPIURL, NewBGRemovalService("key", "").apiURL)
}

func TestBGRemovalServiceUsesConfiguredURL(t *testing.T) {
	var gotPath, gotKey, gotSize string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotSize = r.FormValue("size")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer srv.Close()

	svc := NewBGRemovalService("test-key", srv.URL+"/v1.0/removebg")
	svc.httpClient = srv.Client() // the default client refuses loopback addresses

	out, contentType, err := svc.RemoveBackground(context.Background(), []byte("image"), "product.jpg")
	require.NoError(t, err)
	assert.Equal(t, []byte("png-bytes"), out)
	assert.Equal(t, "image/png", contentType)
	assert.Equal(t, "/v1.0/removebg", gotPath)
	assert.Equal(t, "test-key", gotKey)
	assert.Equal(t, "auto", gotSize)
}
