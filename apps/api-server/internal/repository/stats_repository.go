package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type StatsRepository struct{}

func NewStatsRepository() *StatsRepository {
	return &StatsRepository{}
}

func (r *StatsRepository) GetOrderCountByStatus(ctx context.Context, tx pgx.Tx) (map[string]int, error) {
	rows, err := tx.Query(ctx, `SELECT status, COUNT(*) FROM orders GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count by status: %w", err)
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetOrderCountBySource(ctx context.Context, tx pgx.Tx) (map[string]int, error) {
	rows, err := tx.Query(ctx, `SELECT source, COUNT(*) FROM orders GROUP BY source`)
	if err != nil {
		return nil, fmt.Errorf("count by source: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, fmt.Errorf("scan count by source: %w", err)
		}
		result[source] = count
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetTotalRevenue(ctx context.Context, tx pgx.Tx) (float64, error) {
	var total float64
	err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(total_amount), 0) FROM orders`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("total revenue: %w", err)
	}
	return total, nil
}

func (r *StatsRepository) GetDailyRevenue(ctx context.Context, tx pgx.Tx, days int) ([]model.DailyRevenue, error) {
	rows, err := tx.Query(ctx,
		`SELECT DATE(created_at) as date, SUM(total_amount) as amount, COUNT(*) as count
		 FROM orders
		 WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		 GROUP BY DATE(created_at)
		 ORDER BY date ASC`, days)
	if err != nil {
		return nil, fmt.Errorf("daily revenue: %w", err)
	}
	defer rows.Close()

	result := []model.DailyRevenue{}
	for rows.Next() {
		var dr model.DailyRevenue
		var date time.Time
		if err := rows.Scan(&date, &dr.Amount, &dr.Count); err != nil {
			return nil, fmt.Errorf("scan daily revenue: %w", err)
		}
		dr.Date = date.Format("2006-01-02")
		result = append(result, dr)
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetMostCommonCurrency(ctx context.Context, tx pgx.Tx) (string, error) {
	var currency *string
	err := tx.QueryRow(ctx,
		`SELECT currency FROM orders GROUP BY currency ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&currency)
	if err != nil {
		// No orders at all — return empty string, not an error
		return "", nil
	}
	if currency == nil {
		return "", nil
	}
	return *currency, nil
}

func (r *StatsRepository) GetRecentOrders(ctx context.Context, tx pgx.Tx, limit int) ([]model.OrderSummary, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, customer_name, status, source, total_amount, currency, created_at
		 FROM orders
		 ORDER BY created_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent orders: %w", err)
	}
	defer rows.Close()

	result := []model.OrderSummary{}
	for rows.Next() {
		var os model.OrderSummary
		var id uuid.UUID
		if err := rows.Scan(&id, &os.CustomerName, &os.Status, &os.Source, &os.TotalAmount, &os.Currency, &os.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent orders: %w", err)
		}
		os.ID = id.String()
		result = append(result, os)
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetTopProducts(ctx context.Context, tx pgx.Tx, limit int) ([]model.TopProduct, error) {
	rows, err := tx.Query(ctx,
		`WITH eligible_orders AS (
		     SELECT items FROM orders
		     WHERE items IS NOT NULL
		       AND items != 'null'::jsonb
		       AND jsonb_typeof(items) = 'array'
		       AND jsonb_array_length(items) > 0
		 )
		 SELECT COALESCE(i.name, 'Bez nazwy'),
		        SUM(COALESCE(i.quantity, 0))::int  AS total_quantity,
		        SUM(COALESCE(i.price, 0) * COALESCE(i.quantity, 0)) AS total_revenue
		 FROM eligible_orders,
		      jsonb_to_recordset(eligible_orders.items) AS i(name text, quantity int, price numeric)
		 WHERE i.name IS NOT NULL AND i.name != ''
		 GROUP BY i.name
		 ORDER BY total_revenue DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("top products: %w", err)
	}
	defer rows.Close()

	result := []model.TopProduct{}
	for rows.Next() {
		var tp model.TopProduct
		if err := rows.Scan(&tp.Name, &tp.TotalQuantity, &tp.TotalRevenue); err != nil {
			return nil, fmt.Errorf("scan top products: %w", err)
		}
		result = append(result, tp)
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetRevenueBySource(ctx context.Context, tx pgx.Tx, days int) ([]model.SourceRevenue, error) {
	rows, err := tx.Query(ctx,
		`SELECT source, SUM(total_amount) as revenue, COUNT(*) as count
		 FROM orders
		 WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		 GROUP BY source
		 ORDER BY revenue DESC`, days)
	if err != nil {
		return nil, fmt.Errorf("revenue by source: %w", err)
	}
	defer rows.Close()

	result := []model.SourceRevenue{}
	for rows.Next() {
		var sr model.SourceRevenue
		if err := rows.Scan(&sr.Source, &sr.Revenue, &sr.Count); err != nil {
			return nil, fmt.Errorf("scan revenue by source: %w", err)
		}
		result = append(result, sr)
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetOrderTrends(ctx context.Context, tx pgx.Tx, days int) ([]model.DailyOrderTrend, error) {
	rows, err := tx.Query(ctx,
		`SELECT DATE(created_at) as date, COUNT(*) as count, AVG(total_amount) as avg_value
		 FROM orders
		 WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		 GROUP BY DATE(created_at)
		 ORDER BY date ASC`, days)
	if err != nil {
		return nil, fmt.Errorf("order trends: %w", err)
	}
	defer rows.Close()

	result := []model.DailyOrderTrend{}
	for rows.Next() {
		var dt model.DailyOrderTrend
		var date time.Time
		if err := rows.Scan(&date, &dt.Count, &dt.AvgValue); err != nil {
			return nil, fmt.Errorf("scan order trends: %w", err)
		}
		dt.Date = date.Format("2006-01-02")
		result = append(result, dt)
	}
	return result, rows.Err()
}

func (r *StatsRepository) GetPaymentMethodStats(ctx context.Context, tx pgx.Tx) (map[string]int, error) {
	rows, err := tx.Query(ctx,
		`SELECT COALESCE(payment_method, 'unknown'), COUNT(*)
		 FROM orders
		 GROUP BY payment_method`)
	if err != nil {
		return nil, fmt.Errorf("payment method stats: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var method string
		var count int
		if err := rows.Scan(&method, &count); err != nil {
			return nil, fmt.Errorf("scan payment method stats: %w", err)
		}
		result[method] = count
	}
	return result, rows.Err()
}
