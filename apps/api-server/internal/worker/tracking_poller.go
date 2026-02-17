package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type trackableShipment struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	Provider       string
	TrackingNumber string
	Status         string
	CarrierData    json.RawMessage
	Credentials    *string
	Settings       json.RawMessage
}

// ShipmentStatusTransitioner abstracts the ShipmentService.TransitionStatus method
// to avoid importing the service package from the worker package.
type ShipmentStatusTransitioner interface {
	TransitionStatus(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.ShipmentStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Shipment, error)
}

// TrackingPoller periodically polls carrier APIs for shipment tracking updates.
type TrackingPoller struct {
	pool            *pgxpool.Pool
	encryptionKey   []byte
	shipmentService ShipmentStatusTransitioner
	logger          *slog.Logger
}

func NewTrackingPoller(pool *pgxpool.Pool, encryptionKey []byte, shipmentRepo repository.ShipmentRepo, shipmentService ShipmentStatusTransitioner, logger *slog.Logger) *TrackingPoller {
	// Note: shipmentRepo is accepted for backward compatibility but status transitions
	// now go through shipmentService to ensure webhooks, audit, and order sync fire.
	return &TrackingPoller{
		pool:            pool,
		encryptionKey:   encryptionKey,
		shipmentService: shipmentService,
		logger:          logger,
	}
}

func (w *TrackingPoller) Name() string {
	return "tracking_poller"
}

func (w *TrackingPoller) Interval() time.Duration {
	return 10 * time.Minute
}

func (w *TrackingPoller) Run(ctx context.Context) error {
	w.logger.Info("tracking poller: checking shipments")

	rows, err := w.pool.Query(ctx,
		`SELECT s.id, s.tenant_id, s.provider, s.tracking_number, s.status, s.carrier_data,
		        i.credentials, i.settings
		 FROM shipments s
		 LEFT JOIN integrations i ON i.id = s.integration_id AND i.status = 'active'
		 WHERE s.tracking_number IS NOT NULL
		   AND s.status NOT IN ('delivered', 'returned', 'failed', 'cancelled')`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var shipments []trackableShipment
	for rows.Next() {
		var ts trackableShipment
		if err := rows.Scan(
			&ts.ID, &ts.TenantID, &ts.Provider, &ts.TrackingNumber,
			&ts.Status, &ts.CarrierData, &ts.Credentials, &ts.Settings,
		); err != nil {
			return err
		}
		shipments = append(shipments, ts)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(shipments) == 0 {
		w.logger.Info("tracking poller: no shipments to track")
		return nil
	}

	// Group by provider+credentials for efficiency (reuse carrier instances)
	type carrierKey struct {
		provider    string
		credentials string
	}
	groups := make(map[carrierKey][]trackableShipment)
	for _, ts := range shipments {
		key := carrierKey{provider: ts.Provider}
		if ts.Credentials != nil {
			key.credentials = *ts.Credentials
		}
		groups[key] = append(groups[key], ts)
	}

	updated := 0
	errCount := 0

	for key, group := range groups {
		if key.credentials == "" {
			w.logger.Warn("tracking poller: skipping shipments with no integration credentials",
				"provider", key.provider, "count", len(group))
			continue
		}

		// Decrypt credentials
		credJSON, err := crypto.Decrypt(key.credentials, w.encryptionKey)
		if err != nil {
			w.logger.Error("tracking poller: decrypt failed",
				"provider", key.provider, "error", err)
			errCount += len(group)
			continue
		}

		// Get settings from first shipment in group (same integration)
		settings := group[0].Settings

		carrier, err := integration.NewCarrierProvider(key.provider, credJSON, settings)
		if err != nil {
			w.logger.Error("tracking poller: create carrier provider failed",
				"provider", key.provider, "error", err)
			errCount += len(group)
			continue
		}

		for _, ts := range group {
			events, err := carrier.GetTracking(ctx, ts.TrackingNumber)
			if err != nil {
				w.logger.Error("tracking poller: get tracking failed",
					"shipment_id", ts.ID, "tracking_number", ts.TrackingNumber, "error", err)
				errCount++
				continue
			}

			if len(events) == 0 {
				continue
			}

			// Check last event for status update
			lastEvent := events[len(events)-1]
			omsStatus, ok := carrier.MapStatus(lastEvent.Status)
			if !ok || omsStatus == ts.Status {
				continue
			}

			// Transition shipment status through the service layer
			// This ensures webhooks, audit logs, and order status sync are triggered
			_, err = w.shipmentService.TransitionStatus(ctx, ts.TenantID, ts.ID,
				model.ShipmentStatusTransitionRequest{Status: omsStatus},
				uuid.Nil, "tracking_poller")
			if err != nil {
				w.logger.Error("tracking poller: transition status failed",
					"operation", "shipment.status_update",
					"tenant_id", ts.TenantID,
					"entity_id", ts.ID,
					"from", ts.Status, "to", omsStatus, "error", err)
				errCount++
				continue
			}

			w.logger.Info("tracking poller: shipment status updated",
				"shipment_id", ts.ID, "tenant_id", ts.TenantID,
				"from", ts.Status, "to", omsStatus,
				"tracking_number", ts.TrackingNumber)
			updated++
		}
	}

	w.logger.Info("tracking poller: completed",
		"total", len(shipments), "updated", updated, "errors", errCount)
	return nil
}
