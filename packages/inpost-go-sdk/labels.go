package inpost

import (
	"context"
	"fmt"
	"net/http"
)

// LabelFormat specifies the label output format.
type LabelFormat string

const (
	LabelPDF LabelFormat = "Pdf"
	LabelZPL LabelFormat = "Zpl"
	LabelEPL LabelFormat = "Epl"
)

// LabelPageFormat specifies the label page size.
type LabelPageFormat string

const (
	LabelPageNormal LabelPageFormat = "normal" // A6
	LabelPageA4     LabelPageFormat = "A4"
)

// LabelService handles label-related API operations.
type LabelService struct {
	client *Client
}

type generateLabelsRequest struct {
	ShipmentIDs []int64 `json:"shipment_ids"`
	Format      string  `json:"format"`
	Type        string  `json:"type"`
}

// Generate creates labels for one or more shipments. Returns raw binary data.
func (s *LabelService) Generate(ctx context.Context, shipmentIDs []int64, format LabelFormat, pageFormat LabelPageFormat) ([]byte, error) {
	body := generateLabelsRequest{
		ShipmentIDs: shipmentIDs,
		Format:      string(format),
		Type:        string(pageFormat),
	}

	raw, _, err := s.client.doRaw(ctx, http.MethodPost, "/v1/shipments/labels", body)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Get retrieves the label for a single shipment. Returns raw binary data.
func (s *LabelService) Get(ctx context.Context, shipmentID int64, format LabelFormat) ([]byte, error) {
	path := fmt.Sprintf("/v1/shipments/%d/label?format=%s", shipmentID, format)

	raw, _, err := s.client.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
