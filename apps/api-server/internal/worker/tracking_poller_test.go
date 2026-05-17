package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackableShipmentsQueryUsesStableBatchLimit(t *testing.T) {
	query := trackableShipmentsQuery()

	assert.Contains(t, query, "JOIN integrations")
	assert.NotContains(t, query, "LEFT JOIN integrations")
	assert.Contains(t, query, "i.credentials IS NOT NULL")
	assert.Contains(t, query, `i.credentials <> '""'::jsonb`)
	assert.Contains(t, query, "i.credentials <> '{}'::jsonb")
	assert.Contains(t, query, "i.credentials <> 'null'::jsonb")
	assert.Contains(t, query, "ORDER BY s.updated_at ASC, s.id ASC")
	assert.Contains(t, query, "LIMIT $1")
	assert.Equal(t, 100, trackingPollerBatchLimit)
}
