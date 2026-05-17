package dpd

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

const infoServicesLanguage = "EN"

type infoServicesSOAPEnvelope struct {
	Body infoServicesSOAPBody `xml:"Body"`
}

type infoServicesSOAPBody struct {
	Fault    *infoServicesSOAPFault    `xml:"Fault"`
	Response infoServicesWaybillResult `xml:"getEventsForWaybillV1Response"`
}

type infoServicesSOAPFault struct {
	FaultString string `xml:"faultstring"`
}

type infoServicesWaybillResult struct {
	Return infoServicesWaybillReturn `xml:"return"`
}

type infoServicesWaybillReturn struct {
	Events []infoServicesEvent `xml:"eventsList"`
}

type infoServicesEvent struct {
	BusinessCode     string `xml:"businessCode"`
	Country          string `xml:"country"`
	DepotName        string `xml:"depotName"`
	Description      string `xml:"description"`
	EventTime        string `xml:"eventTime"`
	Waybill          string `xml:"waybill"`
	PackageReference string `xml:"packageReference"`
	ParcelReference  string `xml:"parcelReference"`
}

func (c *Client) getInfoServicesTracking(ctx context.Context, waybill string) (*TrackingResponse, error) {
	infoChannel := strings.TrimSpace(c.infoChannel)
	infoLogin := strings.TrimSpace(c.infoLogin)
	infoPassword := strings.TrimSpace(c.infoPassword)
	waybill = strings.TrimSpace(waybill)

	if infoChannel == "" {
		return nil, fmt.Errorf("dpd: info services channel is required for tracking")
	}
	if infoLogin == "" || infoPassword == "" {
		return nil, fmt.Errorf("dpd: info services login and password are required for tracking")
	}
	if waybill == "" {
		return nil, fmt.Errorf("dpd: waybill is required for tracking")
	}

	body := buildInfoServicesWaybillEnvelope(waybill, infoLogin, infoPassword, infoChannel)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.infoServicesBaseURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("dpd: create info services request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Accept", "text/xml")
	req.Header.Set("SOAPAction", "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dpd: info services request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("dpd: read info services response: %w", err)
	}
	var envelope infoServicesSOAPEnvelope
	parseErr := xml.Unmarshal(respBody, &envelope)
	if envelope.Body.Fault != nil {
		msg := strings.TrimSpace(envelope.Body.Fault.FaultString)
		if msg == "" {
			msg = "unknown SOAP fault"
		}
		return nil, fmt.Errorf("dpd: info services fault: %s", msg)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("dpd: info services api error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if parseErr != nil {
		return nil, fmt.Errorf("dpd: parse info services response: %w", parseErr)
	}

	events := envelope.Body.Response.Return.Events
	trackingEvents := make([]TrackingEvent, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		eventTime, err := parseInfoServicesTime(ev.EventTime)
		if err != nil {
			return nil, fmt.Errorf("dpd: parse info services event time %q: %w", ev.EventTime, err)
		}
		trackingEvents = append(trackingEvents, TrackingEvent{
			Status:      strings.TrimSpace(ev.BusinessCode),
			Description: strings.TrimSpace(ev.Description),
			Location:    formatInfoServicesLocation(ev.DepotName, ev.Country),
			DateTime:    eventTime,
			Waybill:     strings.TrimSpace(ev.Waybill),
		})
	}

	return &TrackingResponse{Events: trackingEvents}, nil
}

func buildInfoServicesWaybillEnvelope(waybill, login, password, channel string) string {
	var b bytes.Buffer
	b.WriteString(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:even="http://events.dpdinfoservices.dpd.com.pl/">`)
	b.WriteString(`<soapenv:Header/>`)
	b.WriteString(`<soapenv:Body>`)
	b.WriteString(`<even:getEventsForWaybillV1>`)
	writeXMLNode(&b, "waybill", waybill)
	writeXMLNode(&b, "eventsSelectType", "ONLY_LAST")
	writeXMLNode(&b, "language", infoServicesLanguage)
	b.WriteString(`<authDataV1>`)
	writeXMLNode(&b, "channel", channel)
	writeXMLNode(&b, "login", login)
	writeXMLNode(&b, "password", password)
	b.WriteString(`</authDataV1>`)
	b.WriteString(`</even:getEventsForWaybillV1>`)
	b.WriteString(`</soapenv:Body>`)
	b.WriteString(`</soapenv:Envelope>`)
	return b.String()
}

func writeXMLNode(b *bytes.Buffer, name, value string) {
	b.WriteByte('<')
	b.WriteString(name)
	b.WriteByte('>')
	b.WriteString(html.EscapeString(value))
	b.WriteString("</")
	b.WriteString(name)
	b.WriteByte('>')
}

func parseInfoServicesTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			if t.Location() == time.Local {
				return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format")
}

func formatInfoServicesLocation(depotName, country string) string {
	depotName = strings.TrimSpace(depotName)
	country = strings.TrimSpace(country)
	switch {
	case depotName != "" && country != "":
		return depotName + ", " + country
	case depotName != "":
		return depotName
	default:
		return country
	}
}
