package model

import "time"

// DashboardStats aggregates key metrics for the main dashboard.
type DashboardStats struct {
	OrderCounts  OrderCounts    `json:"order_counts"`
	Revenue      Revenue        `json:"revenue"`
	RecentOrders []OrderSummary `json:"recent_orders"`
}

// OrderCounts holds order count breakdowns for the dashboard.
type OrderCounts struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
	BySource map[string]int `json:"by_source"`
}

// Revenue holds revenue totals and daily breakdown for the dashboard.
type Revenue struct {
	Total    float64        `json:"total"`
	Currency string         `json:"currency"`
	Daily    []DailyRevenue `json:"daily"`
}

// DailyRevenue holds revenue and order count for a single day.
type DailyRevenue struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// OrderSummary is a lightweight order representation used in dashboard lists.
type OrderSummary struct {
	ID           string    `json:"id"`
	CustomerName string    `json:"customer_name"`
	Status       string    `json:"status"`
	Source       string    `json:"source"`
	TotalAmount  float64   `json:"total_amount"`
	Currency     string    `json:"currency"`
	CreatedAt    time.Time `json:"created_at"`
}

// TopProduct holds aggregated sales data for a product in the dashboard.
type TopProduct struct {
	Name          string  `json:"name"`
	SKU           string  `json:"sku,omitempty"`
	TotalQuantity int     `json:"total_quantity"`
	TotalRevenue  float64 `json:"total_revenue"`
}

// SourceRevenue holds revenue data broken down by sales channel.
type SourceRevenue struct {
	Source  string  `json:"source"`
	Revenue float64 `json:"revenue"`
	Count   int     `json:"count"`
}

// DailyOrderTrend holds order count and average value for a single day.
type DailyOrderTrend struct {
	Date     string  `json:"date"`
	Count    int     `json:"count"`
	AvgValue float64 `json:"avg_value"`
}
