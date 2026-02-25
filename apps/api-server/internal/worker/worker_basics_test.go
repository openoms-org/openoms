package worker

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// DelayedActionWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestDelayedActionWorker_Name(t *testing.T) {
	w := NewDelayedActionWorker(nil, nil, nil, slog.Default())
	assert.Equal(t, "delayed_action_executor", w.Name())
}

func TestDelayedActionWorker_Interval(t *testing.T) {
	w := NewDelayedActionWorker(nil, nil, nil, slog.Default())
	assert.Equal(t, 30*time.Second, w.Interval())
}

func TestDelayedActionWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewDelayedActionWorker(nil, nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// ExchangeRateWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestExchangeRateWorker_Name(t *testing.T) {
	w := NewExchangeRateWorker(nil, nil, slog.Default())
	assert.Equal(t, "exchange_rate_fetcher", w.Name())
}

func TestExchangeRateWorker_Interval(t *testing.T) {
	w := NewExchangeRateWorker(nil, nil, slog.Default())
	assert.Equal(t, 24*time.Hour, w.Interval())
}

func TestExchangeRateWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewExchangeRateWorker(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// KSeFStatusWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestKSeFStatusWorker_Name(t *testing.T) {
	w := NewKSeFStatusWorker(nil, nil, slog.Default())
	assert.Equal(t, "ksef_status_checker", w.Name())
}

func TestKSeFStatusWorker_Interval(t *testing.T) {
	w := NewKSeFStatusWorker(nil, nil, slog.Default())
	assert.Equal(t, 5*time.Minute, w.Interval())
}

func TestKSeFStatusWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewKSeFStatusWorker(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// ListingSyncWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestListingSyncWorker_Name(t *testing.T) {
	w := NewListingSyncWorker(nil, nil, nil, slog.Default())
	assert.Equal(t, "listing_sync", w.Name())
}

func TestListingSyncWorker_Interval(t *testing.T) {
	w := NewListingSyncWorker(nil, nil, nil, slog.Default())
	assert.Equal(t, 5*time.Minute, w.Interval())
}

func TestListingSyncWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewListingSyncWorker(nil, nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// OAuthRefresher — Name / Interval
// ---------------------------------------------------------------------------

func TestOAuthRefresher_Name(t *testing.T) {
	w := NewOAuthRefresher(nil, nil, slog.Default())
	assert.Equal(t, "oauth_refresher", w.Name())
}

func TestOAuthRefresher_Interval(t *testing.T) {
	w := NewOAuthRefresher(nil, nil, slog.Default())
	assert.Equal(t, 30*time.Minute, w.Interval())
}

func TestOAuthRefresher_ImplementsWorkerInterface(_ *testing.T) {
	w := NewOAuthRefresher(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// RecurringOrderWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestRecurringOrderWorker_Name(t *testing.T) {
	w := NewRecurringOrderWorker(nil, nil, slog.Default())
	assert.Equal(t, "recurring_order_processor", w.Name())
}

func TestRecurringOrderWorker_Interval(t *testing.T) {
	w := NewRecurringOrderWorker(nil, nil, slog.Default())
	assert.Equal(t, 1*time.Hour, w.Interval())
}

func TestRecurringOrderWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewRecurringOrderWorker(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// RepricingWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestRepricingWorker_Name(t *testing.T) {
	w := NewRepricingWorker(nil, nil, slog.Default())
	assert.Equal(t, "repricing_engine", w.Name())
}

func TestRepricingWorker_Interval(t *testing.T) {
	w := NewRepricingWorker(nil, nil, slog.Default())
	assert.Equal(t, 15*time.Minute, w.Interval())
}

func TestRepricingWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewRepricingWorker(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// SupplierSyncWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestSupplierSyncWorker_Name(t *testing.T) {
	w := NewSupplierSyncWorker(nil, nil, slog.Default())
	assert.Equal(t, "supplier-sync", w.Name())
}

func TestSupplierSyncWorker_Interval(t *testing.T) {
	w := NewSupplierSyncWorker(nil, nil, slog.Default())
	assert.Equal(t, 1*time.Minute, w.Interval())
}

func TestSupplierSyncWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewSupplierSyncWorker(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// TrackingPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestTrackingPoller_Name(t *testing.T) {
	w := NewTrackingPoller(nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "tracking_poller", w.Name())
}

func TestTrackingPoller_Interval(t *testing.T) {
	w := NewTrackingPoller(nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 10*time.Minute, w.Interval())
}

func TestTrackingPoller_ImplementsWorkerInterface(_ *testing.T) {
	w := NewTrackingPoller(nil, nil, nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// PriceSyncWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestPriceSyncWorker_Name(t *testing.T) {
	w := NewPriceSyncWorker(nil, nil, slog.Default())
	assert.Equal(t, "price_sync", w.Name())
}

func TestPriceSyncWorker_Interval(t *testing.T) {
	w := NewPriceSyncWorker(nil, nil, slog.Default())
	assert.Equal(t, 5*time.Minute, w.Interval())
}

func TestPriceSyncWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewPriceSyncWorker(nil, nil, slog.Default())
	var _ Worker = w
}

// ---------------------------------------------------------------------------
// StockSyncWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestStockSyncWorker_Name(t *testing.T) {
	w := NewStockSyncWorker(nil, nil, slog.Default())
	assert.Equal(t, "stock_sync", w.Name())
}

func TestStockSyncWorker_Interval(t *testing.T) {
	w := NewStockSyncWorker(nil, nil, slog.Default())
	assert.Equal(t, 5*time.Minute, w.Interval())
}

func TestStockSyncWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewStockSyncWorker(nil, nil, slog.Default())
	var _ Worker = w
}
