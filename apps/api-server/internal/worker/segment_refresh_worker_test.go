package worker

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// SegmentRefreshWorker — Name / Interval
// ---------------------------------------------------------------------------

func TestSegmentRefreshWorker_Name(t *testing.T) {
	w := NewSegmentRefreshWorker(nil, nil, slog.Default())
	assert.Equal(t, "segment_refresh", w.Name())
}

func TestSegmentRefreshWorker_Interval(t *testing.T) {
	w := NewSegmentRefreshWorker(nil, nil, slog.Default())
	assert.Equal(t, 1*time.Hour, w.Interval())
}

func TestSegmentRefreshWorker_ImplementsWorkerInterface(_ *testing.T) {
	w := NewSegmentRefreshWorker(nil, nil, slog.Default())
	var _ Worker = w
}
