//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// lsEncKey is the 32-byte AES-256 key used both to encrypt the seeded integration
// credentials and to construct the services under test, so crypto.Decrypt round-trips.
var lsEncKey = []byte("0123456789abcdef0123456789abcdef")

// --- Fake marketplace provider that records UpdatePrice / UpdateStock calls ---
//
// The provider factory is the same seam the wired stock_sync / price_sync code uses
// (integration.NewMarketplaceProvider is a name-keyed factory). Tests run
// sequentially within the package, so a package-level recorder is reset per test.

type lsCall struct {
	op    string // "price" or "stock"
	ext   string
	price float64
	qty   int
}

type lsRecorder struct {
	mu    sync.Mutex
	calls []lsCall
}

func (r *lsRecorder) add(c lsCall) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, c)
}

func (r *lsRecorder) byOp(op string) []lsCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []lsCall
	for _, c := range r.calls {
		if c.op == op {
			out = append(out, c)
		}
	}
	return out
}

var (
	lsRec *lsRecorder

	// Structural probe (case f): when set, the provider performs an independent
	// WithTenant read DURING the push. On a single-connection pool this only
	// succeeds if the service is NOT holding a transaction across the push.
	lsProbePool   *pgxpool.Pool
	lsProbeTenant uuid.UUID
	lsProbeOK     bool
)

type lsFakeProvider struct{ failPrice bool }

func (p *lsFakeProvider) ProviderName() string { return "fake_listing_sync" }
func (p *lsFakeProvider) PollOrders(context.Context, string) ([]integration.MarketplaceOrder, string, error) {
	return nil, "", nil
}
func (p *lsFakeProvider) GetOrder(context.Context, string) (*integration.MarketplaceOrder, error) {
	return nil, nil
}
func (p *lsFakeProvider) PushOffer(context.Context, *model.Product, map[string]any) (string, error) {
	return "", nil
}
func (p *lsFakeProvider) UpdateStock(_ context.Context, ext string, qty int) error {
	lsRec.add(lsCall{op: "stock", ext: ext, qty: qty})
	return nil
}
func (p *lsFakeProvider) UpdatePrice(ctx context.Context, ext string, price float64) error {
	lsRec.add(lsCall{op: "price", ext: ext, price: price})
	lsRunProbe(ctx)
	if p.failPrice {
		return errors.New("simulated price push failure")
	}
	return nil
}

// lsFakeAsyncProvider additionally implements AsyncPriceUpdater + AsyncFeedResult so
// the service persists sync_status='pending' plus feed metadata. It mirrors the real
// Amazon provider's timing precisely: lastFeed is reset at the start of each
// UpdatePrice and set — to a per-offer feed id — only AFTER the feed submission
// succeeds. FeedResult therefore returns nil before any push and a distinct id per
// offer afterwards. A service that captured feed metadata before the push loop, or
// reused one id across offers, would be caught by this fake.
type lsFakeAsyncProvider struct {
	lsFakeProvider
	lastFeed *integration.FeedSubmission
}

func (p *lsFakeAsyncProvider) ProviderName() string { return "fake_listing_sync_async" }
func (p *lsFakeAsyncProvider) IsAsyncPriceUpdate()  {}
func (p *lsFakeAsyncProvider) UpdatePrice(ctx context.Context, ext string, price float64) error {
	p.lastFeed = nil
	if err := p.lsFakeProvider.UpdatePrice(ctx, ext, price); err != nil {
		return err
	}
	p.lastFeed = &integration.FeedSubmission{FeedID: "FEED-" + ext, FeedType: "pricing"}
	return nil
}
func (p *lsFakeAsyncProvider) FeedResult() *integration.FeedSubmission {
	return p.lastFeed
}

func lsRunProbe(ctx context.Context) {
	if lsProbePool == nil {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := database.WithTenant(cctx, lsProbePool, lsProbeTenant, func(tx pgx.Tx) error {
		var n int
		return tx.QueryRow(cctx, "SELECT 1").Scan(&n)
	}); err == nil {
		lsProbeOK = true
	}
}

func init() {
	integration.RegisterMarketplaceProvider("fake_listing_sync", func(_ json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return &lsFakeProvider{failPrice: lsSettingsFailPrice(settings)}, nil
	})
	integration.RegisterMarketplaceProvider("fake_listing_sync_async", func(_ json.RawMessage, _ json.RawMessage) (integration.MarketplaceProvider, error) {
		return &lsFakeAsyncProvider{}, nil
	})
}

func lsSettingsFailPrice(settings json.RawMessage) bool {
	if len(settings) == 0 {
		return false
	}
	var s struct {
		FailPrice bool `json:"fail_price"`
	}
	_ = json.Unmarshal(settings, &s)
	return s.FailPrice
}

// --- Seeding helpers ---

func lsResetRecorder() {
	lsRec = &lsRecorder{}
	lsProbePool = nil
	lsProbeTenant = uuid.Nil
	lsProbeOK = false
}

func lsNewServices(pool *pgxpool.Pool) *service.ListingSyncService {
	stockSvc := service.NewStockSyncService(
		repository.NewStockSyncChannelRepository(),
		repository.NewStockSyncEventRepository(),
		repository.NewProductRepository(),
		repository.NewAuditRepository(),
		repository.NewProductListingRepository(),
		repository.NewIntegrationRepository(),
		pool, nil, lsEncKey, slog.Default(),
	)
	listingSvc := service.NewListingSyncService(
		repository.NewListingSyncRepository(),
		repository.NewProductRepository(),
		repository.NewProductListingRepository(),
		repository.NewAuditRepository(),
		repository.NewIntegrationRepository(),
		pool, lsEncKey, slog.Default(),
	)
	listingSvc.SetStockSyncService(stockSvc)
	return listingSvc
}

// lsSeedIntegrationAndConfig inserts an active integration with encrypted credentials
// and a push-direction listing sync config, returning their ids.
func lsSeedIntegrationAndConfig(t *testing.T, ctx context.Context, tenantID uuid.UUID, provider, settings, priceRule string, priceModifier float64) (integrationID, configID uuid.UUID) {
	t.Helper()
	integrationID = uuid.New()
	configID = uuid.New()
	ct, err := crypto.Encrypt([]byte(`{"token":"x"}`), lsEncKey)
	require.NoError(t, err)
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx,
			`INSERT INTO integrations (id, tenant_id, provider, status, credentials, settings)
			 VALUES ($1,$2,$3,'active', to_jsonb($4::text), $5::jsonb)`,
			integrationID, tenantID, provider, ct, settings); e != nil {
			return e
		}
		_, e := tx.Exec(ctx,
			`INSERT INTO listing_sync_configs (id, tenant_id, integration_id, sync_direction, price_rule, price_modifier)
			 VALUES ($1,$2,$3,'push',$4,$5)`,
			configID, tenantID, integrationID, priceRule, priceModifier)
		return e
	}))
	return integrationID, configID
}

func lsSeedProduct(t *testing.T, ctx context.Context, tenantID uuid.UUID, price float64, stockQty int) uuid.UUID {
	t.Helper()
	var productID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO products (tenant_id, name, price, stock_quantity) VALUES ($1,'P',$2,$3) RETURNING id`,
			tenantID, price, stockQty).Scan(&productID)
	}))
	return productID
}

func lsSeedListing(t *testing.T, ctx context.Context, tenantID, productID, integrationID uuid.UUID, externalID *string, status, mode string, priceOverride *float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO product_listings (id, tenant_id, product_id, integration_id, external_id, status, stock_sync_mode, price_override)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			id, tenantID, productID, integrationID, externalID, status, mode, priceOverride)
		return e
	}))
	return id
}

func lsSeedWarehouseStock(t *testing.T, ctx context.Context, tenantID, productID uuid.UUID, quantity, reserved int) {
	t.Helper()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var warehouseID uuid.UUID
		if e := tx.QueryRow(ctx,
			`INSERT INTO warehouses (tenant_id, name, is_default) VALUES ($1,'WH',true) RETURNING id`,
			tenantID).Scan(&warehouseID); e != nil {
			return e
		}
		_, e := tx.Exec(ctx,
			`INSERT INTO warehouse_stock (tenant_id, warehouse_id, product_id, quantity, reserved)
			 VALUES ($1,$2,$3,$4,$5)`,
			tenantID, warehouseID, productID, quantity, reserved)
		return e
	}))
}

type lsListingState struct {
	syncStatus    string
	priceOverride *float64
	errorMessage  *string
	lastSyncedAt  *time.Time
	metadata      string
}

func lsReadListing(t *testing.T, ctx context.Context, tenantID, listingID uuid.UUID) lsListingState {
	t.Helper()
	var st lsListingState
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT sync_status, price_override, error_message, last_synced_at, metadata::text
			 FROM product_listings WHERE id = $1`, listingID).
			Scan(&st.syncStatus, &st.priceOverride, &st.errorMessage, &st.lastSyncedAt, &st.metadata)
	}))
	return st
}

func f64(v float64) *float64  { return &v }
func strptr(v string) *string { return &v }

// --- Cases (a) + (b): pushes eligible listings with markup, skips ineligible ---

func TestSyncPrices_PushesEligibleWithMarkup_SkipsIneligible(t *testing.T) {
	ctx := context.Background()
	lsResetRecorder()
	tenantID := seedTenant(t, ctx)
	integrationID, configID := lsSeedIntegrationAndConfig(t, ctx, tenantID, "fake_listing_sync", "{}", "markup_pct", 10)

	// product price 100 -> markup 10% -> 110.00 expected. Each listing needs its own
	// product (unique index on tenant_id, product_id, integration_id).
	prod := lsSeedProduct(t, ctx, tenantID, 100, 5)
	eligible := lsSeedListing(t, ctx, tenantID, prod, integrationID, strptr("OFFER-A"), "active", "auto", nil)

	// Ineligible siblings: nil external_id, manual mode, inactive status.
	_ = lsSeedListing(t, ctx, tenantID, lsSeedProduct(t, ctx, tenantID, 100, 5), integrationID, nil, "active", "auto", nil)
	_ = lsSeedListing(t, ctx, tenantID, lsSeedProduct(t, ctx, tenantID, 100, 5), integrationID, strptr("OFFER-MANUAL"), "active", "manual", nil)
	_ = lsSeedListing(t, ctx, tenantID, lsSeedProduct(t, ctx, tenantID, 100, 5), integrationID, strptr("OFFER-INACTIVE"), "inactive", "auto", nil)

	svc := lsNewServices(lsAppPool(t, ctx, 4))
	res, err := svc.SyncPrices(ctx, tenantID, configID)
	require.NoError(t, err)

	priceCalls := lsRec.byOp("price")
	require.Len(t, priceCalls, 1, "exactly one eligible listing pushed")
	assert.Equal(t, "OFFER-A", priceCalls[0].ext)
	assert.InDelta(t, 110.0, priceCalls[0].price, 0.001, "markup-applied price pushed")
	assert.Equal(t, 1, res.ItemsProcessed)
	assert.Equal(t, 0, res.ItemsFailed)

	st := lsReadListing(t, ctx, tenantID, eligible)
	assert.Equal(t, "synced", st.syncStatus)
	require.NotNil(t, st.priceOverride)
	assert.InDelta(t, 110.0, *st.priceOverride, 0.001, "price_override persisted to the markup value")
	assert.NotNil(t, st.lastSyncedAt, "last_synced_at set on success")
}

// --- Case (c): unchanged price makes ZERO provider calls (idempotency) ---

func TestSyncPrices_UnchangedPrice_NoProviderCall(t *testing.T) {
	ctx := context.Background()
	lsResetRecorder()
	tenantID := seedTenant(t, ctx)
	integrationID, configID := lsSeedIntegrationAndConfig(t, ctx, tenantID, "fake_listing_sync", "{}", "markup_pct", 10)

	prod := lsSeedProduct(t, ctx, tenantID, 100, 5)
	// price_override already equals the computed markup (110) -> nothing to push.
	listing := lsSeedListing(t, ctx, tenantID, prod, integrationID, strptr("OFFER-C"), "active", "auto", f64(110))

	svc := lsNewServices(lsAppPool(t, ctx, 4))
	res, err := svc.SyncPrices(ctx, tenantID, configID)
	require.NoError(t, err)

	assert.Empty(t, lsRec.byOp("price"), "no UpdatePrice call for an unchanged offer (no duplicate feed)")
	assert.Equal(t, 1, res.ItemsProcessed, "the unchanged listing still counts as processed")
	assert.Equal(t, 0, res.ItemsFailed)

	st := lsReadListing(t, ctx, tenantID, listing)
	require.NotNil(t, st.priceOverride)
	assert.InDelta(t, 110.0, *st.priceOverride, 0.001)
}

// --- Case (d): provider error -> sync_status='error' + error_message, not 'synced' ---

func TestSyncPrices_ProviderError_SetsErrorStatus(t *testing.T) {
	ctx := context.Background()
	lsResetRecorder()
	tenantID := seedTenant(t, ctx)
	integrationID, configID := lsSeedIntegrationAndConfig(t, ctx, tenantID, "fake_listing_sync", `{"fail_price":true}`, "markup_pct", 10)

	prod := lsSeedProduct(t, ctx, tenantID, 100, 5)
	listing := lsSeedListing(t, ctx, tenantID, prod, integrationID, strptr("OFFER-D"), "active", "auto", nil)

	svc := lsNewServices(lsAppPool(t, ctx, 4))
	res, err := svc.SyncPrices(ctx, tenantID, configID)
	require.NoError(t, err)

	require.Len(t, lsRec.byOp("price"), 1, "the push was attempted")
	assert.Equal(t, 1, res.ItemsFailed)
	assert.Equal(t, 0, res.ItemsProcessed)

	st := lsReadListing(t, ctx, tenantID, listing)
	assert.Equal(t, "error", st.syncStatus)
	require.NotNil(t, st.errorMessage)
	assert.Contains(t, *st.errorMessage, "simulated price push failure")
	assert.Nil(t, st.priceOverride, "price_override not advanced on a failed push")
}

// --- Case (e): async feed provider -> sync_status='pending' + feed metadata ---

func TestSyncPrices_AsyncProvider_PendingWithFeedMeta(t *testing.T) {
	ctx := context.Background()
	lsResetRecorder()
	tenantID := seedTenant(t, ctx)
	integrationID, configID := lsSeedIntegrationAndConfig(t, ctx, tenantID, "fake_listing_sync_async", "{}", "markup_pct", 10)

	// Two eligible listings so we can assert each pending listing carries ITS OWN
	// feed id — proving feed metadata is captured after each push, not hoisted above
	// the loop (where it would be nil) nor shared across offers.
	prod1 := lsSeedProduct(t, ctx, tenantID, 100, 5)
	listing1 := lsSeedListing(t, ctx, tenantID, prod1, integrationID, strptr("OFFER-E1"), "active", "auto", nil)
	prod2 := lsSeedProduct(t, ctx, tenantID, 200, 5)
	listing2 := lsSeedListing(t, ctx, tenantID, prod2, integrationID, strptr("OFFER-E2"), "active", "auto", nil)

	svc := lsNewServices(lsAppPool(t, ctx, 4))
	res, err := svc.SyncPrices(ctx, tenantID, configID)
	require.NoError(t, err)
	require.Len(t, lsRec.byOp("price"), 2)
	assert.Equal(t, 2, res.ItemsProcessed)

	st1 := lsReadListing(t, ctx, tenantID, listing1)
	assert.Equal(t, "pending", st1.syncStatus, "async provider marks pending, not synced")
	assert.Contains(t, st1.metadata, "FEED-OFFER-E1", "listing 1's own feed id persisted for status polling")
	assert.NotContains(t, st1.metadata, "FEED-OFFER-E2", "feed ids are not shared across offers")
	require.NotNil(t, st1.priceOverride)
	assert.InDelta(t, 110.0, *st1.priceOverride, 0.001)

	st2 := lsReadListing(t, ctx, tenantID, listing2)
	assert.Equal(t, "pending", st2.syncStatus)
	assert.Contains(t, st2.metadata, "FEED-OFFER-E2", "listing 2's own feed id persisted for status polling")
	require.NotNil(t, st2.priceOverride)
	assert.InDelta(t, 220.0, *st2.priceOverride, 0.001)
}

// --- Case (f): push happens with NO DB transaction held (two-phase) ---

func TestSyncPrices_PushOutsideTransaction(t *testing.T) {
	ctx := context.Background()
	lsResetRecorder()
	tenantID := seedTenant(t, ctx)
	integrationID, configID := lsSeedIntegrationAndConfig(t, ctx, tenantID, "fake_listing_sync", "{}", "markup_pct", 10)

	prod := lsSeedProduct(t, ctx, tenantID, 100, 5)
	_ = lsSeedListing(t, ctx, tenantID, prod, integrationID, strptr("OFFER-F"), "active", "auto", nil)

	// Run the service against a SINGLE-connection pool. The provider performs an
	// independent WithTenant read during the push; on a 1-conn pool it can only
	// succeed if no gather transaction is held open across the marketplace call.
	single := lsAppPool(t, ctx, 1)
	lsProbePool = single
	lsProbeTenant = tenantID

	svc := lsNewServices(single)
	_, err := svc.SyncPrices(ctx, tenantID, configID)
	require.NoError(t, err)

	require.Len(t, lsRec.byOp("price"), 1, "push attempted")
	assert.True(t, lsProbeOK, "an independent DB read succeeded mid-push -> no transaction held across the marketplace call")
}

// --- Stock delegation: warehouse_stock is the source, not products.stock_quantity ---

func TestSyncStock_DelegatesToWarehouseStockSource(t *testing.T) {
	ctx := context.Background()
	lsResetRecorder()
	tenantID := seedTenant(t, ctx)
	integrationID, configID := lsSeedIntegrationAndConfig(t, ctx, tenantID, "fake_listing_sync", "{}", "same", 0)

	// products.stock_quantity is deliberately the WRONG (stale) source.
	prod := lsSeedProduct(t, ctx, tenantID, 100, 999)
	// warehouse_stock is canonical: 42 - 2 reserved = 40 available.
	lsSeedWarehouseStock(t, ctx, tenantID, prod, 42, 2)
	_ = lsSeedListing(t, ctx, tenantID, prod, integrationID, strptr("OFFER-S"), "active", "auto", nil)

	svc := lsNewServices(lsAppPool(t, ctx, 4))
	res, err := svc.SyncStock(ctx, tenantID, configID)
	require.NoError(t, err)

	stockCalls := lsRec.byOp("stock")
	require.Len(t, stockCalls, 1, "stock pushed via the delegated StockSyncService path")
	assert.Equal(t, "OFFER-S", stockCalls[0].ext)
	assert.Equal(t, 40, stockCalls[0].qty, "warehouse-derived quantity, NOT products.stock_quantity (999)")
	assert.NotEqual(t, 999, stockCalls[0].qty)
	assert.Equal(t, 1, res.ItemsProcessed)

	// No price calls were made by a stock sync.
	assert.Empty(t, lsRec.byOp("price"))
}

// lsAppPool opens a pool authenticated as openoms_app (so RLS applies) with the
// given connection cap, using the SAME simple query protocol production runs behind
// the Supabase transaction pooler — exactly as the shared harness appPool does. This
// is deliberate: the service's batch product fetch must encode its parameters under
// simple protocol just as in production (which is why ProductRepository.FindByIDs
// uses scalar IN placeholders rather than = ANY on a []uuid.UUID — the latter cannot
// be encoded under simple protocol). A per-test pool, rather than the shared appPool,
// is used only so the structural two-phase probe (case f) can pin a hard
// single-connection cap.
func lsAppPool(t *testing.T, ctx context.Context, maxConns int32) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, raw)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	u.User = url.UserPassword("openoms_app", appRolePassword)
	cfg, err := pgxpool.ParseConfig(u.String())
	require.NoError(t, err)
	cfg.MaxConns = maxConns
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		tm := conn.TypeMap()
		tm.RegisterType(&pgtype.Type{
			Name:  "jsonb",
			OID:   pgtype.JSONBOID,
			Codec: &pgtype.JSONBCodec{Marshal: json.Marshal, Unmarshal: json.Unmarshal},
		})
		tm.RegisterDefaultPgType(json.RawMessage{}, "jsonb")
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))
	return pool
}
