package ksef

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

// InvoiceService handles invoice operations in KSeF.
type InvoiceService struct {
	client *Client
}

// Send sends a structured invoice XML to KSeF within an active session.
// The invoiceXML should be a valid FA(2) schema XML document.
func (s *InvoiceService) Send(ctx context.Context, sessionToken string, invoiceXML []byte) (*SendInvoiceResponse, error) {
	headers := map[string]string{
		"SessionToken": sessionToken,
	}

	var resp SendInvoiceResponse
	raw, err := s.client.doRawBody(
		ctx,
		http.MethodPut,
		"/online/Invoice/Send",
		bytes.NewReader(invoiceXML),
		"application/octet-stream",
		headers,
	)
	if err != nil {
		return nil, fmt.Errorf("send invoice: %w", err)
	}

	if err := parseJSON(raw, &resp); err != nil {
		return nil, fmt.Errorf("send invoice: decode response: %w", err)
	}

	return &resp, nil
}

// GetStatus checks the processing status of a submitted invoice.
func (s *InvoiceService) GetStatus(ctx context.Context, sessionToken, referenceNumber string) (*InvoiceStatus, error) {
	path := fmt.Sprintf("/online/Invoice/Status/%s", referenceNumber)

	headers := map[string]string{
		"SessionToken": sessionToken,
	}

	var resp InvoiceStatus
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, &resp, headers); err != nil {
		return nil, fmt.Errorf("invoice status: %w", err)
	}

	return &resp, nil
}

// Get downloads a structured invoice from KSeF by its KSeF reference number.
func (s *InvoiceService) Get(ctx context.Context, sessionToken, ksefReferenceNumber string) ([]byte, error) {
	path := fmt.Sprintf("/online/Invoice/Get/%s", ksefReferenceNumber)

	headers := map[string]string{
		"SessionToken": sessionToken,
	}

	raw, err := s.client.doRaw(ctx, http.MethodGet, path, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	return raw, nil
}

// GetUPO downloads the UPO (Urzedowe Poswiadczenie Odbioru) for a session.
// The UPO is the official receipt confirming acceptance of invoices.
func (s *InvoiceService) GetUPO(ctx context.Context, referenceNumber string) (*UPOResponse, error) {
	path := fmt.Sprintf("/common/Status/%s", referenceNumber)

	var resp UPOResponse
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
		return nil, fmt.Errorf("get UPO: %w", err)
	}

	return &resp, nil
}

// GetUPOBytes downloads and decodes the UPO document bytes.
func (s *InvoiceService) GetUPOBytes(ctx context.Context, referenceNumber string) ([]byte, error) {
	upo, err := s.GetUPO(ctx, referenceNumber)
	if err != nil {
		return nil, err
	}

	if upo.UPO == "" {
		return nil, fmt.Errorf("ksef: UPO not yet available (processing code: %d)", upo.ProcessingCode)
	}

	data, err := base64.StdEncoding.DecodeString(upo.UPO)
	if err != nil {
		return nil, fmt.Errorf("ksef: decode UPO: %w", err)
	}

	return data, nil
}
