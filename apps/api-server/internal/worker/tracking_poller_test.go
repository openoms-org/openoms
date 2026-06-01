package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackableShipmentsQueryUsesKeysetPagination(t *testing.T) {
	query := trackableShipmentsQuery()

	assert.Contains(t, query, "JOIN integrations")
	assert.NotContains(t, query, "LEFT JOIN integrations")
	assert.Contains(t, query, "i.credentials IS NOT NULL")
	assert.Contains(t, query, `i.credentials <> '""'::jsonb`)
	assert.Contains(t, query, "i.credentials <> '{}'::jsonb")
	assert.Contains(t, query, "i.credentials <> 'null'::jsonb")
	assert.Contains(t, query, "ORDER BY s.updated_at ASC, s.id ASC")
	assert.Contains(t, query, "LIMIT $1")
	// Keyset cursor for pagination (OPE-487).
	assert.Contains(t, query, "(s.updated_at, s.id) > ($2, $3)")
	assert.Equal(t, 100, trackingPollerPageSize)
}

func fullPage() []trackableShipment {
	page := make([]trackableShipment, trackingPollerPageSize)
	for i := range page {
		page[i] = trackableShipment{ID: uuid.New(), UpdatedAt: time.Unix(int64(i)+1, 0)}
	}
	return page
}

func TestCollectTrackableShipments_PaginatesUntilShortPage(t *testing.T) {
	calls := 0
	w := &TrackingPoller{logger: slog.Default()}
	w.fetchPage = func(_ context.Context, _ time.Time, _ uuid.UUID) ([]trackableShipment, error) {
		calls++
		if calls <= 2 {
			return fullPage(), nil
		}
		return []trackableShipment{{ID: uuid.New(), UpdatedAt: time.Unix(9999, 0)}}, nil // short page -> stop
	}

	got, err := w.collectTrackableShipments(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 3, calls, "fetch pages until a short (partial) page is returned")
	assert.Len(t, got, 2*trackingPollerPageSize+1)
}

func TestCollectTrackableShipments_AdvancesCursorToPreviousPageTail(t *testing.T) {
	var page1Tail trackableShipment
	var cursorOnPage2 uuid.UUID
	calls := 0

	w := &TrackingPoller{logger: slog.Default()}
	w.fetchPage = func(_ context.Context, _ time.Time, cursorID uuid.UUID) ([]trackableShipment, error) {
		calls++
		if calls == 1 {
			page := fullPage()
			page1Tail = page[len(page)-1]
			return page, nil
		}
		cursorOnPage2 = cursorID
		return []trackableShipment{{ID: uuid.New(), UpdatedAt: time.Unix(9999, 0)}}, nil
	}

	_, err := w.collectTrackableShipments(context.Background())

	require.NoError(t, err)
	assert.Equal(t, page1Tail.ID, cursorOnPage2, "page 2 must be fetched with the cursor at page 1's last row")
}

func TestCollectTrackableShipments_RespectsPerRunCap(t *testing.T) {
	w := &TrackingPoller{logger: slog.Default()}
	w.fetchPage = func(_ context.Context, _ time.Time, _ uuid.UUID) ([]trackableShipment, error) {
		return fullPage(), nil // always full -> would loop forever without the cap
	}

	got, err := w.collectTrackableShipments(context.Background())

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), trackingPollerMaxShipments)
	assert.Less(t, len(got), trackingPollerMaxShipments+trackingPollerPageSize, "stops once the per-run cap is reached")
}
