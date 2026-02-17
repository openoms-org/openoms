package model

import "github.com/google/uuid"

// Forecast holds the demand forecast for a single product over N days ahead.
type Forecast struct {
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	SKU             string    `json:"sku"`
	CurrentStock    int       `json:"current_stock"`
	DaysAhead       int       `json:"days_ahead"`
	ForecastedUnits float64   `json:"forecasted_units"`
	ConfidenceLow   float64   `json:"confidence_low"`
	ConfidenceHigh  float64   `json:"confidence_high"`
	AvgDailySales   float64   `json:"avg_daily_sales"`
	TrendDirection  string    `json:"trend_direction"` // "up", "down", "stable"
	TrendPct        float64   `json:"trend_pct"`
	DaysOfStock     float64   `json:"days_of_stock"`
	LowDataWarning  bool      `json:"low_data_warning"`
}

// ReorderRecommendation describes a reorder suggestion for a product.
type ReorderRecommendation struct {
	ProductID         uuid.UUID  `json:"product_id"`
	ProductName       string     `json:"product_name"`
	SKU               string     `json:"sku"`
	CurrentStock      int        `json:"current_stock"`
	ForecastedDemand  float64    `json:"forecasted_demand"`
	SafetyStock       int        `json:"safety_stock"`
	ReorderPoint      int        `json:"reorder_point"`
	RecommendedQty    int        `json:"recommended_qty"`
	Urgency           string     `json:"urgency"` // "critical", "soon", "planned"
	DaysUntilStockout float64    `json:"days_until_stockout"`
	SupplierID        *uuid.UUID `json:"supplier_id,omitempty"`
	SupplierName      string     `json:"supplier_name,omitempty"`
	EstimatedCost     float64    `json:"estimated_cost"`
}

// SeasonalityData holds the sales seasonality analysis for a product.
type SeasonalityData struct {
	ProductID   uuid.UUID        `json:"product_id"`
	ProductName string           `json:"product_name"`
	ByDayOfWeek []DayOfWeekSales `json:"by_day_of_week"`
	ByMonth     []MonthSales     `json:"by_month"`
}

// DayOfWeekSales holds average sales for a specific day of the week.
type DayOfWeekSales struct {
	Day      string  `json:"day"`
	AvgSales float64 `json:"avg_sales"`
	Index    float64 `json:"index"` // 1.0 = average, >1 = above avg
}

// MonthSales holds average sales for a specific month.
type MonthSales struct {
	Month    string  `json:"month"`
	AvgSales float64 `json:"avg_sales"`
	Index    float64 `json:"index"`
}

// ProductVelocity holds velocity/ABC classification data for a product.
type ProductVelocity struct {
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	SKU          string    `json:"sku"`
	TotalRevenue float64   `json:"total_revenue"`
	TotalUnits   int       `json:"total_units"`
	ABCClass     string    `json:"abc_class"` // "A", "B", "C"
	DaysOfStock  float64   `json:"days_of_stock"`
	CurrentStock int       `json:"current_stock"`
}

// ForecastConfig holds configurable parameters for demand forecasting.
type ForecastConfig struct {
	DefaultLeadTimeDays int `json:"default_lead_time_days"`
	SafetyStockDays     int `json:"safety_stock_days"`
	ForecastDaysAhead   int `json:"forecast_days_ahead"`
}

// DefaultForecastConfig returns sensible defaults.
func DefaultForecastConfig() ForecastConfig {
	return ForecastConfig{
		DefaultLeadTimeDays: 14,
		SafetyStockDays:     7,
		ForecastDaysAhead:   30,
	}
}
