package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	amazonintegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/amazon"
)

type pendingFeedListing struct {
	ListingID string
	FeedID    string
	FeedType  string
}

// AmazonFeedStatusWorker polls Amazon GetFeed() for pending listings
// that have an amazon_feed_id in their metadata, and resolves them
// to 'synced' or 'error' based on the feed processing status.
type AmazonFeedStatusWorker struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	logger        *slog.Logger
}

func NewAmazonFeedStatusWorker(pool *pgxpool.Pool, encryptionKey []byte, logger *slog.Logger) *AmazonFeedStatusWorker {
	return &AmazonFeedStatusWorker{pool: pool, encryptionKey: encryptionKey, logger: logger}
}

func (w *AmazonFeedStatusWorker) Name() string           { return "amazon_feed_status" }
func (w *AmazonFeedStatusWorker) Interval() time.Duration { return 2 * time.Minute }

func (w *AmazonFeedStatusWorker) Run(ctx context.Context) error {
	tis, err := ListActiveIntegrations(ctx, w.pool, "amazon")
	if err != nil {
		return err
	}
	if len(tis) == 0 {
		return nil
	}

	totalChecked, totalResolved := 0, 0

	for _, ti := range tis {
		// Gather pending listings with amazon_feed_id
		var pending []pendingFeedListing
		gatherErr := database.WithTenant(ctx, w.pool, ti.TenantID, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx,
				`SELECT id, metadata->>'amazon_feed_id', COALESCE(metadata->>'amazon_feed_type', 'inventory')
				 FROM product_listings
				 WHERE integration_id = $1 AND sync_status = 'pending'
				   AND metadata->>'amazon_feed_id' IS NOT NULL
				 LIMIT 1000`,
				ti.IntegrationID)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var p pendingFeedListing
				if err := rows.Scan(&p.ListingID, &p.FeedID, &p.FeedType); err != nil {
					w.logger.Error("amazon feed status: scan", "error", err)
					continue
				}
				pending = append(pending, p)
			}
			return rows.Err()
		})
		if gatherErr != nil {
			w.logger.Error("amazon feed status: gather", "tenant_id", ti.TenantID, "error", gatherErr)
			continue
		}
		if len(pending) == 0 {
			continue
		}

		// Group by feedID
		feedGroups := make(map[string][]pendingFeedListing)
		for _, p := range pending {
			feedGroups[p.FeedID] = append(feedGroups[p.FeedID], p)
		}

		// Create Amazon provider
		credJSON, err := crypto.Decrypt(ti.Credentials, w.encryptionKey)
		if err != nil {
			w.logger.Error("amazon feed status: decrypt", "error", err)
			continue
		}
		provider, err := integration.NewMarketplaceProvider("amazon", credJSON, ti.Settings)
		if err != nil {
			w.logger.Error("amazon feed status: create provider", "error", err)
			continue
		}
		amazonProvider, ok := provider.(*amazonintegration.Provider)
		if !ok {
			w.logger.Error("amazon feed status: unexpected provider type")
			closeProvider(provider)
			continue
		}

		for feedID, listings := range feedGroups {
			totalChecked++
			feed, err := amazonProvider.GetFeedStatus(ctx, feedID)
			if err != nil {
				w.logger.Error("amazon feed status: get feed", "feed_id", feedID, "error", err)
				continue
			}

			switch feed.ProcessingStatus {
			case "DONE":
				w.resolveListings(ctx, ti.TenantID, listings, "synced", nil)
				totalResolved += len(listings)
			case "CANCELLED", "FATAL":
				errMsg := fmt.Sprintf("Amazon feed %s: %s", feedID, feed.ProcessingStatus)
				w.resolveListings(ctx, ti.TenantID, listings, "error", &errMsg)
				totalResolved += len(listings)
			default:
				// IN_QUEUE, IN_PROGRESS — still processing, leave as pending
			}
			time.Sleep(500 * time.Millisecond) // rate limit between feed status checks
		}
		closeProvider(provider)
	}

	if totalChecked > 0 {
		w.logger.Info("amazon feed status: cycle complete", "feeds_checked", totalChecked, "resolved", totalResolved)
	}
	return nil
}

func (w *AmazonFeedStatusWorker) resolveListings(ctx context.Context, tenantID uuid.UUID, listings []pendingFeedListing, status string, errMsg *string) {
	_ = database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
		for _, l := range listings {
			var execErr error
			switch status {
			case "synced":
				_, execErr = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(),
					 metadata = metadata - 'amazon_feed_id' - 'amazon_feed_type', updated_at = NOW() WHERE id = $1`,
					l.ListingID)
			default:
				msg := ""
				if errMsg != nil {
					msg = *errMsg
				}
				_, execErr = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'error', error_message = $2,
					 metadata = metadata - 'amazon_feed_id' - 'amazon_feed_type', updated_at = NOW() WHERE id = $1`,
					l.ListingID, msg)
			}
			if execErr != nil {
				w.logger.Error("amazon feed status: update listing", "listing_id", l.ListingID, "error", execErr)
			}
		}
		return nil
	})
}
