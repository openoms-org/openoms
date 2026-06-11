package apaczka

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// fixedNow returns a deterministic unix timestamp for tests.
func fixedNow() int64 { return 1700000000 }

// TestSign asserts sign() against a precomputed golden HMAC-SHA256 value.
// The golden hex was computed independently (Python) so this test fails if the
// signing algorithm, message layout, or encoding regresses:
//
//	hmac.new(b"secret", b"appid:service_structure/:{}:1700000000", sha256).hexdigest()
func TestSign(t *testing.T) {
	const (
		appID     = "appid"
		route     = "service_structure/"
		request   = "{}"
		expires   = int64(1700000000)
		appSecret = "secret"

		golden = "73d444c0552eedb0e362e3aa84607efa5a3a2c187843e843f5865545d1e77988"
	)

	got := sign(appID, route, request, expires, appSecret)
	if got != golden {
		t.Errorf("sign() = %q, want golden %q", got, golden)
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("appid", "secret")
	if c.appID != "appid" {
		t.Errorf("appID = %q, want %q", c.appID, "appid")
	}
	if c.appSecret != "secret" {
		t.Errorf("appSecret = %q, want %q", c.appSecret, "secret")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.httpClient != http.DefaultClient {
		t.Error("expected default http.Client")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("a", "b", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Error("expected custom http client")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("a", "b", WithBaseURL("http://localhost:9999/"))
	if c.baseURL != "http://localhost:9999/" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:9999/")
	}
}

func TestWithNow(t *testing.T) {
	c := NewClient("a", "b", WithNow(fixedNow))
	if c.now() != 1700000000 {
		t.Errorf("now() = %d, want 1700000000", c.now())
	}
}

// TestDoSendsAllFormFields verifies the four required form fields are present
// and that the dispatched signature matches a precomputed golden value.
// goldenSig was computed independently (Python) for:
//
//	hmac.new(b"testsecret", b"testapp:service_structure/:{}:1700001800", sha256).hexdigest()
//
// where expires = fixedNow()+1800 = 1700001800.
func TestDoSendsAllFormFields(t *testing.T) {
	const appID = "testapp"
	const appSecret = "testsecret"
	const route = "service_structure/"
	const goldenSig = "19acf90a6000de69bed248ea641c799da786e8164d31683103846abbf004d2e2"

	var (
		gotAppID   string
		gotRequest string
		gotExpires string
		gotSig     string
		gotCT      string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", 400)
			return
		}
		gotAppID = r.FormValue("app_id")
		gotRequest = r.FormValue("request")
		gotExpires = r.FormValue("expires")
		gotSig = r.FormValue("signature")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":200,"response":{}}`)
	}))
	defer srv.Close()

	c := NewClient(appID, appSecret,
		WithBaseURL(srv.URL+"/"),
		WithNow(fixedNow),
	)
	_ = c.do(context.Background(), route, nil, nil)

	if gotCT != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotCT)
	}
	if gotAppID != appID {
		t.Errorf("app_id = %q, want %q", gotAppID, appID)
	}
	if gotRequest != "{}" {
		t.Errorf("request = %q, want {}", gotRequest)
	}

	expiresInt, _ := strconv.ParseInt(gotExpires, 10, 64)
	if expiresInt != fixedNow()+1800 {
		t.Errorf("expires = %d, want %d", expiresInt, fixedNow()+1800)
	}

	if gotSig != goldenSig {
		t.Errorf("signature = %q, want golden %q", gotSig, goldenSig)
	}
}

func TestDoDecodesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":200,"response":{"key":"value"}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	var result map[string]string
	if err := c.do(context.Background(), "test/", nil, &result); err != nil {
		t.Fatalf("do() error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("result[key] = %q, want %q", result["key"], "value")
	}
}

func TestDoReturnsAPIErrorOn403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":403,"message":"forbidden"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	err := c.do(context.Background(), "test/", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 403 {
		t.Errorf("Status = %d, want 403", apiErr.Status)
	}
	if apiErr.Message != "forbidden" {
		t.Errorf("Message = %q, want forbidden", apiErr.Message)
	}
}

func TestDoReturnsAPIErrorOnNon200Status(t *testing.T) {
	tests := []struct {
		status  int
		message string
	}{
		{401, "unauthorized"},
		{500, "internal error"},
		{422, "validation error"},
	}
	for _, tc := range tests {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"status":%d,"message":%q}`, tc.status, tc.message)
			}))
			defer srv.Close()

			c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
			err := c.do(context.Background(), "test/", nil, nil)
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.Status != tc.status {
				t.Errorf("Status = %d, want %d", apiErr.Status, tc.status)
			}
		})
	}
}

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		err  APIError
		want string
	}{
		{APIError{Status: 403, Message: "x"}, "apaczka: api error 403: x"},
		{APIError{Status: 500}, "apaczka: api error 500"},
	}
	for _, tc := range tests {
		got := tc.err.Error()
		if got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}

func TestDoContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":200,"response":{}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.do(ctx, "test/", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// TestDoSendsPayloadAsJSONString verifies that a non-nil payload is marshalled
// and sent as the "request" form field.
func TestDoSendsPayloadAsJSONString(t *testing.T) {
	var gotRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		vals, _ := url.ParseQuery(string(body))
		gotRequest = vals.Get("request")
		fmt.Fprint(w, `{"status":200,"response":{}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	payload := map[string]string{"foo": "bar"}
	_ = c.do(context.Background(), "test/", payload, nil)

	if !strings.Contains(gotRequest, `"foo"`) {
		t.Errorf("request field %q does not contain payload key", gotRequest)
	}
}
