package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
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

// TrackingPoller periodically polls carrier APIs for shipment tracking updates.
type TrackingPoller struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	shipmentRepo  repository.ShipmentRepo
	logger        *slog.Logger
}

func NewTrackingPoller(pool *pgxpool.Pool, encryptionKey []byte, shipmentRepo repository.ShipmentRepo, logger *slog.Logger) *TrackingPoller {
	return &TrackingPoller{
		pool:          pool,
		encryptionKey: encryptionKey,
		shipmentRepo:  shipmentRepo,
		logger:        logger,
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

			// Update shipment status within tenant context
			err = database.WithTenant(ctx, w.pool, ts.TenantID, func(tx pgx.Tx) error {
				return w.shipmentRepo.UpdateStatus(ctx, tx, ts.ID, omsStatus)
			})
			if err != nil {
				w.logger.Error("tracking poller: update status failed",
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
