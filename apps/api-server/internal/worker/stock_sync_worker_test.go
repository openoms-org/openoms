package worker

import (
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/stretchr/testify/assert"
)

func TestChunkSlice(t *testing.T) {
	items := make([]integration.StockUpdate, 250)
	for i := range items {
		items[i] = integration.StockUpdate{ExternalOfferID: "offer", Quantity: i}
	}

	chunks := chunkStockUpdates(items, 100)

	assert.Len(t, chunks, 3)
	assert.Len(t, chunks[0], 100)
	assert.Len(t, chunks[1], 100)
	assert.Len(t, chunks[2], 50)
}

func TestChunkSlice_Empty(t *testing.T) {
	chunks := chunkStockUpdates(nil, 100)
	assert.Nil(t, chunks)
}
