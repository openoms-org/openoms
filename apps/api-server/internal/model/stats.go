package model

import "time"

type DashboardStats struct {
	OrderCounts  OrderCounts    `json:"order_counts"`
	Revenue      Revenue        `json:"revenue"`
	RecentOrders []OrderSummary `json:"recent_orders"`
}

type OrderCounts struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
	BySource map[string]int `json:"by_source"`
}

type Revenue struct {
	Total    float64        `json:"total"`
	Currency string         `json:"currency"`
	Daily    []DailyRevenue `json:"daily"`
}

type DailyRevenue struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type OrderSummary struct {
	ID           string    `json:"id"`
	CustomerName string    `json:"customer_name"`
	Status       string    `json:"status"`
	Source       string    `json:"source"`
	TotalAmount  float64   `json:"total_amount"`
	Currency     string    `json:"currency"`
	CreatedAt    time.Time `json:"created_at"`
}

type TopProduct struct {
	Name          string  `json:"name"`
	SKU           string  `json:"sku,omitempty"`
	TotalQuantity int     `json:"total_quantity"`
	TotalRevenue  float64 `json:"total_revenue"`
}

type SourceRevenue struct {
	Source  string  `json:"source"`
	Revenue float64 `json:"revenue"`
	Count   int     `json:"count"`
}

type DailyOrderTrend struct {
	Date     string  `json:"date"`
	Count    int     `json:"count"`
	AvgValue float64 `json:"avg_value"`
}
