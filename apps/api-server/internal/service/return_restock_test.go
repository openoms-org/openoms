package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// settingsTenantRepoStub serves a canned tenants.settings blob.
type settingsTenantRepoStub struct {
	settings json.RawMessage
	err      error
}

func (s settingsTenantRepoStub) FindBySlug(_ context.Context, _ string) (*model.Tenant, error) {
	return nil, nil
}

func (s settingsTenantRepoStub) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Tenant, error) {
	return nil, nil
}

func (s settingsTenantRepoStub) SlugExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (s settingsTenantRepoStub) Create(_ context.Context, _ pgx.Tx, _ *model.Tenant) error {
	return nil
}

func (s settingsTenantRepoStub) GetSettings(_ context.Context, _ pgx.Tx, _ uuid.UUID) (json.RawMessage, error) {
	return s.settings, s.err
}

func (s settingsTenantRepoStub) GetSettingsForUpdate(_ context.Context, _ pgx.Tx, _ uuid.UUID) (json.RawMessage, error) {
	return s.settings, s.err
}

func (s settingsTenantRepoStub) ListAllTenantIDs(_ context.Context, _ *pgxpool.Pool) ([]uuid.UUID, error) {
	return nil, nil
}

func (s settingsTenantRepoStub) UpdateSettings(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}

// noopRestocker satisfies StockRestocker without touching a database.
type noopRestocker struct {
	calls int
	items json.RawMessage
	err   error
}

func (n *noopRestocker) RestockItems(_ context.Context, _ uuid.UUID, items json.RawMessage) error {
	n.calls++
	n.items = items
	return n.err
}

// TestReturnRestockTriggerTypeIsAccepted guards the stock-sync trigger allowlist:
// StockSyncService.OnStockChange drops an unknown trigger type with a warning, which
// would silently skip pushing the restocked availability to marketplaces.
func TestReturnRestockTriggerTypeIsAccepted(t *testing.T) {
	assert.True(t, model.IsValidTriggerType("return_restocked"))
}

// TestReturnSettings_RestockStatus covers the policy resolution: the default point of
// restock, the disable words, and a custom status.
func TestReturnSettings_RestockStatus(t *testing.T) {
	assert.Equal(t, model.ReturnRestockDefaultStatus, model.ReturnSettings{}.RestockStatus(),
		"unset policy restocks when the warehouse confirms receipt")
	assert.Equal(t, "received", model.ReturnSettings{RestockOn: " RECEIVED "}.RestockStatus())
	assert.Equal(t, "refunded", model.ReturnSettings{RestockOn: "refunded"}.RestockStatus())

	for _, off := range []string{"off", "none", "never", "disabled", "OFF"} {
		assert.Empty(t, model.ReturnSettings{RestockOn: off}.RestockStatus(), "%q disables restocking", off)
	}
}

// TestReturnService_RestockStatus_Policy verifies the tenant policy read: the default
// when nothing is configured, the configured status when it is, and "" when returns must
// not restock or the dependencies are unwired.
func TestReturnService_RestockStatus_Policy(t *testing.T) {
	newSvc := func(settings string) *ReturnService {
		svc := NewReturnService(nil, nil, nil, nil, nil)
		svc.SetRestockPolicy(settingsTenantRepoStub{settings: json.RawMessage(settings)}, &noopRestocker{})
		return svc
	}

	cases := []struct {
		name     string
		settings string
		want     string
	}{
		{"no settings at all", "", model.ReturnRestockDefaultStatus},
		{"settings without a returns block", `{"inventory":{"strict_mode":true}}`, model.ReturnRestockDefaultStatus},
		{"empty returns block", `{"returns":{}}`, model.ReturnRestockDefaultStatus},
		{"explicit status", `{"returns":{"restock_on":"refunded"}}`, "refunded"},
		{"disabled", `{"returns":{"restock_on":"off"}}`, ""},
		{"malformed returns block falls back to the default", `{"returns":"nonsense"}`, model.ReturnRestockDefaultStatus},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newSvc(tc.settings).restockStatus(context.Background(), nil, uuid.New())
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("unwired policy never restocks", func(t *testing.T) {
		got, err := NewReturnService(nil, nil, nil, nil, nil).restockStatus(context.Background(), nil, uuid.New())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("settings read failure surfaces", func(t *testing.T) {
		boom := errors.New("connection reset")
		svc := NewReturnService(nil, nil, nil, nil, nil)
		svc.SetRestockPolicy(settingsTenantRepoStub{err: boom}, &noopRestocker{})
		_, err := svc.restockStatus(context.Background(), nil, uuid.New())
		assert.ErrorIs(t, err, boom)
	})
}
