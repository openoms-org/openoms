package model

// CarbonStats holds aggregate carbon footprint statistics for a tenant.
type CarbonStats struct {
	TotalShipments   int                  `json:"total_shipments"`
	TotalCarbonKg    float64              `json:"total_carbon_kg"`
	AvgCarbonPerShip float64              `json:"avg_carbon_per_shipment"`
	TotalDistanceKm  float64              `json:"total_distance_km"`
	ByCarrier        []CarrierCarbonStats `json:"by_carrier"`
	MonthlyTrend     []MonthlyCarbonStat  `json:"monthly_trend"`
}

// CarrierCarbonStats holds carbon stats broken down by carrier/provider.
type CarrierCarbonStats struct {
	Carrier     string  `json:"carrier"`
	Shipments   int     `json:"shipments"`
	TotalCarbon float64 `json:"total_carbon_kg"`
	AvgCarbon   float64 `json:"avg_carbon_kg"`
}

// MonthlyCarbonStat holds carbon stats for a single month.
type MonthlyCarbonStat struct {
	Month       string  `json:"month"` // "2026-02"
	Shipments   int     `json:"shipments"`
	TotalCarbon float64 `json:"total_carbon_kg"`
}
