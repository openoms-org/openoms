package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- Mocks for BL product import tests ---

// mockProductRepo implements repository.ProductRepo for unit tests.
type mockProductRepo struct {
	products  map[string]*model.Product // keyed by SKU
	created   []*model.Product
	updated   []mockProductUpdateCall
	byID      map[uuid.UUID]*model.Product
	createErr error
	updateErr error

	// canonical available stock, plus a record of the lookups made against it
	availableStock      map[uuid.UUID]int
	availableStockErr   error
	availableStockCalls [][]uuid.UUID
}

type mockProductUpdateCall struct {
	ID  uuid.UUID
	Req model.UpdateProductRequest
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products: make(map[string]*model.Product),
		byID:     make(map[uuid.UUID]*model.Product),
	}
}

func (m *mockProductRepo) List(_ context.Context, _ pgx.Tx, _ model.ProductListFilter) ([]model.Product, int, error) {
	return nil, 0, nil
}

func (m *mockProductRepo) FindByID(_ context.Context, _ pgx.Tx, id uuid.UUID) (*model.Product, error) {
	if p, ok := m.byID[id]; ok {
		return p, nil
	}
	return nil, nil
}

func (m *mockProductRepo) FindByIDs(_ context.Context, _ pgx.Tx, _ []uuid.UUID) ([]model.Product, error) {
	return nil, nil
}

func (m *mockProductRepo) FindBySKU(_ context.Context, _ pgx.Tx, sku string) (*model.Product, error) {
	if p, ok := m.products[sku]; ok {
		return p, nil
	}
	return nil, nil
}

func (m *mockProductRepo) FindByEAN(_ context.Context, _ pgx.Tx, _ string) (*model.Product, error) {
	return nil, nil
}
func (m *mockProductRepo) FindIDsByEANs(_ context.Context, _ pgx.Tx, _ []string) (map[string]uuid.UUID, error) {
	return map[string]uuid.UUID{}, nil
}

func (m *mockProductRepo) AvailableStockBatch(_ context.Context, _ pgx.Tx, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	m.availableStockCalls = append(m.availableStockCalls, ids)
	if m.availableStockErr != nil {
		return nil, m.availableStockErr
	}
	return m.availableStock, nil
}

func (m *mockProductRepo) Create(_ context.Context, _ pgx.Tx, product *model.Product) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, product)
	if product.SKU != nil {
		m.products[*product.SKU] = product
	}
	m.byID[product.ID] = product
	return nil
}

func (m *mockProductRepo) Update(_ context.Context, _ pgx.Tx, id uuid.UUID, req model.UpdateProductRequest) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, mockProductUpdateCall{ID: id, Req: req})
	return nil
}

func (m *mockProductRepo) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error {
	return nil
}

// mockVariantRepo implements repository.VariantRepo for unit tests.
type mockVariantRepo struct {
	created []*model.ProductVariant
}

func newMockVariantRepo() *mockVariantRepo {
	return &mockVariantRepo{}
}

func (m *mockVariantRepo) List(_ context.Context, _ pgx.Tx, _ model.VariantListFilter) ([]model.ProductVariant, int, error) {
	return nil, 0, nil
}

func (m *mockVariantRepo) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.ProductVariant, error) {
	return nil, nil
}

func (m *mockVariantRepo) FindBySKU(_ context.Context, _ pgx.Tx, _ string) ([]model.ProductVariant, error) {
	return nil, nil
}

func (m *mockVariantRepo) FindByEAN(_ context.Context, _ pgx.Tx, _ string) ([]model.ProductVariant, error) {
	return nil, nil
}

func (m *mockVariantRepo) Create(_ context.Context, _ pgx.Tx, variant *model.ProductVariant) error {
	m.created = append(m.created, variant)
	return nil
}

func (m *mockVariantRepo) Update(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.UpdateVariantRequest) error {
	return nil
}

func (m *mockVariantRepo) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error {
	return nil
}

func (m *mockVariantRepo) CountByProductID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (int, error) {
	return 0, nil
}

// mockCategoryRepo implements repository.ProductCategoryRepo for unit tests.
type mockCategoryRepo struct {
	categories []model.ProductCategory
	created    []*model.ProductCategory
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{}
}

func (m *mockCategoryRepo) List(_ context.Context, _ pgx.Tx, _ model.CategoryListFilter) ([]model.ProductCategory, error) {
	return m.categories, nil
}

func (m *mockCategoryRepo) FindByID(_ context.Context, _ pgx.Tx, id uuid.UUID) (*model.ProductCategory, error) {
	for i := range m.categories {
		if m.categories[i].ID == id {
			return &m.categories[i], nil
		}
	}
	return nil, nil
}

func (m *mockCategoryRepo) FindBySlug(_ context.Context, _ pgx.Tx, slug string) (*model.ProductCategory, error) {
	for i := range m.categories {
		if m.categories[i].Slug == slug {
			return &m.categories[i], nil
		}
	}
	return nil, nil
}

func (m *mockCategoryRepo) Create(_ context.Context, _ pgx.Tx, c *model.ProductCategory) error {
	m.created = append(m.created, c)
	m.categories = append(m.categories, *c)
	return nil
}

func (m *mockCategoryRepo) Update(_ context.Context, _ pgx.Tx, _ *model.ProductCategory) error {
	return nil
}

func (m *mockCategoryRepo) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error {
	return nil
}

func (m *mockCategoryRepo) GetDescendantIDs(_ context.Context, _ pgx.Tx, id uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{id}, nil
}

func (m *mockCategoryRepo) FuzzyMatch(_ context.Context, _ pgx.Tx, name string) ([]model.ProductCategory, error) {
	var matches []model.ProductCategory
	for _, c := range m.categories {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			matches = append(matches, c)
		}
	}
	return matches, nil
}

func (m *mockCategoryRepo) CountBySlug(_ context.Context, _ pgx.Tx, _ string) (int, error) {
	return 0, nil
}

// --- Helper to build CSV ---

func buildBLProductCSV(headers []string, rows [][]string) []byte {
	var b strings.Builder
	b.WriteString(strings.Join(headers, ","))
	b.WriteByte('\n')
	for _, row := range rows {
		for i, field := range row {
			if i > 0 {
				b.WriteByte(',')
			}
			// Quote fields that contain commas or semicolons.
			if strings.ContainsAny(field, ",;\n\"") {
				b.WriteByte('"')
				b.WriteString(strings.ReplaceAll(field, "\"", "\"\""))
				b.WriteByte('"')
			} else {
				b.WriteString(field)
			}
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// --- Tests ---

func TestBLProductImport_Aliases(t *testing.T) {
	// BL-specific column names should be auto-detected and mapped.
	headers := []string{"product_id", "product_name", "product_sku", "product_ean", "product_price_brutto", "product_quantity", "category_name", "image_url"}
	rows := [][]string{
		{"BL-100", "Widget A", "SKU-001", "5901234123457", "49.99", "100", "Electronics", "http://img.com/1.jpg"},
	}
	csvData := buildBLProductCSV(headers, rows)

	_, _, canonIdx, mappings, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	// Verify canonical index has the expected fields.
	assert.Contains(t, canonIdx, "external_id")
	assert.Contains(t, canonIdx, "name")
	assert.Contains(t, canonIdx, "sku")
	assert.Contains(t, canonIdx, "ean")
	assert.Contains(t, canonIdx, "price")
	assert.Contains(t, canonIdx, "stock_quantity")
	assert.Contains(t, canonIdx, "category")
	assert.Contains(t, canonIdx, "images")

	// Verify values resolve correctly.
	row := []string{"BL-100", "Widget A", "SKU-001", "5901234123457", "49.99", "100", "Electronics", "http://img.com/1.jpg"}
	assert.Equal(t, "BL-100", getCanonVal(row, canonIdx, "external_id"))
	assert.Equal(t, "Widget A", getCanonVal(row, canonIdx, "name"))
	assert.Equal(t, "SKU-001", getCanonVal(row, canonIdx, "sku"))
	assert.Equal(t, "5901234123457", getCanonVal(row, canonIdx, "ean"))
	assert.Equal(t, "49.99", getCanonVal(row, canonIdx, "price"))
	assert.Equal(t, "100", getCanonVal(row, canonIdx, "stock_quantity"))
	assert.Equal(t, "Electronics", getCanonVal(row, canonIdx, "category"))
	assert.Equal(t, "http://img.com/1.jpg", getCanonVal(row, canonIdx, "images"))

	// Verify mappings are populated.
	assert.NotEmpty(t, mappings)
	foundName := false
	for _, m := range mappings {
		if m.CSVColumn == "product_name" && m.OrderField == "name" {
			foundName = true
		}
	}
	assert.True(t, foundName, "expected mapping for product_name -> name")
}

func TestBLProductImport_VariantGrouping(t *testing.T) {
	// Rows with variant_id are classified as variants; rows without are parents.
	headers := []string{"product_id", "name", "sku", "variant_id", "variant_name", "price_brutto", "quantity"}
	rows := [][]string{
		{"100", "T-Shirt", "TSHIRT-MAIN", "", "", "59.99", "50"},      // parent
		{"100", "T-Shirt", "TSHIRT-S", "V1", "Size S", "59.99", "20"}, // variant
		{"100", "T-Shirt", "TSHIRT-M", "V2", "Size M", "59.99", "15"}, // variant
		{"100", "T-Shirt", "TSHIRT-L", "V3", "Size L", "59.99", "15"}, // variant
		{"200", "Hoodie", "HOODIE-MAIN", "", "", "99.99", "30"},       // parent
		{"200", "Hoodie", "HOODIE-RED", "V4", "Red", "99.99", "10"},   // variant
	}
	csvData := buildBLProductCSV(headers, rows)

	_, records, canonIdx, _, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	parentRows, variantRows, parentOrder := classifyBLRows(records, canonIdx)

	assert.Equal(t, 2, len(parentRows))
	assert.Equal(t, 4, len(variantRows))
	assert.Equal(t, []string{"100", "200"}, parentOrder)

	// Verify parent rows have no variant_id.
	for _, pr := range parentRows {
		assert.Empty(t, pr.VariantID)
	}

	// Verify variant rows have variant_id.
	for _, vr := range variantRows {
		assert.NotEmpty(t, vr.VariantID)
	}

	// Check variant external_id matches parent.
	assert.Equal(t, "100", variantRows[0].ExternalID)
	assert.Equal(t, "100", variantRows[1].ExternalID)
	assert.Equal(t, "100", variantRows[2].ExternalID)
	assert.Equal(t, "200", variantRows[3].ExternalID)
}

func TestBLProductImport_ImageURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single URL",
			input:    "http://example.com/img1.jpg",
			expected: []string{"http://example.com/img1.jpg"},
		},
		{
			name:     "semicolon-separated URLs",
			input:    "http://example.com/1.jpg;http://example.com/2.jpg;http://example.com/3.jpg",
			expected: []string{"http://example.com/1.jpg", "http://example.com/2.jpg", "http://example.com/3.jpg"},
		},
		{
			name:     "URLs with spaces around semicolons",
			input:    "http://example.com/1.jpg ; http://example.com/2.jpg ; http://example.com/3.jpg",
			expected: []string{"http://example.com/1.jpg", "http://example.com/2.jpg", "http://example.com/3.jpg"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "trailing semicolons",
			input:    "http://example.com/1.jpg;http://example.com/2.jpg;;",
			expected: []string{"http://example.com/1.jpg", "http://example.com/2.jpg"},
		},
		{
			name:     "whitespace only",
			input:    "  ;  ;  ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBLImages(tt.input)
			require.NotNil(t, result)

			var urls []string
			err := json.Unmarshal(result, &urls)
			require.NoError(t, err)

			if tt.expected == nil {
				assert.Empty(t, urls)
			} else {
				assert.Equal(t, tt.expected, urls)
			}
		})
	}
}

func TestBLProductImport_CategoryCreation(t *testing.T) {
	categoryRepo := newMockCategoryRepo()
	result := &BLProductImportResult{Errors: []model.ImportError{}}
	cache := make(map[string]*model.ProductCategory)
	tenantID := uuid.New()

	svc := &BaseLinkerProductImportService{
		categoryRepo: categoryRepo,
	}

	// First call: creates new category.
	cat1, err := svc.resolveCategory(context.Background(), nil, tenantID, "Electronics", cache, result)
	require.NoError(t, err)
	require.NotNil(t, cat1)
	assert.Equal(t, "Electronics", cat1.Name)
	assert.Equal(t, 1, result.CategoriesCreated)
	assert.Equal(t, 1, len(categoryRepo.created))

	// Second call: uses cache.
	cat2, err := svc.resolveCategory(context.Background(), nil, tenantID, "Electronics", cache, result)
	require.NoError(t, err)
	require.NotNil(t, cat2)
	assert.Equal(t, cat1.ID, cat2.ID)
	assert.Equal(t, 1, result.CategoriesCreated) // Still 1, not 2.
	assert.Equal(t, 1, len(categoryRepo.created))
}

func TestBLProductImport_CategoryExistingMatch(t *testing.T) {
	categoryRepo := newMockCategoryRepo()
	existingCat := model.ProductCategory{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Electronics",
		Slug:     "electronics",
	}
	categoryRepo.categories = append(categoryRepo.categories, existingCat)

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	cache := make(map[string]*model.ProductCategory)

	svc := &BaseLinkerProductImportService{
		categoryRepo: categoryRepo,
	}

	cat, err := svc.resolveCategory(context.Background(), nil, uuid.New(), "Electronics", cache, result)
	require.NoError(t, err)
	require.NotNil(t, cat)
	assert.Equal(t, existingCat.ID, cat.ID)
	assert.Equal(t, 0, result.CategoriesCreated) // No new category created.
}

func TestBLProductImport_ImportProduct_CreatesNew(t *testing.T) {
	productRepo := newMockProductRepo()
	variantRepo := newMockVariantRepo()
	categoryRepo := newMockCategoryRepo()
	auditRepo := &mockAuditRepo{}

	svc := &BaseLinkerProductImportService{
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
	}

	tenantID := uuid.New()
	headers := []string{"product_id", "product_name", "product_sku", "product_ean", "product_price_brutto", "product_quantity", "category_name", "description", "description_extra", "weight", "image_url"}
	rows := [][]string{
		{"BL-100", "Widget A", "WID-001", "5901234123457", "49.99", "100", "Widgets", "Short desc", "Long description", "0.5", "http://img.com/1.jpg;http://img.com/2.jpg"},
	}
	csvData := buildBLProductCSV(headers, rows)

	_, records, canonIdx, _, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	parentRows, _, _ := classifyBLRows(records, canonIdx)
	require.Len(t, parentRows, 1)

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	externalIDsWithVariants := make(map[string]bool)
	categoryCache := make(map[string]*model.ProductCategory)

	productID, importErr := svc.importParentProduct(context.Background(), nil, tenantID, parentRows[0], canonIdx, result, externalIDsWithVariants, categoryCache)
	assert.Nil(t, importErr)
	assert.NotEqual(t, uuid.Nil, productID)
	assert.Equal(t, 1, result.ProductsCreated)

	require.Len(t, productRepo.created, 1)
	p := productRepo.created[0]

	assert.Equal(t, "baselinker", p.Source)
	assert.Equal(t, "Widget A", p.Name)
	require.NotNil(t, p.SKU)
	assert.Equal(t, "WID-001", *p.SKU)
	require.NotNil(t, p.EAN)
	assert.Equal(t, "5901234123457", *p.EAN)
	assert.InDelta(t, 49.99, p.Price, 0.001)
	assert.Equal(t, 100, p.StockQuantity)
	require.NotNil(t, p.ExternalID)
	assert.Equal(t, "BL-100", *p.ExternalID)
	assert.Equal(t, "Short desc", p.DescriptionShort)
	assert.Equal(t, "Long description", p.DescriptionLong)
	require.NotNil(t, p.Weight)
	assert.InDelta(t, 0.5, *p.Weight, 0.001)
	assert.False(t, p.HasVariants)

	// Verify images parsed from semicolons.
	var images []string
	err = json.Unmarshal(p.Images, &images)
	require.NoError(t, err)
	assert.Equal(t, []string{"http://img.com/1.jpg", "http://img.com/2.jpg"}, images)

	// Verify category was created.
	assert.Equal(t, 1, result.CategoriesCreated)
	require.NotNil(t, p.Category)
	assert.Equal(t, "Widgets", *p.Category)
	require.NotNil(t, p.CategoryID)
}

func TestBLProductImport_ImportProduct_UpdatesExisting(t *testing.T) {
	existingProduct := &model.Product{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Old Widget",
		Source:   "manual",
	}
	existingSKU := "WID-001"
	existingProduct.SKU = &existingSKU

	productRepo := newMockProductRepo()
	productRepo.products[existingSKU] = existingProduct
	productRepo.byID[existingProduct.ID] = existingProduct

	variantRepo := newMockVariantRepo()
	categoryRepo := newMockCategoryRepo()
	auditRepo := &mockAuditRepo{}

	svc := &BaseLinkerProductImportService{
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
	}

	tenantID := uuid.New()
	headers := []string{"product_id", "product_name", "product_sku", "product_price_brutto", "product_quantity"}
	rows := [][]string{
		{"BL-100", "Updated Widget", "WID-001", "59.99", "200"},
	}
	csvData := buildBLProductCSV(headers, rows)

	_, records, canonIdx, _, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	parentRows, _, _ := classifyBLRows(records, canonIdx)
	require.Len(t, parentRows, 1)

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	externalIDsWithVariants := make(map[string]bool)
	categoryCache := make(map[string]*model.ProductCategory)

	productID, importErr := svc.importParentProduct(context.Background(), nil, tenantID, parentRows[0], canonIdx, result, externalIDsWithVariants, categoryCache)
	assert.Nil(t, importErr)
	assert.Equal(t, existingProduct.ID, productID)
	assert.Equal(t, 0, result.ProductsCreated)
	assert.Equal(t, 1, result.ProductsUpdated)

	// Verify update was called.
	require.Len(t, productRepo.updated, 1)
	updateReq := productRepo.updated[0].Req
	require.NotNil(t, updateReq.Name)
	assert.Equal(t, "Updated Widget", *updateReq.Name)
	require.NotNil(t, updateReq.Price)
	assert.InDelta(t, 59.99, *updateReq.Price, 0.001)
	require.NotNil(t, updateReq.StockQuantity)
	assert.Equal(t, 200, *updateReq.StockQuantity)
	require.NotNil(t, updateReq.Source)
	assert.Equal(t, "baselinker", *updateReq.Source)
	require.NotNil(t, updateReq.ExternalID)
	assert.Equal(t, "BL-100", *updateReq.ExternalID)
}

func TestBLProductImport_ImportProduct_WithVariants(t *testing.T) {
	productRepo := newMockProductRepo()
	variantRepo := newMockVariantRepo()
	categoryRepo := newMockCategoryRepo()
	auditRepo := &mockAuditRepo{}

	svc := &BaseLinkerProductImportService{
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
	}

	tenantID := uuid.New()
	headers := []string{"product_id", "name", "sku", "variant_id", "variant_name", "price_brutto", "quantity", "ean"}
	rows := [][]string{
		{"100", "T-Shirt", "TSHIRT", "", "", "59.99", "50", ""},
		{"100", "T-Shirt", "TSHIRT-S", "V1", "Size S", "59.99", "20", "1234567890123"},
		{"100", "T-Shirt", "TSHIRT-M", "V2", "Size M", "59.99", "15", "1234567890456"},
		{"100", "T-Shirt", "TSHIRT-L", "V3", "", "69.99", "15", ""},
	}
	csvData := buildBLProductCSV(headers, rows)

	_, records, canonIdx, _, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	parentRows, variantRows, _ := classifyBLRows(records, canonIdx)

	// Map which external_ids have variants.
	externalIDsWithVariants := make(map[string]bool)
	for _, vr := range variantRows {
		if vr.ExternalID != "" {
			externalIDsWithVariants[vr.ExternalID] = true
		}
	}

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	productByExternalID := make(map[string]uuid.UUID)
	categoryCache := make(map[string]*model.ProductCategory)

	// Phase 1: import parent.
	require.Len(t, parentRows, 1)
	productID, importErr := svc.importParentProduct(context.Background(), nil, tenantID, parentRows[0], canonIdx, result, externalIDsWithVariants, categoryCache)
	assert.Nil(t, importErr)
	assert.NotEqual(t, uuid.Nil, productID)
	productByExternalID["100"] = productID

	// Verify parent has has_variants = true.
	require.Len(t, productRepo.created, 1)
	assert.True(t, productRepo.created[0].HasVariants)

	// Phase 2: import variants.
	variantCounter := make(map[string]int)
	for _, vr := range variantRows {
		variantErr := svc.importVariantRow(context.Background(), nil, tenantID, vr, canonIdx, result, productByExternalID, variantCounter)
		assert.Nil(t, variantErr)
	}

	assert.Equal(t, 3, result.VariantsCreated)
	require.Len(t, variantRepo.created, 3)

	// First variant: Size S with explicit name.
	v0 := variantRepo.created[0]
	assert.Equal(t, "Size S", v0.Name)
	assert.Equal(t, productID, v0.ProductID)
	require.NotNil(t, v0.SKU)
	assert.Equal(t, "TSHIRT-S", *v0.SKU)
	require.NotNil(t, v0.EAN)
	assert.Equal(t, "1234567890123", *v0.EAN)
	require.NotNil(t, v0.PriceOverride)
	assert.InDelta(t, 59.99, *v0.PriceOverride, 0.001)
	assert.Equal(t, 20, v0.StockQuantity)
	assert.True(t, v0.Active)

	// Second variant: Size M.
	v1 := variantRepo.created[1]
	assert.Equal(t, "Size M", v1.Name)
	require.NotNil(t, v1.EAN)
	assert.Equal(t, "1234567890456", *v1.EAN)
	assert.Equal(t, 15, v1.StockQuantity)

	// Third variant: auto-generated name.
	v2 := variantRepo.created[2]
	assert.Equal(t, "Wariant 1", v2.Name) // Auto-generated since variant_name was empty.
	require.NotNil(t, v2.PriceOverride)
	assert.InDelta(t, 69.99, *v2.PriceOverride, 0.001)
	assert.Equal(t, 15, v2.StockQuantity)
}

func TestBLProductImport_VariantMissingParent(t *testing.T) {
	svc := &BaseLinkerProductImportService{}

	variantRow := &blProductRow{
		ExternalID: "UNKNOWN",
		VariantID:  "V1",
		Row:        []string{"UNKNOWN", "Variant", "SKU-V1", "V1", "Variant Name", "10.00", "5"},
		RowIndex:   2,
	}

	canonIdx := map[string]int{
		"external_id": 0, "name": 1, "sku": 2,
		"variant_id": 3, "variant_name": 4, "price": 5, "stock_quantity": 6,
	}

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	productByExternalID := make(map[string]uuid.UUID)
	variantCounter := make(map[string]int)

	importErr := svc.importVariantRow(context.Background(), nil, uuid.New(), variantRow, canonIdx, result, productByExternalID, variantCounter)
	require.NotNil(t, importErr)
	assert.Contains(t, importErr.Message, "parent product not found")
}

func TestBLProductImport_StandardHeaders(t *testing.T) {
	// Standard (non-BL) column names should also work where they match aliases.
	headers := []string{"name", "sku", "ean", "category", "weight", "images"}
	rows := [][]string{
		{"Product A", "SKU-A", "1234567890123", "Electronics", "1.5", "http://img.com/a.jpg;http://img.com/b.jpg"},
	}
	csvData := buildBLProductCSV(headers, rows)

	_, _, canonIdx, _, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	assert.Contains(t, canonIdx, "name")
	assert.Contains(t, canonIdx, "sku")
	assert.Contains(t, canonIdx, "ean")
	assert.Contains(t, canonIdx, "category")
	assert.Contains(t, canonIdx, "weight")
	assert.Contains(t, canonIdx, "images")
}

func TestBLProductImport_EuropeanDecimals(t *testing.T) {
	productRepo := newMockProductRepo()
	variantRepo := newMockVariantRepo()
	categoryRepo := newMockCategoryRepo()
	auditRepo := &mockAuditRepo{}

	svc := &BaseLinkerProductImportService{
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
	}

	tenantID := uuid.New()
	// Use quoted values to handle commas in CSV properly.
	csvData := []byte("product_id,product_name,product_sku,product_price_brutto,product_quantity,weight\nBL-1,Widget,WID-1,\"49,99\",100,\"1,5\"\n")

	_, records, canonIdx, _, err := parseBLProductCSV(csvData)
	require.NoError(t, err)

	parentRows, _, _ := classifyBLRows(records, canonIdx)
	require.Len(t, parentRows, 1)

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	externalIDsWithVariants := make(map[string]bool)
	categoryCache := make(map[string]*model.ProductCategory)

	productID, importErr := svc.importParentProduct(context.Background(), nil, tenantID, parentRows[0], canonIdx, result, externalIDsWithVariants, categoryCache)
	assert.Nil(t, importErr)
	assert.NotEqual(t, uuid.Nil, productID)

	require.Len(t, productRepo.created, 1)
	p := productRepo.created[0]
	assert.InDelta(t, 49.99, p.Price, 0.001)
	require.NotNil(t, p.Weight)
	assert.InDelta(t, 1.5, *p.Weight, 0.001)
}

func TestBLProductImport_EmptyNameError(t *testing.T) {
	svc := &BaseLinkerProductImportService{}

	parentRow := &blProductRow{
		ExternalID: "BL-1",
		Row:        []string{"BL-1", "", "SKU-1"},
		RowIndex:   1,
	}
	canonIdx := map[string]int{"external_id": 0, "name": 1, "sku": 2}

	result := &BLProductImportResult{Errors: []model.ImportError{}}
	externalIDsWithVariants := make(map[string]bool)
	categoryCache := make(map[string]*model.ProductCategory)

	_, importErr := svc.importParentProduct(context.Background(), nil, uuid.New(), parentRow, canonIdx, result, externalIDsWithVariants, categoryCache)
	require.NotNil(t, importErr)
	assert.Equal(t, "name", importErr.Field)
	assert.Contains(t, importErr.Message, "product name is required")
}
