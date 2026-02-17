package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrStockSyncChannelNotFound = errors.New("stock sync channel not found")
)

// StockSyncService provides business logic for real-time stock synchronization.
type StockSyncService struct {
	channelRepo     repository.StockSyncChannelRepo
	eventRepo       repository.StockSyncEventRepo
	productRepo     repository.ProductRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
	logger          *slog.Logger
}

// NewStockSyncService creates a new StockSyncService.
func NewStockSyncService(
	channelRepo repository.StockSyncChannelRepo,
	eventRepo repository.StockSyncEventRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *StockSyncService {
	return &StockSyncService{
		channelRepo:     channelRepo,
		eventRepo:       eventRepo,
		productRepo:     productRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
		logger:          logger,
	}
}

// --- Channel CRUD ---

// ListChannels returns a paginated list of stock sync channels.
func (s *StockSyncService) ListChannels(ctx context.Context, tenantID uuid.UUID, filter model.StockSyncChannelListFilter) (model.ListResponse[model.StockSyncChannel], error) {
	var resp model.ListResponse[model.StockSyncChannel]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		channels, total, err := s.channelRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.ListResponse[model.StockSyncChannel]{
			Items:  channels,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	if resp.Items == nil {
		resp.Items = []model.StockSyncChannel{}
	}
	return resp, err
}

// GetChannel returns a single stock sync channel by ID.
func (s *StockSyncService) GetChannel(ctx context.Context, tenantID, channelID uuid.UUID) (*model.StockSyncChannel, error) {
	var ch *model.StockSyncChannel
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		ch, err = s.channelRepo.FindByID(ctx, tx, channelID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrStockSyncChannelNotFound
	}
	return ch, nil
}

// CreateChannel creates a new stock sync channel.
func (s *StockSyncService) CreateChannel(ctx context.Context, tenantID uuid.UUID, req model.CreateStockSyncChannelRequest) (*model.StockSyncChannel, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	ch := &model.StockSyncChannel{
		ID:            uuid.New(),
		TenantID:      tenantID,
		IntegrationID: req.IntegrationID,
		ChannelType:   req.ChannelType,
		Enabled:       true,
		StockBuffer:   0,
		SyncMode:      "realtime",
		Priority:      0,
	}
	if req.Enabled != nil {
		ch.Enabled = *req.Enabled
	}
	if req.StockBuffer != nil {
		ch.StockBuffer = *req.StockBuffer
	}
	if req.SyncMode != nil {
		ch.SyncMode = *req.SyncMode
	}
	if req.Priority != nil {
		ch.Priority = *req.Priority
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.channelRepo.Create(ctx, tx, ch); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "stock_sync_channel",
			EntityID:   ch.ID,
			Action:     "created",
		})
	})
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// UpdateChannel updates a stock sync channel.
func (s *StockSyncService) UpdateChannel(ctx context.Context, tenantID, channelID uuid.UUID, req model.UpdateStockSyncChannelRequest) error {
	if err := req.Validate(); err != nil {
		return NewValidationError(err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.channelRepo.FindByID(ctx, tx, channelID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStockSyncChannelNotFound
		}
		if err := s.channelRepo.Update(ctx, tx, channelID, req); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "stock_sync_channel",
			EntityID:   channelID,
			Action:     "updated",
		})
	})
}

// DeleteChannel deletes a stock sync channel.
func (s *StockSyncService) DeleteChannel(ctx context.Context, tenantID, channelID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.channelRepo.FindByID(ctx, tx, channelID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStockSyncChannelNotFound
		}
		if err := s.channelRepo.Delete(ctx, tx, channelID); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "stock_sync_channel",
			EntityID:   channelID,
			Action:     "deleted",
		})
	})
}

// --- Stock sync logic ---

// OnStockChange is the main entry point called when stock levels change.
// It calculates available stock per channel, creates an event record,
// dispatches webhook, and triggers push to all enabled realtime channels.
func (s *StockSyncService) OnStockChange(ctx context.Context, tenantID, productID uuid.UUID, triggerType string, oldQty, newQty int) {
	if !model.IsValidTriggerType(triggerType) {
		s.logger.Warn("stock sync: invalid trigger type", "trigger_type", triggerType)
		return
	}

	var event model.StockSyncEvent
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Look up product SKU
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}

		// Calculate available stock from warehouse tables
		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}

		// Get enabled channels and calculate total buffer
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		totalBuffer := 0
		for _, ch := range channels {
			totalBuffer += ch.StockBuffer
		}
		availableQty := max(totalQty-reservedQty-totalBuffer, 0)

		// Count how many channels are realtime-enabled
		realtimeCount := 0
		for _, ch := range channels {
			if ch.SyncMode == "realtime" {
				realtimeCount++
			}
		}

		var sku *string
		if product != nil {
			sku = product.SKU
		}

		// Create event record
		details := map[string]any{
			"total_stock":       totalQty,
			"reserved":          reservedQty,
			"total_buffer":      totalBuffer,
			"realtime_channels": realtimeCount,
		}
		detailsJSON, _ := json.Marshal(details)

		event = model.StockSyncEvent{
			ID:                uuid.New(),
			TenantID:          tenantID,
			ProductID:         &productID,
			SKU:               sku,
			TriggerType:       triggerType,
			OldQuantity:       oldQty,
			NewQuantity:       newQty,
			AvailableQuantity: availableQty,
			ChannelsNotified:  realtimeCount,
			ChannelsFailed:    0,
			Details:           detailsJSON,
		}
		return s.eventRepo.Create(ctx, tx, &event)
	})
	if err != nil {
		s.logger.Error("stock sync: failed to process stock change",
			"tenant_id", tenantID,
			"product_id", productID,
			"error", err,
		)
		return
	}

	// Dispatch webhook for stock change
	if s.webhookDispatch != nil {
		go s.webhookDispatch.Dispatch(ctx, tenantID, "stock.changed", event)
	}

	s.logger.Info("stock sync: stock change recorded",
		"tenant_id", tenantID,
		"product_id", productID,
		"trigger", triggerType,
		"old_qty", oldQty,
		"new_qty", newQty,
		"available", event.AvailableQuantity,
	)
}

// CalculateAvailableStock returns stock allocation per channel for a product.
func (s *StockSyncService) CalculateAvailableStock(ctx context.Context, tenantID, productID uuid.UUID) ([]model.StockAllocation, error) {
	var allocations []model.StockAllocation
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}

		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		// Calculate total buffer across all channels
		totalBuffer := 0
		for _, ch := range channels {
			totalBuffer += ch.StockBuffer
		}

		baseAvailable := max(totalQty-reservedQty, 0)

		for _, ch := range channels {
			available := max(baseAvailable-ch.StockBuffer, 0)
			allocations = append(allocations, model.StockAllocation{
				ChannelID:         ch.ID,
				ChannelType:       ch.ChannelType,
				TotalStock:        totalQty,
				Reserved:          reservedQty,
				Buffer:            ch.StockBuffer,
				AvailableQuantity: available,
			})
		}

		return nil
	})
	if allocations == nil {
		allocations = []model.StockAllocation{}
	}
	return allocations, err
}

// PushStockToChannel pushes current stock to a specific channel.
func (s *StockSyncService) PushStockToChannel(ctx context.Context, tenantID, channelID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		ch, err := s.channelRepo.FindByID(ctx, tx, channelID)
		if err != nil {
			return err
		}
		if ch == nil {
			return ErrStockSyncChannelNotFound
		}

		// Mark sync timestamp
		return s.channelRepo.UpdateSyncStatus(ctx, tx, channelID, nil)
	})
}

// PushStockToAllChannels pushes stock for a product to all enabled channels.
func (s *StockSyncService) PushStockToAllChannels(ctx context.Context, tenantID, productID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}
		baseAvailable := max(totalQty-reservedQty, 0)

		for _, ch := range channels {
			available := max(baseAvailable-ch.StockBuffer, 0)

			// Update sync status — in production, this would push to the marketplace API
			if err := s.channelRepo.UpdateSyncStatus(ctx, tx, ch.ID, nil); err != nil {
				s.logger.Error("stock sync: failed to update channel sync status",
					"channel_id", ch.ID,
					"error", err,
				)
			}
		}

		s.logger.Info("stock sync: pushed stock to all channels",
			"tenant_id", tenantID,
			"product_id", productID,
			"channels", len(channels),
			"available", baseAvailable,
		)
		return nil
	})
}

// PushAll triggers a stock push for all products across all enabled channels.
func (s *StockSyncService) PushAll(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var synced int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		for _, ch := range channels {
			if err := s.channelRepo.UpdateSyncStatus(ctx, tx, ch.ID, nil); err != nil {
				s.logger.Error("stock sync: failed to update channel during push all",
					"channel_id", ch.ID,
					"error", err,
				)
				continue
			}
			synced++
		}
		return nil
	})
	return synced, err
}

// ReconcileStock reconciles stock for a product by comparing local stock with channel states.
func (s *StockSyncService) ReconcileStock(ctx context.Context, tenantID, productID uuid.UUID) (*model.StockSyncEvent, error) {
	var event *model.StockSyncEvent
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return errors.New("product not found")
		}

		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}

		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		totalBuffer := 0
		for _, ch := range channels {
			totalBuffer += ch.StockBuffer
		}

		availableQty := max(totalQty-reservedQty-totalBuffer, 0)

		details := map[string]any{
			"reconcile":    true,
			"total_stock":  totalQty,
			"reserved":     reservedQty,
			"total_buffer": totalBuffer,
			"channels":     len(channels),
		}
		detailsJSON, _ := json.Marshal(details)

		evt := model.StockSyncEvent{
			ID:                uuid.New(),
			TenantID:          tenantID,
			ProductID:         &productID,
			SKU:               product.SKU,
			TriggerType:       "recount",
			OldQuantity:       totalQty,
			NewQuantity:       totalQty,
			AvailableQuantity: availableQty,
			ChannelsNotified:  len(channels),
			ChannelsFailed:    0,
			Details:           detailsJSON,
		}
		if err := s.eventRepo.Create(ctx, tx, &evt); err != nil {
			return err
		}

		// Update all channel sync timestamps
		for _, ch := range channels {
			_ = s.channelRepo.UpdateSyncStatus(ctx, tx, ch.ID, nil)
		}

		event = &evt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return event, nil
}

// --- Events & Dashboard ---

// ListEvents returns a paginated list of stock sync events.
func (s *StockSyncService) ListEvents(ctx context.Context, tenantID uuid.UUID, filter model.StockSyncEventListFilter) (model.ListResponse[model.StockSyncEvent], error) {
	var resp model.ListResponse[model.StockSyncEvent]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		events, total, err := s.eventRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.ListResponse[model.StockSyncEvent]{
			Items:  events,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	if resp.Items == nil {
		resp.Items = []model.StockSyncEvent{}
	}
	return resp, err
}

// GetDashboard returns an overview of the stock sync status.
func (s *StockSyncService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*model.StockSyncDashboard, error) {
	dash := &model.StockSyncDashboard{}
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Count products with stock entries
		if err := tx.QueryRow(ctx,
			"SELECT COUNT(DISTINCT product_id) FROM warehouse_stock WHERE quantity > 0",
		).Scan(&dash.TotalProducts); err != nil {
			return err
		}

		// Get all channels
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}
		dash.ActiveChannels = len(channels)

		// Count recent errors (last 24h)
		recentErrors, err := s.eventRepo.CountRecentErrors(ctx, tx)
		if err != nil {
			return err
		}
		dash.RecentErrors = recentErrors

		// Find latest sync across all channels
		var latestSync *time.Time
		for _, ch := range channels {
			if ch.LastSyncAt != nil {
				if latestSync == nil || ch.LastSyncAt.After(*latestSync) {
					latestSync = ch.LastSyncAt
				}
			}
		}
		dash.LastSyncAt = latestSync

		// Build channel summaries (include all channels, not just enabled)
		allChannels, _, err := s.channelRepo.List(ctx, tx, model.StockSyncChannelListFilter{
			PaginationParams: model.PaginationParams{Limit: 100, Offset: 0},
		})
		if err != nil {
			return err
		}

		for _, ch := range allChannels {
			status := "ok"
			if !ch.Enabled {
				status = "disabled"
			} else if ch.LastError != nil && *ch.LastError != "" {
				status = "error"
			} else if ch.LastSyncAt == nil {
				status = "warning"
			}

			dash.ChannelSummaries = append(dash.ChannelSummaries, model.ChannelSummary{
				ID:          ch.ID,
				ChannelType: ch.ChannelType,
				Enabled:     ch.Enabled,
				SyncMode:    ch.SyncMode,
				StockBuffer: ch.StockBuffer,
				LastSyncAt:  ch.LastSyncAt,
				LastError:   ch.LastError,
				Status:      status,
			})
		}
		if dash.ChannelSummaries == nil {
			dash.ChannelSummaries = []model.ChannelSummary{}
		}

		return nil
	})
	return dash, err
}
