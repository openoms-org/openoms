package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildSegmentMemberInsert(t *testing.T) {
	tenantID := uuid.New()
	segmentID := uuid.New()
	c1, c2 := uuid.New(), uuid.New()

	t.Run("single row", func(t *testing.T) {
		q, args := buildSegmentMemberInsert(tenantID, segmentID, []uuid.UUID{c1})
		assert.Contains(t, q, "INSERT INTO customer_segment_members (tenant_id, segment_id, customer_id) VALUES ($1, $2, $3)")
		assert.Contains(t, q, "ON CONFLICT DO NOTHING")
		assert.Equal(t, []any{tenantID, segmentID, c1}, args)
	})

	t.Run("multi row uses sequential placeholders", func(t *testing.T) {
		q, args := buildSegmentMemberInsert(tenantID, segmentID, []uuid.UUID{c1, c2})
		assert.Contains(t, q, "VALUES ($1, $2, $3), ($4, $5, $6) ON CONFLICT DO NOTHING")
		// tenantID and segmentID repeat per row; customer id varies.
		assert.Equal(t, []any{tenantID, segmentID, c1, tenantID, segmentID, c2}, args)
		assert.Len(t, args, 6)
	})
}
