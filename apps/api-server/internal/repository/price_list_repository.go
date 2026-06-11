package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PriceListRepository handles persistence for price lists and their items.
type PriceListRepository struct{}

// NewPriceListRepository creates a new PriceListRepository.
func NewPriceListRepository() *PriceListRepository {
	return &PriceListRepository{}
}

var priceListColumns = `id, tenant_id, name, description, currency, is_default, discount_type, active, valid_from, valid_to, created_at, updated_at`

func scanPriceList(row interface{ Scan(dest ...any) error }) (*model.PriceList, error) {
	var pl model.PriceList
	err := row.Scan(
		&pl.ID, &pl.TenantID, &pl.Name, &pl.Description, &pl.Currency,
		&pl.IsDefault, &pl.DiscountType, &pl.Active, &pl.ValidFrom, &pl.ValidTo,
		&pl.CreatedAt, &pl.UpdatedAt,
	)
	return &pl, err
}

// List returns a paginated list of price lists matching the filter.
func (r *PriceListRepository) List(ctx context.Context, tx pgx.Tx, filter model.PriceListListFilter) ([]model.PriceList, int, error) {
	qb := NewQueryBuilder()
	if filter.Active != nil {
		qb.Add("active = $%d", *filter.Active)
	}
	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM price_lists %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count price lists: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s FROM price_lists %s %s LIMIT $%d OFFSET $%d`,
		priceListColumns, where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list price lists: %w", err)
	}
	defer rows.Close()

	var priceLists []model.PriceList
	for rows.Next() {
		pl, err := scanPriceList(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan price list: %w", err)
		}
		priceLists = append(priceLists, *pl)
	}
	return priceLists, total, rows.Err()
}

// FindByID returns a price list by its ID.
func (r *PriceListRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PriceList, error) {
	pl, err := scanPriceList(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM price_lists WHERE id = $1", priceListColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find price list by id: %w", err)
	}
	return pl, nil
}

// Create inserts a new price list.
func (r *PriceListRepository) Create(ctx context.Context, tx pgx.Tx, pl *model.PriceList) error {
	return tx.QueryRow(ctx,
		`INSERT INTO price_lists (id, tenant_id, name, description, currency, is_default, discount_type, active, valid_from, valid_to)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		pl.ID, pl.TenantID, pl.Name, pl.Description, pl.Currency,
		pl.IsDefault, pl.DiscountType, pl.Active, pl.ValidFrom, pl.ValidTo,
	).Scan(&pl.CreatedAt, &pl.UpdatedAt)
}

// Update applies partial updates to a price list.
func (r *PriceListRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdatePriceListRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "description", req.Description)
	SetPtr(ub, "currency", req.Currency)
	SetPtr(ub, "is_default", req.IsDefault)
	SetPtr(ub, "discount_type", req.DiscountType)
	SetPtr(ub, "active", req.Active)
	SetPtr(ub, "valid_from", req.ValidFrom)
	SetPtr(ub, "valid_to", req.ValidTo)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE price_lists SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update price list: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("price list not found")
	}
	return nil
}

// Delete removes a price list by its ID.
func (r *PriceListRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM price_lists WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete price list: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("price list not found")
	}
	return nil
}

// --- Price List Items ---

var priceListItemColumns = `id, tenant_id, price_list_id, product_id, variant_id, price, discount, min_quantity, created_at, updated_at`

func scanPriceListItem(row interface{ Scan(dest ...any) error }) (*model.PriceListItem, error) {
	var item model.PriceListItem
	err := row.Scan(
		&item.ID, &item.TenantID, &item.PriceListID, &item.ProductID,
		&item.VariantID, &item.Price, &item.Discount, &item.MinQuantity,
		&item.CreatedAt, &item.UpdatedAt,
	)
	return &item, err
}

// ListItems returns paginated items belonging to a price list.
func (r *PriceListRepository) ListItems(ctx context.Context, tx pgx.Tx, priceListID uuid.UUID, limit, offset int) ([]model.PriceListItem, int, error) {
	var total int
	if err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM price_list_items WHERE price_list_id = $1", priceListID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count price list items: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM price_list_items WHERE price_list_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		priceListItemColumns,
	)
	rows, err := tx.Query(ctx, query, priceListID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list price list items: %w", err)
	}
	defer rows.Close()

	var items []model.PriceListItem
	for rows.Next() {
		item, err := scanPriceListItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan price list item: %w", err)
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

// CreateItem inserts a new price list item.
func (r *PriceListRepository) CreateItem(ctx context.Context, tx pgx.Tx, item *model.PriceListItem) error {
	return tx.QueryRow(ctx,
		`INSERT INTO price_list_items (id, tenant_id, price_list_id, product_id, variant_id, price, discount, min_quantity)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, updated_at`,
		item.ID, item.TenantID, item.PriceListID, item.ProductID,
		item.VariantID, item.Price, item.Discount, item.MinQuantity,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

// DeleteItem removes a price list item by its ID.
func (r *PriceListRepository) DeleteItem(ctx context.Context, tx pgx.Tx, itemID uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM price_list_items WHERE id = $1", itemID)
	if err != nil {
		return fmt.Errorf("delete price list item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("price list item not found")
	}
	return nil
}

// FindItemsByProduct returns price list items matching product, variant, and quantity tier.
func (r *PriceListRepository) FindItemsByProduct(ctx context.Context, tx pgx.Tx, priceListID, productID uuid.UUID, variantID *uuid.UUID, quantity int) ([]model.PriceListItem, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM price_list_items
		 WHERE price_list_id = $1 AND product_id = $2
		   AND (variant_id IS NULL OR variant_id = $3)
		   AND min_quantity <= $4
		 ORDER BY min_quantity DESC`,
		priceListItemColumns,
	)
	rows, err := tx.Query(ctx, query, priceListID, productID, variantID, quantity)
	if err != nil {
		return nil, fmt.Errorf("find price list items by product: %w", err)
	}
	defer rows.Close()

	var items []model.PriceListItem
	for rows.Next() {
		item, err := scanPriceListItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan price list item: %w", err)
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
