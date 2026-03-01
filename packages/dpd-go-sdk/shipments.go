package dpd

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

// ShipmentService handles DPD shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates a new shipment via POST /public/shipment/v1/generatePackagesNumbers.
func (s *ShipmentService) Create(ctx context.Context, req *CreateParcelRequest) (*CreateParcelResponse, error) {
	var raw createParcelRawResponse
	if err := s.client.do(ctx, http.MethodPost, "/public/shipment/v1/generatePackagesNumbers", req, &raw); err != nil {
		return nil, fmt.Errorf("dpd: create parcel: %w", err)
	}

	resp := &CreateParcelResponse{
		SessionID: raw.SessionID,
		Status:    raw.Status,
	}
	if len(raw.Packages) > 0 && len(raw.Packages[0].Parcels) > 0 {
		resp.Waybill = raw.Packages[0].Parcels[0].Waybill
		resp.ParcelID = raw.Packages[0].Parcels[0].Reference
	}
	return resp, nil
}

// GetLabel retrieves the shipping label via POST /public/shipment/v1/generateSpedLabels.
// waybillRef is the parcel waybill number returned by Create.
func (s *ShipmentService) GetLabel(ctx context.Context, waybillRef string) ([]byte, error) {
	labelReq := generateLabelRequest{
		DPDServicesParcelsPPLOD: []labelParcel{
			{WaybillRef: waybillRef},
		},
		OutputDocFormat: "PDF",
		OutputDocPage:   "A4",
	}

	var resp generateLabelResponse
	if err := s.client.do(ctx, http.MethodPost, "/public/shipment/v1/generateSpedLabels", labelReq, &resp); err != nil {
		return nil, fmt.Errorf("dpd: get label: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.DocumentData)
	if err != nil {
		return nil, fmt.Errorf("dpd: decode label data: %w", err)
	}
	return data, nil
}

// GetTracking is not available in the DPD REST API.
// Use the DPD tracking portal at https://tracktrace.dpd.com.pl/.
func (s *ShipmentService) GetTracking(_ context.Context, _ string) (*TrackingResponse, error) {
	return nil, fmt.Errorf("dpd: tracking not available via DPD REST API — use DPD tracking portal at tracktrace.dpd.com.pl")
}

// Cancel attempts to cancel a parcel. Returns an error if the carrier does not support it.
func (s *ShipmentService) Cancel(_ context.Context, _ string) error {
	return fmt.Errorf("dpd: parcel cancellation not supported via DPD REST API — contact DPD support")
}
