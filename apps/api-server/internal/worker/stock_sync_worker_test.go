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

func TestTruncateErrorMessage(t *testing.T) {
	short := "short error"
	assert.Equal(t, short, truncateErrorMessage(short))

	long := make([]byte, 600)
	for i := range long {
		long[i] = 'x'
	}
	result := truncateErrorMessage(string(long))
	assert.Len(t, result, 500)
}

func TestCloseProvider_WithCloser(t *testing.T) {
	called := false
	p := &mockCloser{onClose: func() { called = true }}
	closeProvider(p)
	assert.True(t, called)
}

func TestCloseProvider_WithoutCloser(_ *testing.T) {
	// Should not panic on a type without Close()
	closeProvider("not a closer")
}

type mockCloser struct {
	onClose func()
}

func (m *mockCloser) Close() {
	if m.onClose != nil {
		m.onClose()
	}
}
