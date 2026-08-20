package allegro

import (
	"context"
	"encoding/base64"
	"strings"
)

// ExistingWzAShipment is a WzA shipment already created in Allegro (Sales Center
// or a previous API create) and discovered by read-only GETs.
type ExistingWzAShipment struct {
	ShipmentID       string
	Waybill          string
	CarrierID        string
	Carrier          string
	DeliveryMethodID string
}

// PackageWaybill returns the first non-empty waybill from the official payload.
func (s ManagedShipment) PackageWaybill() string {
	if w := strings.TrimSpace(s.Waybill); w != "" {
		return w
	}
	for _, pkg := range s.Packages {
		if w := strings.TrimSpace(pkg.Waybill); w != "" {
			return w
		}
		for _, info := range pkg.TransportingInfo {
			if w := strings.TrimSpace(info.CarrierWaybill); w != "" {
				return w
			}
		}
	}
	return ""
}

// FindExistingWzA lists tracking numbers already attached to a checkout-form
// and hydrates WzA details when the checkout shipment id is a WzA UUID.
// It never POSTs create-commands, create-shipment, or AddTracking.
func (c *Client) FindExistingWzA(ctx context.Context, checkoutFormID string) ([]ExistingWzAShipment, error) {
	checkoutFormID = strings.TrimSpace(checkoutFormID)
	if checkoutFormID == "" {
		return nil, ErrWzANoExistingShipment
	}
	trackings, err := c.Fulfillment.ListShipments(ctx, checkoutFormID)
	if err != nil {
		return nil, err
	}
	return resolveExistingWzA(ctx, trackings, c.ShipmentManagement.GetShipment)
}

func resolveExistingWzA(
	ctx context.Context,
	trackings []OrderShipment,
	getShipment func(context.Context, string) (*ManagedShipment, error),
) ([]ExistingWzAShipment, error) {
	var found []ExistingWzAShipment
	for _, tr := range trackings {
		waybill := firstNonEmpty(strings.TrimSpace(tr.Waybill), waybillFromTrackingAssignmentID(tr.ID))
		item := ExistingWzAShipment{
			Waybill:   waybill,
			CarrierID: strings.TrimSpace(tr.CarrierID),
		}
		if looksLikeWzAShipmentID(tr.ID) {
			item.ShipmentID = strings.TrimSpace(tr.ID)
			if getShipment != nil {
				sh, err := getShipment(ctx, item.ShipmentID)
				if err == nil && sh != nil {
					if w := sh.PackageWaybill(); w != "" {
						item.Waybill = w
					}
					item.ShipmentID = firstNonEmpty(strings.TrimSpace(sh.ID), item.ShipmentID)
					item.DeliveryMethodID = strings.TrimSpace(sh.DeliveryMethodID)
					item.Carrier = firstNonEmpty(strings.TrimSpace(sh.Carrier), strings.TrimSpace(sh.CarrierID))
				}
			}
		}
		if item.Waybill == "" {
			continue
		}
		found = append(found, item)
	}
	if len(found) == 0 {
		return nil, ErrWzANoExistingShipment
	}
	return found, nil
}

// waybillFromTrackingAssignmentID decodes the official checkout-forms shipment
// id (base64 of "CARRIER:waybill"). A WzA UUID must not be treated as a waybill.
func waybillFromTrackingAssignmentID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || looksLikeWzAShipmentID(id) {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(id)
		if err != nil {
			return ""
		}
	}
	decoded := strings.TrimSpace(string(raw))
	carrier, waybill, ok := strings.Cut(decoded, ":")
	if !ok || strings.TrimSpace(carrier) == "" {
		return ""
	}
	return strings.TrimSpace(waybill)
}

func looksLikeWzAShipmentID(id string) bool {
	id = strings.TrimSpace(id)
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !isHexRune(c) {
				return false
			}
		}
	}
	return true
}

func isHexRune(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
