package dpd

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const dpdInfoServicesOKResponse = `<?xml version="1.0" encoding="UTF-8"?>
<S:Envelope xmlns:S="http://schemas.xmlsoap.org/soap/envelope/">
  <S:Body>
    <ns2:getEventsForWaybillV1Response xmlns:ns2="http://events.dpdinfoservices.dpd.com.pl/">
      <return>
        <confirmId>0</confirmId>
        <eventsList>
          <businessCode>190101</businessCode>
          <country>PL</country>
          <depot>1357</depot>
          <depotName>KATOWICE 2</depotName>
          <description>Parcel delivered</description>
          <eventTime>2026-05-17T12:15:30.123</eventTime>
          <objectId>1357500000032543569</objectId>
          <packageReference>ORDER-1</packageReference>
          <parcelReference/>
          <waybill>0000012345678</waybill>
        </eventsList>
        <eventsList>
          <businessCode>040101</businessCode>
          <country>PL</country>
          <depot>1322</depot>
          <depotName>Katowice</depotName>
          <description>Parcel collected by courier</description>
          <eventTime>2026-05-17T09:00:00</eventTime>
          <objectId>1322500000032543000</objectId>
          <packageReference>ORDER-1</packageReference>
          <parcelReference/>
          <waybill>0000012345678</waybill>
        </eventsList>
      </return>
    </ns2:getEventsForWaybillV1Response>
  </S:Body>
</S:Envelope>`

func TestInfoServicesGetTracking_SendsWaybillSOAPRequest(t *testing.T) {
	var gotMethod, gotBody, gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(dpdInfoServicesOKResponse))
	}))
	defer srv.Close()

	c := NewClient("rest-user", "rest-pass", "FID123",
		WithInfoServicesBaseURL(srv.URL),
		WithInfoServicesCredentials("info-user", "info-pass", "channel-1"),
		WithHTTPClient(srv.Client()),
	)

	if _, err := c.Shipments.GetTracking(context.Background(), "0000012345678"); err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if !strings.Contains(gotContentType, "text/xml") {
		t.Fatalf("Content-Type = %q, want text/xml", gotContentType)
	}
	for _, want := range []string{
		"getEventsForWaybillV1",
		"<waybill>0000012345678</waybill>",
		"<eventsSelectType>ONLY_LAST</eventsSelectType>",
		"<language>EN</language>",
		"<login>info-user</login>",
		"<password>info-pass</password>",
		"<channel>channel-1</channel>",
	} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("SOAP body missing %q:\n%s", want, gotBody)
		}
	}
}

func TestInfoServicesGetTracking_ParsesEventsChronologically(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(dpdInfoServicesOKResponse))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID123",
		WithInfoServicesBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.GetTracking(context.Background(), "0000012345678")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(resp.Events))
	}

	first := resp.Events[0]
	if first.Status != "040101" {
		t.Fatalf("first status = %q, want oldest 040101", first.Status)
	}
	if first.Description != "Parcel collected by courier" {
		t.Fatalf("first description = %q", first.Description)
	}
	if first.Location != "Katowice, PL" {
		t.Fatalf("first location = %q, want Katowice, PL", first.Location)
	}
	if first.Waybill != "0000012345678" {
		t.Fatalf("first waybill = %q", first.Waybill)
	}
	wantTime := time.Date(2026, 5, 17, 9, 0, 0, 0, time.UTC)
	if !first.DateTime.Equal(wantTime) {
		t.Fatalf("first DateTime = %s, want %s", first.DateTime, wantTime)
	}

	last := resp.Events[1]
	if last.Status != "190101" {
		t.Fatalf("last status = %q, want newest 190101", last.Status)
	}
	if last.Location != "KATOWICE 2, PL" {
		t.Fatalf("last location = %q, want KATOWICE 2, PL", last.Location)
	}
}

func TestInfoServicesGetTracking_ReturnsSOAPFault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<S:Envelope xmlns:S="http://schemas.xmlsoap.org/soap/envelope/">
  <S:Body>
    <S:Fault>
      <faultcode>S:Server</faultcode>
      <faultstring>Access denied to secured webserwis method</faultstring>
    </S:Fault>
  </S:Body>
</S:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "bad-pass", "FID123",
		WithInfoServicesBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "0000012345678")
	if err == nil {
		t.Fatal("expected SOAP fault error")
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Fatalf("error = %v, want SOAP fault text", err)
	}
}

func TestInfoServicesGetTracking_RequiresInfoChannel(t *testing.T) {
	c := NewClient("user", "pass", "")

	_, err := c.Shipments.GetTracking(context.Background(), "0000012345678")
	if err == nil {
		t.Fatal("expected configuration error")
	}
	if !strings.Contains(err.Error(), "info services channel") {
		t.Fatalf("error = %v, want missing info services channel", err)
	}
}
