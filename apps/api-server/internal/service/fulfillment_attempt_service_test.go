package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// TestFulfillmentService_Recording_DisabledNoOp verifies the OPE-417 best-effort
// recording methods are complete no-ops when the service is disabled. With a nil
// pool (recording not wired) AND disabled, none of these may panic or do work —
// the guarantee that gated-off behavior is byte-for-byte unchanged.
func TestFulfillmentService_Recording_DisabledNoOp(t *testing.T) {
	ctx := context.Background()
	svc := NewFulfillmentService(false, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	assert.False(t, svc.Enabled())

	tenantID, orderID := uuid.New(), uuid.New()

	// None of these touch the (nil) pool because recordingReady() is false.
	assert.Nil(t, svc.RecordProviderAttempt(ctx, ProviderAttemptInput{
		TenantID: tenantID, OrderID: orderID, Provider: "inpost", Operation: model.ProviderOpCreateShipment,
	}))
	svc.EmitFulfillmentStep(ctx, tenantID, orderID, model.StepCreateShipment, model.FulfillmentStatusSucceeded, "corr", nil)
	assert.Nil(t, svc.CreateCarrierBlocker(ctx, tenantID, orderID, model.CarrierFailAuth, "x"))
	svc.RecordTrackingSyncAttempt(ctx, tenantID, orderID, "inpost", "corr",
		model.NewTrackingStatusMapping("RAW", "delivered", true), model.ProviderAttemptSucceeded)
	svc.RecordTrackingSyncFailure(ctx, tenantID, orderID, "inpost", "corr", errors.New("boom"))
}

// TestFulfillmentService_Recording_EnabledButUnwiredNoOp verifies that even when
// the service is enabled, recording is a no-op when the pool/attempt repo are not
// wired via WithRecording — so it degrades safely rather than panicking on nil.
func TestFulfillmentService_Recording_EnabledButUnwiredNoOp(t *testing.T) {
	ctx := context.Background()
	svc := NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	assert.True(t, svc.Enabled())
	assert.False(t, svc.recordingReady(), "no pool wired -> recording not ready")

	tenantID, orderID := uuid.New(), uuid.New()
	assert.Nil(t, svc.RecordProviderAttempt(ctx, ProviderAttemptInput{
		TenantID: tenantID, OrderID: orderID, Provider: "dhl", Operation: model.ProviderOpGenerateLabel,
	}))
	svc.EmitFulfillmentStep(ctx, tenantID, orderID, model.StepGenerateLabel, model.FulfillmentStatusFailed, "corr", nil)
	assert.Nil(t, svc.CreateCarrierBlocker(ctx, tenantID, orderID, model.CarrierFailRateLimit, "x"))
}

func TestHashPayload_StableAndOrderSensitive(t *testing.T) {
	a := HashPayload("inpost", "EXT-123", "pdf")
	b := HashPayload("inpost", "EXT-123", "pdf")
	assert.Equal(t, a, b, "same inputs -> same hash (idempotency intent)")
	assert.Len(t, a, 64, "sha256 hex digest length")

	// Different inputs differ; the separator prevents trivial concatenation collisions.
	assert.NotEqual(t, a, HashPayload("inpost", "EXT-1234", "pdf"))
	assert.NotEqual(t, HashPayload("ab", "c"), HashPayload("a", "bc"))
}

func TestClassifyCarrierError(t *testing.T) {
	cases := map[string]string{
		"rate limit exceeded":            model.CarrierFailRateLimit,
		"HTTP 429 Too Many Requests":     model.CarrierFailRateLimit,
		"401 unauthorized":               model.CarrierFailAuth,
		"invalid credentials provided":   model.CarrierFailAuth,
		"context deadline exceeded":      model.CarrierFailProviderOutage,
		"service unavailable (503)":      model.CarrierFailProviderOutage,
		"missing required field: street": model.CarrierFailMissingData,
		"validation failed":              model.CarrierFailMissingData,
		"something weird happened":       model.CarrierFailProviderRejection, // default
	}
	for msg, want := range cases {
		assert.Equal(t, want, classifyCarrierError(errors.New(msg)), "msg %q", msg)
	}
	// nil error -> provider rejection (safe default, never panics).
	assert.Equal(t, model.CarrierFailProviderRejection, classifyCarrierError(nil))

	// Every classification maps to a valid, existing blocker code.
	for _, cls := range []string{
		model.CarrierFailRateLimit, model.CarrierFailAuth, model.CarrierFailProviderOutage,
		model.CarrierFailMissingData, model.CarrierFailProviderRejection,
	} {
		assert.True(t, model.IsValidBlockerCode(model.CarrierFailureBlockerCode(cls)), "class %q", cls)
	}
}

func TestStepKeyForOperation(t *testing.T) {
	assert.Equal(t, model.StepGenerateLabel, stepKeyForOperation(model.ProviderOpGenerateLabel))
	assert.Equal(t, model.StepGenerateLabel, stepKeyForOperation(model.ProviderOpDownloadLabel))
	assert.Equal(t, model.StepAwaitTracking, stepKeyForOperation(model.ProviderOpSyncTracking))
	assert.Equal(t, model.StepCreateShipment, stepKeyForOperation(model.ProviderOpCreateShipment))
	assert.Equal(t, model.StepCreateShipment, stepKeyForOperation("unknown_op"))
}
