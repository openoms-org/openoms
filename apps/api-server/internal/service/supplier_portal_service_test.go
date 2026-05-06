package service

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

func TestBuildSupplierPortalLinkUsesFragmentToken(t *testing.T) {
	link := buildSupplierPortalLink("https://app.openoms.org", "abc123")

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	fragmentValues, err := url.ParseQuery(parsed.Fragment)
	require.NoError(t, err)

	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "app.openoms.org", parsed.Host)
	assert.Equal(t, "/supplier-portal", parsed.Path)
	assert.Empty(t, parsed.RawQuery)
	assert.Equal(t, "abc123", fragmentValues.Get("token"))
}

func TestBuildSupplierPortalLinkTrimsTrailingSlash(t *testing.T) {
	link := buildSupplierPortalLink("https://app.openoms.org/", "abc123")

	assert.Equal(t, "https://app.openoms.org/supplier-portal#token=abc123", link)
}

var _ repository.PurchaseOrderRepo = (*fakeSupplierPortalPurchaseOrderRepo)(nil)

type fakeSupplierPortalPurchaseOrderRepo struct {
	po    *model.PurchaseOrder
	err   error
	gotID uuid.UUID
}

func (f *fakeSupplierPortalPurchaseOrderRepo) List(context.Context, pgx.Tx, model.PurchaseOrderListFilter) ([]model.PurchaseOrder, int, error) {
	return nil, 0, nil
}

func (f *fakeSupplierPortalPurchaseOrderRepo) FindByID(_ context.Context, _ pgx.Tx, id uuid.UUID) (*model.PurchaseOrder, error) {
	f.gotID = id
	return f.po, f.err
}

func (f *fakeSupplierPortalPurchaseOrderRepo) Create(context.Context, pgx.Tx, *model.PurchaseOrder) error {
	return nil
}

func (f *fakeSupplierPortalPurchaseOrderRepo) Update(context.Context, pgx.Tx, uuid.UUID, model.UpdatePurchaseOrderRequest) error {
	return nil
}

func (f *fakeSupplierPortalPurchaseOrderRepo) UpdateStatus(context.Context, pgx.Tx, uuid.UUID, string) error {
	return nil
}

func (f *fakeSupplierPortalPurchaseOrderRepo) UpdateTotalAmount(context.Context, pgx.Tx, uuid.UUID, float64) error {
	return nil
}

func (f *fakeSupplierPortalPurchaseOrderRepo) Delete(context.Context, pgx.Tx, uuid.UUID) error {
	return nil
}

func (f *fakeSupplierPortalPurchaseOrderRepo) GeneratePONumber(context.Context, pgx.Tx, uuid.UUID) (string, error) {
	return "", nil
}

func TestSupplierPortalService_ListMessagesForSupplierRejectsDifferentSupplier(t *testing.T) {
	supplierID := uuid.New()
	otherSupplierID := uuid.New()
	poID := uuid.New()
	poRepo := &fakeSupplierPortalPurchaseOrderRepo{
		po: &model.PurchaseOrder{
			ID:         poID,
			SupplierID: &otherSupplierID,
			Status:     "sent",
		},
	}
	svc := &SupplierPortalService{poRepo: poRepo}

	messages, err := svc.listMessagesForSupplier(context.Background(), nil, supplierID, poID)

	require.ErrorIs(t, err, ErrPortalPONotOwned)
	assert.Nil(t, messages)
	assert.Equal(t, poID, poRepo.gotID)
}

func TestSupplierPortalService_ListMessagesForSupplierHidesDraftOrders(t *testing.T) {
	supplierID := uuid.New()
	poID := uuid.New()
	poRepo := &fakeSupplierPortalPurchaseOrderRepo{
		po: &model.PurchaseOrder{
			ID:         poID,
			SupplierID: &supplierID,
			Status:     "draft",
		},
	}
	svc := &SupplierPortalService{poRepo: poRepo}

	messages, err := svc.listMessagesForSupplier(context.Background(), nil, supplierID, poID)

	require.ErrorIs(t, err, ErrPortalPONotFound)
	assert.Nil(t, messages)
	assert.Equal(t, poID, poRepo.gotID)
}

func TestSupplierPortalService_ListMessagesForSupplierRejectsMissingOrder(t *testing.T) {
	poID := uuid.New()
	poRepo := &fakeSupplierPortalPurchaseOrderRepo{}
	svc := &SupplierPortalService{poRepo: poRepo}

	messages, err := svc.listMessagesForSupplier(context.Background(), nil, uuid.New(), poID)

	require.ErrorIs(t, err, ErrPortalPONotFound)
	assert.Nil(t, messages)
	assert.Equal(t, poID, poRepo.gotID)
}
