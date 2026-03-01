package dhl

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"
)

// ShipmentService handles DHL24 shipment-related SOAP operations.
type ShipmentService struct {
	client *Client
}

// --- Create Shipment ---

type createShipmentsRequest struct {
	XMLName   xml.Name        `xml:"createShipments"`
	AuthData  authData        `xml:"authData"`
	Shipments []soapShipment  `xml:"shipments"`
}

type soapShipment struct {
	XMLName     xml.Name    `xml:"ShipmentRequest"`
	Shipper     soapParty   `xml:"shipper"`
	Receiver    soapParty   `xml:"receiver"`
	PieceList   []soapPiece `xml:"pieceList>Piece"`
	Service     string      `xml:"service,omitempty"`
	ServiceType string      `xml:"serviceType,omitempty"`
	Reference   string      `xml:"reference,omitempty"`
	Content     string      `xml:"content,omitempty"`
	ShipperRef  string      `xml:"shipperAccountNumber,omitempty"`
	COD         *soapCOD    `xml:"cod,omitempty"`
	Insurance   *soapCOD    `xml:"insurance,omitempty"`
}

type soapParty struct {
	Name       string `xml:"address>name,omitempty"`
	PostalCode string `xml:"address>postalCode,omitempty"`
	City       string `xml:"address>city,omitempty"`
	Street     string `xml:"address>street,omitempty"`
	HouseNo    string `xml:"address>houseNo,omitempty"`
	Country    string `xml:"address>country,omitempty"`
	Phone      string `xml:"address>phone,omitempty"`
	Email      string `xml:"address>email,omitempty"`
}

type soapPiece struct {
	Type     string  `xml:"type,omitempty"`
	Width    float64 `xml:"width,omitempty"`
	Height   float64 `xml:"height,omitempty"`
	Length   float64 `xml:"length,omitempty"`
	Weight   float64 `xml:"weight"`
	Quantity int     `xml:"quantity,omitempty"`
}

type soapCOD struct {
	Amount   float64 `xml:"amount"`
	Currency string  `xml:"currency"`
}

type createShipmentsResponse struct {
	XMLName    xml.Name `xml:"Envelope"`
	ShipmentID string   `xml:"Body>createShipmentsResponse>shipmentId"`
	Tracking   string   `xml:"Body>createShipmentsResponse>trackingNumber"`
	Status     string   `xml:"Body>createShipmentsResponse>status"`
}

// Create creates a new shipment via SOAP createShipments.
func (s *ShipmentService) Create(ctx context.Context, req *CreateShipmentRequest) (*ShipmentResponse, error) {
	svcType := req.ServiceType
	if svcType == "" {
		svcType = "AH"
	}

	shipment := soapShipment{
		ShipperRef:  req.ShipperAccount,
		Service:     svcType,
		ServiceType: svcType,
		Reference:   req.Reference,
		Content:     req.Content,
		Receiver: soapParty{
			Name:       req.Receiver.Name,
			Street:     req.Receiver.Street,
			HouseNo:    req.Receiver.HouseNo,
			City:       req.Receiver.City,
			PostalCode: req.Receiver.PostalCode,
			Country:    req.Receiver.Country,
			Phone:      req.Receiver.Phone,
			Email:      req.Receiver.Email,
		},
		Shipper: soapParty{
			Name:       req.Shipper.Name,
			Street:     req.Shipper.Street,
			HouseNo:    req.Shipper.HouseNo,
			City:       req.Shipper.City,
			PostalCode: req.Shipper.PostalCode,
			Country:    req.Shipper.Country,
		},
		PieceList: []soapPiece{{
			Weight:   req.Piece.Weight,
			Width:    req.Piece.Width,
			Height:   req.Piece.Height,
			Length:   req.Piece.Length,
			Quantity: req.Piece.Quantity,
		}},
	}

	if req.COD != nil {
		shipment.COD = &soapCOD{
			Amount:   req.COD.Amount,
			Currency: req.COD.Currency,
		}
	}

	if req.Insurance != nil {
		shipment.Insurance = &soapCOD{
			Amount:   req.Insurance.Amount,
			Currency: req.Insurance.Currency,
		}
	}

	body := createShipmentsRequest{
		AuthData:  authData{Username: s.client.username, Password: s.client.password},
		Shipments: []soapShipment{shipment},
	}

	raw, err := s.client.doSOAP(ctx, "createShipments", body)
	if err != nil {
		return nil, fmt.Errorf("dhl: create shipment: %w", err)
	}

	var resp createShipmentsResponse
	if err := xml.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dhl: parse create response: %w", err)
	}

	return &ShipmentResponse{
		ShipmentID:     resp.ShipmentID,
		TrackingNumber: resp.Tracking,
		Status:         resp.Status,
	}, nil
}

// --- Get Label ---

type getLabelsRequest struct {
	XMLName    xml.Name `xml:"getLabels"`
	AuthData   authData `xml:"authData"`
	ShipmentID string   `xml:"shipmentId"`
	LabelType  string   `xml:"labelType,omitempty"`
}

type getLabelsResponse struct {
	XMLName   xml.Name `xml:"Envelope"`
	LabelData string   `xml:"Body>getLabelsResponse>labelData"`
}

// GetLabel retrieves a shipment label via SOAP getLabels.
func (s *ShipmentService) GetLabel(ctx context.Context, shipmentID string) ([]byte, error) {
	body := getLabelsRequest{
		AuthData:   authData{Username: s.client.username, Password: s.client.password},
		ShipmentID: shipmentID,
		LabelType:  "PDF",
	}

	raw, err := s.client.doSOAP(ctx, "getLabels", body)
	if err != nil {
		return nil, fmt.Errorf("dhl: get label: %w", err)
	}

	var resp getLabelsResponse
	if err := xml.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dhl: parse label response: %w", err)
	}

	if resp.LabelData == "" {
		return nil, fmt.Errorf("dhl: no label data in response")
	}

	data, err := base64.StdEncoding.DecodeString(resp.LabelData)
	if err != nil {
		return nil, fmt.Errorf("dhl: decode label: %w", err)
	}
	return data, nil
}

// --- Get Tracking ---

type getTrackAndTraceInfoRequest struct {
	XMLName        xml.Name `xml:"getTrackAndTraceInfo"`
	AuthData       authData `xml:"authData"`
	TrackingNumber string   `xml:"trackingNumber"`
}

type soapTrackingEvent struct {
	Status    string `xml:"status"`
	Location  string `xml:"location"`
	Timestamp string `xml:"timestamp"`
	Details   string `xml:"details"`
}

type getTrackAndTraceInfoResponse struct {
	XMLName xml.Name            `xml:"Envelope"`
	Events  []soapTrackingEvent `xml:"Body>getTrackAndTraceInfoResponse>events>event"`
}

// GetTracking retrieves tracking events via SOAP getTrackAndTraceInfo.
func (s *ShipmentService) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	body := getTrackAndTraceInfoRequest{
		AuthData:       authData{Username: s.client.username, Password: s.client.password},
		TrackingNumber: trackingNumber,
	}

	raw, err := s.client.doSOAP(ctx, "getTrackAndTraceInfo", body)
	if err != nil {
		return nil, fmt.Errorf("dhl: get tracking: %w", err)
	}

	var resp getTrackAndTraceInfoResponse
	if err := xml.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dhl: parse tracking response: %w", err)
	}

	events := make([]TrackingEvent, 0, len(resp.Events))
	for _, ev := range resp.Events {
		ts, _ := time.Parse(time.RFC3339, ev.Timestamp)
		events = append(events, TrackingEvent{
			Status:    ev.Status,
			Location:  ev.Location,
			Timestamp: ts,
			Details:   ev.Details,
		})
	}

	return &TrackingResponse{
		TrackingNumber: trackingNumber,
		Events:         events,
	}, nil
}

// --- Cancel ---

type deleteShipmentRequest struct {
	XMLName    xml.Name `xml:"deleteShipment"`
	AuthData   authData `xml:"authData"`
	ShipmentID string   `xml:"shipmentId"`
}

// Cancel cancels a shipment via SOAP deleteShipment.
func (s *ShipmentService) Cancel(ctx context.Context, shipmentID string) error {
	body := deleteShipmentRequest{
		AuthData:   authData{Username: s.client.username, Password: s.client.password},
		ShipmentID: shipmentID,
	}

	_, err := s.client.doSOAP(ctx, "deleteShipment", body)
	if err != nil {
		return fmt.Errorf("dhl: cancel shipment: %w", err)
	}
	return nil
}
