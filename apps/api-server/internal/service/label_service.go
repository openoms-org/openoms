package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	inpost "github.com/openoms-org/openoms/packages/inpost-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrShipmentNotInPost   = errors.New("shipment provider is not inpost")
	ErrShipmentNotCreated  = errors.New("shipment must be in 'created' status to generate label")
	ErrNoInPostIntegration = errors.New("no active InPost integration found")
	ErrNoCustomerContact   = errors.New("order has no customer phone (required for InPost)")
)

type LabelService struct {
	shipmentRepo    *repository.ShipmentRepository
	orderRepo       *repository.OrderRepository
	integrationRepo *repository.IntegrationRepository
	auditRepo       *repository.AuditRepository
	pool            *pgxpool.Pool
	encryptionKey   []byte
	uploadDir       string
	baseURL         string
}

func NewLabelService(
	shipmentRepo *repository.ShipmentRepository,
	orderRepo *repository.OrderRepository,
	integrationRepo *repository.IntegrationRepository,
	auditRepo *repository.AuditRepository,
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
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// First transaction: load all required data from the database
	var shipment *model.Shipment
	var order *model.Order
	var apiToken string
	var orgID string

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
		if shipment.Provider != "inpost" {
			return ErrShipmentNotInPost
		}
		if shipment.Status != "created" {
			return ErrShipmentNotCreated
		}

		// Load linked order
		order, err = s.orderRepo.FindByID(ctx, tx, shipment.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFoundForShipment
		}

		// Check customer phone
		if order.CustomerPhone == nil || *order.CustomerPhone == "" {
			return ErrNoCustomerContact
		}

		// Find active InPost integration
		integration, err := s.integrationRepo.FindByProvider(ctx, tx, "inpost")
		if err != nil {
			return err
		}
		if integration == nil {
			return ErrNoInPostIntegration
		}

		// Decrypt credentials
		plaintext, err := crypto.Decrypt(integration.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypting integration credentials: %w", err)
		}

		var creds struct {
			APIToken       string `json:"api_token"`
			OrganizationID string `json:"organization_id"`
		}
		if err := json.Unmarshal(plaintext, &creds); err != nil {
			return fmt.Errorf("parsing integration credentials: %w", err)
		}

		apiToken = creds.APIToken
		orgID = creds.OrganizationID

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Outside transaction: make external API calls
	client := inpost.NewClient(apiToken, orgID)

	// Build receiver
	receiverEmail := ""
	if order.CustomerEmail != nil {
		receiverEmail = *order.CustomerEmail
	}

	inpostReq := &inpost.CreateShipmentRequest{
		Receiver: inpost.Receiver{
			Name:  order.CustomerName,
			Phone: *order.CustomerPhone,
			Email: receiverEmail,
		},
		Parcels: []inpost.Parcel{
			{
				Template: inpost.ParcelTemplate(req.ParcelSize),
				Weight: inpost.Weight{
					Amount: 1.0,
					Unit:   "kg",
				},
			},
		},
		Service:   inpost.ServiceType(req.ServiceType),
		Reference: shipment.OrderID.String(),
	}

	// Set service-specific attributes
	if req.ServiceType == "inpost_locker_standard" {
		inpostReq.CustomAttributes = &inpost.CustomAttributes{
			TargetPoint: req.TargetPoint,
		}
	} else if req.ServiceType == "inpost_courier_standard" {
		// Parse shipping address for courier delivery
		var addr inpost.Address
		if len(order.ShippingAddress) > 0 {
			if err := json.Unmarshal(order.ShippingAddress, &addr); err != nil {
				slog.Warn("failed to parse shipping address for courier", "error", err)
			}
		}
		inpostReq.Receiver.Address = &addr
	}

	// Create shipment in InPost
	inpostShipment, err := client.Shipments.Create(ctx, inpostReq)
	if err != nil {
		return nil, fmt.Errorf("inpost create shipment: %w", err)
	}

	// Map label format
	var labelFormat inpost.LabelFormat
	switch req.LabelFormat {
	case "pdf":
		labelFormat = inpost.LabelPDF
	case "zpl":
		labelFormat = inpost.LabelZPL
	case "epl":
		labelFormat = inpost.LabelEPL
	default:
		labelFormat = inpost.LabelPDF
	}

	// Get label
	labelBytes, err := client.Labels.Get(ctx, inpostShipment.ID, labelFormat)
	if err != nil {
		return nil, fmt.Errorf("inpost get label: %w", err)
	}

	// Determine file extension
	ext := req.LabelFormat

	// Save label file
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
	trackingNum := inpostShipment.TrackingNumber

	slog.Info("InPost label generated",
		"shipment_id", shipmentID,
		"inpost_shipment_id", inpostShipment.ID,
		"tracking_number", trackingNum,
	)

	// Second transaction: update shipment in database
	var updatedShipment *model.Shipment
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Build carrier_data
		carrierData := map[string]any{
			"inpost_shipment_id": inpostShipment.ID,
			"service":            req.ServiceType,
			"parcel_size":        req.ParcelSize,
			"tracking_number":    trackingNum,
		}
		carrierDataJSON, err := json.Marshal(carrierData)
		if err != nil {
			return fmt.Errorf("marshaling carrier data: %w", err)
		}

		// Update shipment fields
		updateReq := model.UpdateShipmentRequest{
			TrackingNumber: &trackingNum,
			LabelURL:       &labelURL,
			CarrierData:    carrierDataJSON,
		}
		if err := s.shipmentRepo.Update(ctx, tx, shipmentID, updateReq); err != nil {
			return err
		}

		// Update status to label_ready
		if err := s.shipmentRepo.UpdateStatus(ctx, tx, shipmentID, "label_ready"); err != nil {
			return err
		}

		// Audit log
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

		// Re-fetch updated shipment
		updatedShipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedShipment, nil
}
