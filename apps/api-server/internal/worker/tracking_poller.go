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

const (
	// trackingPollerPageSize is the keyset page size for fetching trackable shipments.
	trackingPollerPageSize = 100
	// trackingPollerMaxShipments caps how many shipments a single run processes so a
	// huge backlog can't run unbounded; the remainder is picked up on the next run.
	trackingPollerMaxShipments = 5000
)

type trackableShipment struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	OrderID        uuid.UUID
	Provider       string
	TrackingNumber string
	Status         string
	CarrierData    json.RawMessage
	UpdatedAt      time.Time
	Credentials    *string
	Settings       json.RawMessage
}

// ShipmentStatusTransitioner abstracts the ShipmentService.TransitionStatus method
// to avoid importing the service package from the worker package.
type ShipmentStatusTransitioner interface {
	TransitionStatus(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.ShipmentStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Shipment, error)
}

// TrackingFulfillmentRecorder is the narrow surface the poller uses to record
// tracking-sync provider attempts, emit fulfillment steps and create blockers
// without importing the service package (which would be a cycle). It is satisfied
// by *service.FulfillmentService (OPE-417). All implementations are gated +
// best-effort: nil/disabled implementations make the calls no-ops.
type TrackingFulfillmentRecorder interface {
	Enabled() bool
	RecordTrackingSyncAttempt(ctx context.Context, tenantID, orderID uuid.UUID, provider, correlationID string, mapping model.TrackingStatusMapping, status string)
	RecordTrackingSyncFailure(ctx context.Context, tenantID, orderID uuid.UUID, provider, correlationID string, cause error)
}

// TrackingPoller periodically polls carrier APIs for shipment tracking updates.
type TrackingPoller struct {
	pool            *pgxpool.Pool
	encryptionKey   []byte
	shipmentService ShipmentStatusTransitioner
	fulfillment     TrackingFulfillmentRecorder
	logger          *slog.Logger
	// fetchPage loads one keyset page; defaults to queryTrackablePage and is
	// overridable in tests to exercise the pagination loop without a database.
	fetchPage func(ctx context.Context, cursorTime time.Time, cursorID uuid.UUID) ([]trackableShipment, error)
}

// SetFulfillmentRecorder wires the gated fulfillment recorder used for best-effort
// tracking-sync provider-attempt recording (OPE-417). Nil-safe and a complete
// no-op until FULFILLMENT_PROCESS_ENABLED is set.
func (w *TrackingPoller) SetFulfillmentRecorder(f TrackingFulfillmentRecorder) {
	w.fulfillment = f
}

// fulfillmentEnabled reports whether the gated recorder is wired and enabled.
func (w *TrackingPoller) fulfillmentEnabled() bool {
	return w.fulfillment != nil && w.fulfillment.Enabled()
}

// NewTrackingPoller creates a new TrackingPoller worker.
func NewTrackingPoller(pool *pgxpool.Pool, encryptionKey []byte, _ repository.ShipmentRepo, shipmentService ShipmentStatusTransitioner, logger *slog.Logger) *TrackingPoller {
	// Note: shipmentRepo is accepted for backward compatibility but status transitions
	// now go through shipmentService to ensure webhooks, audit, and order sync fire.
	w := &TrackingPoller{
		pool:            pool,
		encryptionKey:   encryptionKey,
		shipmentService: shipmentService,
		logger:          logger,
	}
	w.fetchPage = w.queryTrackablePage
	return w
}

// Name returns the worker identifier.
func (w *TrackingPoller) Name() string {
	return "tracking_poller"
}

// Interval returns how frequently the worker should run.
func (w *TrackingPoller) Interval() time.Duration {
	return 10 * time.Minute
}

func trackableShipmentsQuery() string {
	return `SELECT s.id, s.tenant_id, s.order_id, s.provider, s.tracking_number, s.status, s.carrier_data, s.updated_at,
	        i.credentials, i.settings
	 FROM shipments s
	 JOIN integrations i ON i.id = s.integration_id
	   AND i.status = 'active'
	   AND i.credentials IS NOT NULL
	   AND i.credentials <> '""'::jsonb
	   AND i.credentials <> '{}'::jsonb
	   AND i.credentials <> 'null'::jsonb
	 WHERE s.tracking_number IS NOT NULL
	   AND s.tracking_number <> ''
	   AND s.status NOT IN ('delivered', 'returned', 'failed', 'cancelled')
	   AND (s.updated_at, s.id) > ($2, $3)
	 ORDER BY s.updated_at ASC, s.id ASC
	 LIMIT $1`
}

// collectTrackableShipments loads trackable shipments using keyset pagination on
// (updated_at, id). Previously a single LIMIT 100 query meant tenants with more
// than one page of active shipments were permanently capped at the oldest 100 and
// lagged on tracking updates (OPE-487). A per-run cap bounds the work; the rest is
// picked up on the next run.
func (w *TrackingPoller) collectTrackableShipments(ctx context.Context) ([]trackableShipment, error) {
	var shipments []trackableShipment
	var cursorTime time.Time
	var cursorID uuid.UUID

	for {
		page, err := w.fetchPage(ctx, cursorTime, cursorID)
		if err != nil {
			return nil, err
		}
		shipments = append(shipments, page...)
		if len(page) < trackingPollerPageSize {
			break
		}
		if len(shipments) >= trackingPollerMaxShipments {
			w.logger.Warn("tracking poller: hit per-run shipment cap; remainder will be processed next run",
				"cap", trackingPollerMaxShipments)
			break
		}
		last := page[len(page)-1]
		cursorTime, cursorID = last.UpdatedAt, last.ID
	}
	return shipments, nil
}

func (w *TrackingPoller) queryTrackablePage(ctx context.Context, cursorTime time.Time, cursorID uuid.UUID) ([]trackableShipment, error) {
	rows, err := w.pool.Query(ctx, trackableShipmentsQuery(), trackingPollerPageSize, cursorTime, cursorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var page []trackableShipment
	for rows.Next() {
		if err := checkWorkerContext(ctx); err != nil {
			return nil, err
		}
		var ts trackableShipment
		if err := rows.Scan(
			&ts.ID, &ts.TenantID, &ts.OrderID, &ts.Provider, &ts.TrackingNumber,
			&ts.Status, &ts.CarrierData, &ts.UpdatedAt, &ts.Credentials, &ts.Settings,
		); err != nil {
			return nil, err
		}
		page = append(page, ts)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return page, nil
}

// Run polls carrier APIs for tracking status updates on active shipments.
func (w *TrackingPoller) Run(ctx context.Context) error {
	w.logger.Info("tracking poller: checking shipments")

	shipments, err := w.collectTrackableShipments(ctx)
	if err != nil {
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
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		key := carrierKey{provider: ts.Provider}
		if ts.Credentials != nil {
			key.credentials = *ts.Credentials
		}
		groups[key] = append(groups[key], ts)
	}

	updated := 0
	errCount := 0

	for key, group := range groups {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
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
			if err := checkWorkerContext(ctx); err != nil {
				closeProvider(carrier)
				return err
			}
			events, err := carrier.GetTracking(ctx, ts.TrackingNumber)
			if err != nil {
				w.logger.Error("tracking poller: get tracking failed",
					"shipment_id", ts.ID, "tracking_number", ts.TrackingNumber, "error", err)
				// OPE-417: record the failed sync_tracking attempt + typed blocker
				// (gated, best-effort — never changes polling control flow).
				if w.fulfillmentEnabled() {
					w.fulfillment.RecordTrackingSyncFailure(ctx, ts.TenantID, ts.OrderID, ts.Provider, ts.ID.String(), err)
				}
				errCount++
				continue
			}

			if len(events) == 0 {
				continue
			}

			// Check last event for status update
			lastEvent := events[len(events)-1]
			omsStatus, ok := carrier.MapStatus(lastEvent.Status)
			// OPE-417: record the sync_tracking attempt, ALWAYS preserving the raw
			// provider status. An unmapped status (ok=false) is an explicit
			// unsupported-capability outcome — recorded, not treated as a failure.
			if w.fulfillmentEnabled() {
				mapping := model.NewTrackingStatusMapping(lastEvent.Status, omsStatus, ok)
				w.fulfillment.RecordTrackingSyncAttempt(ctx, ts.TenantID, ts.OrderID, ts.Provider, ts.ID.String(),
					mapping, model.ProviderAttemptSucceeded)
			}
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
		closeProvider(carrier)
	}

	w.logger.Info("tracking poller: completed",
		"total", len(shipments), "updated", updated, "errors", errCount)
	return nil
}
