package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	allegroint "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// AllegroOrderInboundService lists seller checkout-forms and upserts them as OMS orders.
type AllegroOrderInboundService struct {
	integrationService *IntegrationService
	orderRepo          repository.OrderRepo
	auditRepo          repository.AuditRepo
	pool               *pgxpool.Pool
	logger             *slog.Logger
}

// AllegroOrderSyncResult is the response for a manual inbound order sync.
type AllegroOrderSyncResult struct {
	SyncedCount  int    `json:"synced_count"`
	CreatedCount int    `json:"created_count"`
	Cursor       string `json:"cursor"`
}

// NewAllegroOrderInboundService creates an inbound checkout-form importer.
func NewAllegroOrderInboundService(
	integrationService *IntegrationService,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *AllegroOrderInboundService {
	return &AllegroOrderInboundService{
		integrationService: integrationService,
		orderRepo:          orderRepo,
		auditRepo:          auditRepo,
		pool:               pool,
		logger:             slog.Default().With("component", "allegro_order_inbound"),
	}
}

// SyncOrders lists checkout-forms for the tenant's Allegro integration and upserts
// each as source=allegro, external_id=checkout-form id. Always stamps last_sync_at
// on success so the connection UI is not stuck on "---".
func (s *AllegroOrderInboundService) SyncOrders(ctx context.Context, tenantID uuid.UUID) (*AllegroOrderSyncResult, error) {
	credJSON, integ, err := s.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, err
	}

	provider, err := allegroint.NewProvider(credJSON, nil, allegroint.WithTokenRefreshPersist(
		allegroint.PersistFn(credJSON, func(newJSON []byte) error {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			return s.integrationService.UpdateCredentialsByProvider(persistCtx, tenantID, "allegro", newJSON)
		}),
	))
	if err != nil {
		return nil, fmt.Errorf("allegro inbound: create provider: %w", err)
	}
	defer provider.Close()

	orders, cursor, err := provider.PollOrders(ctx, "")
	if err != nil {
		return nil, err
	}

	createdCount := 0
	for _, mo := range orders {
		if mo.ExternalID == "" {
			continue
		}
		created := false
		if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			orderID, upsertErr := upsertAllegroCheckoutForm(ctx, tx, s.orderRepo, mo, tenantID, integ.ID)
			if upsertErr != nil {
				return upsertErr
			}
			if orderID == uuid.Nil {
				return nil
			}
			created = true
			if s.auditRepo == nil {
				return nil
			}
			return s.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     uuid.Nil,
				Action:     "order.created",
				EntityType: "order",
				EntityID:   orderID,
				Changes:    map[string]string{"source": "allegro", "trigger": "manual_sync", "external_id": mo.ExternalID},
				IPAddress:  "0.0.0.0",
			})
		}); err != nil {
			return nil, fmt.Errorf("allegro inbound: upsert %s: %w", mo.ExternalID, err)
		}
		if created {
			createdCount++
		}
	}

	if err := s.markLastSync(ctx, tenantID, integ.ID, cursor); err != nil {
		return nil, fmt.Errorf("allegro inbound: mark last sync: %w", err)
	}

	s.logger.Info("allegro inbound sync completed",
		"tenant_id", tenantID,
		"integration_id", integ.ID,
		"listed", len(orders),
		"created", createdCount,
	)

	return &AllegroOrderSyncResult{
		SyncedCount:  len(orders),
		CreatedCount: createdCount,
		Cursor:       cursor,
	}, nil
}

func (s *AllegroOrderInboundService) markLastSync(ctx context.Context, tenantID, integrationID uuid.UUID, cursor string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE integrations SET last_sync_at = NOW(), sync_cursor = $1, error_message = NULL, updated_at = NOW() WHERE id = $2`,
			cursor, integrationID,
		)
		return err
	})
}

func upsertAllegroCheckoutForm(
	ctx context.Context,
	tx pgx.Tx,
	orderRepo repository.OrderRepo,
	mo integration.MarketplaceOrder,
	tenantID, integrationID uuid.UUID,
) (uuid.UUID, error) {
	existing, err := orderRepo.FindByExternalID(ctx, tx, "allegro", mo.ExternalID)
	if err != nil {
		return uuid.Nil, err
	}
	if existing != nil {
		return uuid.Nil, nil
	}

	order := allegroint.ToOMSOrder(mo, tenantID, integrationID)
	created, err := orderRepo.CreateIfExternalIDNotExists(ctx, tx, &order)
	if err != nil || !created {
		return uuid.Nil, err
	}
	return order.ID, nil
}
