package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrShipmentNotCreated    = errors.New("shipment must be in 'created' status to generate label")
	ErrNoCarrierIntegration  = errors.New("no active carrier integration found for provider")
	ErrNoCustomerContact     = errors.New("order has no customer email or phone")
)

type LabelService struct {
	shipmentRepo    repository.ShipmentRepo
	orderRepo       repository.OrderRepo
	integrationRepo repository.IntegrationRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	encryptionKey   []byte
	uploadDir       string
	baseURL         string
}

func NewLabelService(
	shipmentRepo repository.ShipmentRepo,
	orderRepo repository.OrderRepo,
	integrationRepo repository.IntegrationRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	encryptionKey []byte,
	uploadDir string,
	baseURL string,
) *LabelService {
	return &LabelService{
		shipmentRepo:    shipmentRepo,
		orderRepo:       orderRepo,
		integrationRepo: integrationRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		encryptionKey:   encryptionKey,
		uploadDir:       uploadDir,
		baseURL:         baseURL,
	}
}

func (s *LabelService) GenerateLabel(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.GenerateLabelRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	// First transaction: load all required data from the database
	var shipment *model.Shipment
	var order *model.Order
	var credJSON []byte
	var integrationSettings json.RawMessage

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error

		// Load shipment
		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if shipment == nil {
			return ErrShipmentNotFound
		}
		if shipment.Status != "created" {
			return ErrShipmentNotCreated
		}

		// Merge carrier_data from shipment into request (fill missing fields)
		if len(shipment.CarrierData) > 0 {
			var cd map[string]interface{}
			if err := json.Unmarshal(shipment.CarrierData, &cd); err == nil {
				if req.TargetPoint == "" {
					if tp, ok := cd["target_point"].(string); ok && tp != "" {
						req.TargetPoint = tp
					}
				}
				if req.ServiceType == "" {
					if st, ok := cd["service_type"].(string); ok && st != "" {
						req.ServiceType = st
					}
				}
				if req.ParcelSize == "" {
					if ps, ok := cd["parcel_size"].(string); ok && ps != "" {
						req.ParcelSize = ps
					}
				}
				if req.SendingMethod == "" {
					if sm, ok := cd["sending_method"].(string); ok && sm != "" {
						req.SendingMethod = sm
					}
				}
			}
		}

		// Validate after merging carrier_data
		if err := req.Validate(); err != nil {
			return NewValidationError(err)
		}

		// Load linked order
		order, err = s.orderRepo.FindByID(ctx, tx, shipment.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFoundForShipment
		}

		// Check customer contact info
		hasPhone := order.CustomerPhone != nil && *order.CustomerPhone != ""
		hasEmail := order.CustomerEmail != nil && *order.CustomerEmail != ""
		if !hasPhone && !hasEmail {
			return ErrNoCustomerContact
		}
		// InPost requires phone number specifically
		if shipment.Provider == "inpost" && !hasPhone {
			return NewValidationError(fmt.Errorf("InPost wymaga numeru telefonu odbiorcy — uzupełnij telefon w zamówieniu"))
		}

		// Find active integration for this carrier
		integrationData, err := s.integrationRepo.FindByProvider(ctx, tx, shipment.Provider)
		if err != nil {
			return err
		}
		if integrationData == nil {
			return ErrNoCarrierIntegration
		}

		integrationSettings = integrationData.Settings

		// Fall back to integration-level default sending method
		if req.SendingMethod == "" && len(integrationSettings) > 0 {
			var settingsMap map[string]interface{}
			if err := json.Unmarshal(integrationSettings, &settingsMap); err == nil {
				if sm, ok := settingsMap["default_sending_method"].(string); ok && sm != "" {
					req.SendingMethod = sm
				}
			}
		}

		// Decrypt credentials
		credJSON, err = crypto.Decrypt(integrationData.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypting integration credentials: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Outside transaction: use carrier abstraction
	carrier, err := integration.NewCarrierProvider(shipment.Provider, credJSON, integrationSettings)
	if err != nil {
		return nil, fmt.Errorf("creating carrier provider: %w", err)
	}

	// Parse shipping address
	var addr model.ShippingAddress
	if len(order.ShippingAddress) > 0 {
		if err := json.Unmarshal(order.ShippingAddress, &addr); err != nil {
			slog.Warn("failed to parse shipping address", "error", err)
		}
	}

	customerEmail := ""
	if order.CustomerEmail != nil {
		customerEmail = *order.CustomerEmail
	}
	customerPhone := ""
	if order.CustomerPhone != nil {
		customerPhone = *order.CustomerPhone
	}

	carrierReq := integration.CarrierShipmentRequest{
		OrderID:     shipment.OrderID.String(),
		ServiceType: req.ServiceType,
		Receiver: integration.CarrierReceiver{
			Name:       order.CustomerName,
			Email:      customerEmail,
			Phone:      customerPhone,
			Street:     addr.Street,
			City:       addr.City,
			PostalCode: addr.PostalCode,
			Country:    addr.Country,
		},
		Parcel: integration.CarrierParcel{
			SizeCode: req.ParcelSize,
			WeightKg: req.WeightKg,
			WidthCm:  req.WidthCm,
			HeightCm: req.HeightCm,
			DepthCm:  req.DepthCm,
		},
		TargetPoint:   req.TargetPoint,
		SendingMethod: req.SendingMethod,
		CODAmount:     req.CODAmount,
		InsuredValue:  req.InsuredValue,
		Reference:     shipment.OrderID.String(),
	}

	resp, err := carrier.CreateShipment(ctx, carrierReq)
	if err != nil {
		return nil, fmt.Errorf("carrier create shipment: %w", err)
	}

	// Get label (some carriers may return label URL in CreateShipment, but we always
	// fetch via GetLabel for a consistent local-file approach)
	labelBytes, err := carrier.GetLabel(ctx, resp.ExternalID, req.LabelFormat)
	if err != nil {
		return nil, fmt.Errorf("carrier get label: %w", err)
	}

	// Save label file
	ext := req.LabelFormat
	labelDir := filepath.Join(s.uploadDir, tenantID.String())
	if err := os.MkdirAll(labelDir, 0755); err != nil {
		return nil, fmt.Errorf("creating label directory: %w", err)
	}

	filename := uuid.New().String() + "." + ext
	labelPath := filepath.Join(labelDir, filename)
	if err := os.WriteFile(labelPath, labelBytes, 0644); err != nil {
		return nil, fmt.Errorf("saving label file: %w", err)
	}

	labelURL := fmt.Sprintf("%s/uploads/%s/%s", s.baseURL, tenantID.String(), filename)
	trackingNum := resp.TrackingNumber

	slog.Info("carrier label generated",
		"shipment_id", shipmentID,
		"provider", shipment.Provider,
		"external_id", resp.ExternalID,
		"tracking_number", trackingNum,
	)

	// Second transaction: update shipment in database
	var updatedShipment *model.Shipment
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		carrierData := map[string]any{
			"external_id":     resp.ExternalID,
			"service":         req.ServiceType,
			"tracking_number": trackingNum,
		}
		if req.ParcelSize != "" {
			carrierData["parcel_size"] = req.ParcelSize
		}
		if req.SendingMethod != "" {
			carrierData["sending_method"] = req.SendingMethod
		}
		carrierDataJSON, err := json.Marshal(carrierData)
		if err != nil {
			return fmt.Errorf("marshaling carrier data: %w", err)
		}

		updateReq := model.UpdateShipmentRequest{
			TrackingNumber: &trackingNum,
			LabelURL:       &labelURL,
			CarrierData:    carrierDataJSON,
		}
		if err := s.shipmentRepo.Update(ctx, tx, shipmentID, updateReq); err != nil {
			return err
		}

		if err := s.shipmentRepo.UpdateStatus(ctx, tx, shipmentID, "label_ready"); err != nil {
			return err
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.label_generated",
			EntityType: "shipment",
			EntityID:   shipmentID,
			Changes:    map[string]string{"tracking_number": trackingNum, "label_url": labelURL},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		updatedShipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedShipment, nil
}

// GetTracking fetches real-time tracking events from the carrier API.
func (s *LabelService) GetTracking(ctx context.Context, tenantID, shipmentID uuid.UUID) ([]integration.TrackingEvent, error) {
	var shipment *model.Shipment
	var credJSON []byte
	var integrationSettings json.RawMessage

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if shipment == nil {
			return ErrShipmentNotFound
		}
		if shipment.TrackingNumber == nil || *shipment.TrackingNumber == "" {
			return nil // no tracking number yet
		}

		integrationData, err := s.integrationRepo.FindByProvider(ctx, tx, shipment.Provider)
		if err != nil {
			return err
		}
		if integrationData == nil {
			return ErrNoCarrierIntegration
		}
		integrationSettings = integrationData.Settings

		credJSON, err = crypto.Decrypt(integrationData.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypting integration credentials: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if shipment.TrackingNumber == nil || *shipment.TrackingNumber == "" {
		return []integration.TrackingEvent{}, nil
	}

	carrier, err := integration.NewCarrierProvider(shipment.Provider, credJSON, integrationSettings)
	if err != nil {
		return nil, fmt.Errorf("creating carrier provider: %w", err)
	}

	events, err := carrier.GetTracking(ctx, *shipment.TrackingNumber)
	if err != nil {
		return nil, fmt.Errorf("carrier get tracking: %w", err)
	}

	if events == nil {
		events = []integration.TrackingEvent{}
	}
	return events, nil
}

// CreateDispatchOrder creates a dispatch order (courier pickup) for the given shipments.
func (s *LabelService) CreateDispatchOrder(ctx context.Context, tenantID uuid.UUID, req model.CreateDispatchOrderRequest, actorID uuid.UUID, ip string) (*model.DispatchOrderResponse, error) {
	var shipments []*model.Shipment
	var credJSON []byte
	var integrationSettings json.RawMessage
	var provider string

	// First transaction: load and validate all shipments, load integration
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for _, sid := range req.ShipmentIDs {
			shipment, err := s.shipmentRepo.FindByID(ctx, tx, sid)
			if err != nil {
				return err
			}
			if shipment == nil {
				return ErrShipmentNotFound
			}
			if shipment.Status != "label_ready" && shipment.Status != "confirmed" {
				return NewValidationError(fmt.Errorf("shipment %s must be in 'label_ready' or 'confirmed' status (current: %s)", sid, shipment.Status))
			}
			if provider == "" {
				provider = shipment.Provider
			} else if shipment.Provider != provider {
				return NewValidationError(fmt.Errorf("all shipments must use the same carrier provider"))
			}
			shipments = append(shipments, shipment)
		}

		integrationData, err := s.integrationRepo.FindByProvider(ctx, tx, provider)
		if err != nil {
			return err
		}
		if integrationData == nil {
			return ErrNoCarrierIntegration
		}
		integrationSettings = integrationData.Settings

		credJSON, err = crypto.Decrypt(integrationData.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypting integration credentials: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Create carrier provider and assert DispatchOrderCreator capability
	carrier, err := integration.NewCarrierProvider(provider, credJSON, integrationSettings)
	if err != nil {
		return nil, fmt.Errorf("creating carrier provider: %w", err)
	}

	dispatchCreator, ok := carrier.(integration.DispatchOrderCreator)
	if !ok {
		return nil, NewValidationError(fmt.Errorf("carrier %q does not support dispatch orders", provider))
	}

	// Extract external IDs from carrier_data
	var externalIDs []int64
	for _, shipment := range shipments {
		var cd map[string]interface{}
		if err := json.Unmarshal(shipment.CarrierData, &cd); err != nil {
			return nil, fmt.Errorf("parsing carrier_data for shipment %s: %w", shipment.ID, err)
		}
		extIDStr, ok := cd["external_id"].(string)
		if !ok || extIDStr == "" {
			return nil, NewValidationError(fmt.Errorf("shipment %s has no external_id in carrier_data", shipment.ID))
		}
		extID, err := strconv.ParseInt(extIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing external_id for shipment %s: %w", shipment.ID, err)
		}
		externalIDs = append(externalIDs, extID)
	}

	// Build address and contact
	address := integration.DispatchOrderAddress{
		Street:         req.Street,
		BuildingNumber: req.BuildingNo,
		City:           req.City,
		PostCode:       req.PostCode,
		CountryCode:    "PL",
	}
	contact := integration.DispatchOrderContact{
		Name:    req.Name,
		Phone:   req.Phone,
		Email:   req.Email,
		Comment: req.Comment,
	}

	// Call carrier API
	orderID, err := dispatchCreator.CreateDispatchOrder(ctx, externalIDs, address, contact)
	if err != nil {
		return nil, fmt.Errorf("carrier create dispatch order: %w", err)
	}

	slog.Info("dispatch order created",
		"dispatch_order_id", orderID,
		"provider", provider,
		"shipment_count", len(shipments),
	)

	// Second transaction: save dispatch_order_id in each shipment's carrier_data and audit log
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for _, shipment := range shipments {
			var cd map[string]interface{}
			if err := json.Unmarshal(shipment.CarrierData, &cd); err != nil {
				cd = map[string]interface{}{}
			}
			cd["dispatch_order_id"] = orderID

			updatedCD, err := json.Marshal(cd)
			if err != nil {
				return fmt.Errorf("marshaling updated carrier_data: %w", err)
			}

			updateReq := model.UpdateShipmentRequest{
				CarrierData: updatedCD,
			}
			if err := s.shipmentRepo.Update(ctx, tx, shipment.ID, updateReq); err != nil {
				return fmt.Errorf("updating shipment %s carrier_data: %w", shipment.ID, err)
			}
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.dispatch_order_created",
			EntityType: "shipment",
			EntityID:   shipments[0].ID,
			Changes:    map[string]string{"dispatch_order_id": strconv.FormatInt(orderID, 10), "shipment_count": strconv.Itoa(len(shipments))},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &model.DispatchOrderResponse{
		ID:     orderID,
		Status: "created",
	}, nil
}
