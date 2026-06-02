package service

import (
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
	"github.com/stretchr/testify/assert"
)

// fakeStorage satisfies storage.ObjectStorage via an embedded (nil) interface;
// its methods are never invoked in these tests.
type fakeStorage struct{ storage.ObjectStorage }

// storageAwareCarrier satisfies integration.CarrierProvider via an embedded
// (nil) interface and implements the optional SetStorage hook.
type storageAwareCarrier struct {
	integration.CarrierProvider
	got storage.ObjectStorage
}

func (c *storageAwareCarrier) SetStorage(s storage.ObjectStorage) { c.got = s }

// plainCarrier satisfies CarrierProvider but does NOT implement SetStorage.
type plainCarrier struct{ integration.CarrierProvider }

func TestInjectCarrierStorage_InjectsWhenSupported(t *testing.T) {
	c := &storageAwareCarrier{}
	store := &fakeStorage{}

	injectCarrierStorage(c, store)

	assert.True(t, c.got == storage.ObjectStorage(store),
		"storage must be injected into a carrier that supports SetStorage")
}

func TestInjectCarrierStorage_NoopWhenUnsupported(t *testing.T) {
	assert.NotPanics(t, func() {
		injectCarrierStorage(&plainCarrier{}, &fakeStorage{})
	}, "carriers without SetStorage must be left untouched")
}

func TestInjectCarrierStorage_NoopWhenStorageNil(t *testing.T) {
	c := &storageAwareCarrier{}

	injectCarrierStorage(c, nil)

	assert.Nil(t, c.got, "no configured storage -> nothing injected")
}
