package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupplierCapabilitySubInterfaces(t *testing.T) {
	// A type implementing both sub-interfaces satisfies them; the bare SupplierProvider does not.
	var p any = &fakeFullSupplier{}
	_, isPreflighter := p.(SupplierPreflighter)
	_, isStatusReader := p.(SupplierStatusReader)
	assert.True(t, isPreflighter)
	assert.True(t, isStatusReader)

	// A bare provider (CreateOrder only) implements neither sub-interface.
	var bare any = &fakeBareSupplier{}
	_, bareIsPreflighter := bare.(SupplierPreflighter)
	_, bareIsStatusReader := bare.(SupplierStatusReader)
	assert.False(t, bareIsPreflighter)
	assert.False(t, bareIsStatusReader)
}

type fakeFullSupplier struct{}

func (f *fakeFullSupplier) ProviderName() string { return "fake" }
func (f *fakeFullSupplier) FetchProducts(_ context.Context) ([]SupplierProduct, error) {
	return nil, nil
}
func (f *fakeFullSupplier) FetchInventory(_ context.Context) ([]SupplierProduct, error) {
	return nil, nil
}
func (f *fakeFullSupplier) CreateOrder(_ context.Context, _ SupplierOrderRequest) (*SupplierOrderResult, error) {
	return &SupplierOrderResult{ExternalOrderID: "x"}, nil
}
func (f *fakeFullSupplier) Preflight(_ context.Context, _ SupplierOrderRequest) (*SupplierPreflightResult, error) {
	return &SupplierPreflightResult{Accepted: true}, nil
}
func (f *fakeFullSupplier) GetOrderStatus(_ context.Context, _ string) (*SupplierOrderStatus, error) {
	return &SupplierOrderStatus{RawStatus: "confirmed"}, nil
}

type fakeBareSupplier struct{}

func (f *fakeBareSupplier) ProviderName() string { return "bare" }
func (f *fakeBareSupplier) FetchProducts(_ context.Context) ([]SupplierProduct, error) {
	return nil, nil
}
func (f *fakeBareSupplier) FetchInventory(_ context.Context) ([]SupplierProduct, error) {
	return nil, nil
}
func (f *fakeBareSupplier) CreateOrder(_ context.Context, _ SupplierOrderRequest) (*SupplierOrderResult, error) {
	return &SupplierOrderResult{ExternalOrderID: "y"}, nil
}
