package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// TestAllegroSyncService_WithFulfillment verifies the OPE-417-followup recorder wiring:
// the setter is nil-safe + chainable, and defaults to no recorder.
func TestAllegroSyncService_WithFulfillment(t *testing.T) {
	s := NewAllegroSyncService(nil)
	assert.Nil(t, s.fulfillment, "no recorder wired by default")

	f := NewFulfillmentService(false, nil, nil)
	got := s.WithFulfillment(f)
	assert.Same(t, s, got, "WithFulfillment returns the service for chaining")
	assert.Same(t, f, s.fulfillment)

	var nilSvc *AllegroSyncService
	assert.NotPanics(t, func() {
		assert.Nil(t, nilSvc.WithFulfillment(f))
	})
}

// TestAllegroSyncService_RecordHelpers_NoRecorder proves the recording helpers are a
// complete no-op (and never panic / touch a DB) when no fulfillment recorder is wired —
// the gated-off default, so instrumentation can never affect the sync itself.
func TestAllegroSyncService_RecordHelpers_NoRecorder(t *testing.T) {
	s := NewAllegroSyncService(nil) // no fulfillment recorder
	order := &model.Order{ID: uuid.New()}
	assert.NotPanics(t, func() {
		s.recordMarketplaceAttempt(context.Background(), uuid.New(), order,
			model.ProviderOpSyncTrackingToMarketplace, model.ProviderAttemptSucceeded, "INPOST")
		s.emitTrackingSyncStep(context.Background(), uuid.New(), order, model.FulfillmentStatusSucceeded)
	})
}
