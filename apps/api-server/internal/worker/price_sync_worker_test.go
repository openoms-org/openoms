package worker

import (
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/stretchr/testify/assert"
)

func TestChunkPriceUpdates(t *testing.T) {
	items := make([]integration.PriceUpdate, 250)
	for i := range items {
		items[i] = integration.PriceUpdate{ExternalOfferID: "id", Amount: float64(i), Currency: "PLN"}
	}

	chunks := chunkPriceUpdates(items, 100)

	assert.Len(t, chunks, 3)
	assert.Len(t, chunks[0], 100)
	assert.Len(t, chunks[1], 100)
	assert.Len(t, chunks[2], 50)
}

func TestChunkPriceUpdates_Empty(t *testing.T) {
	chunks := chunkPriceUpdates(nil, 100)
	assert.Nil(t, chunks)
}
