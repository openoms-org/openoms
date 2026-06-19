package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// ForecastService provides statistical demand forecasting and reorder recommendations.
type ForecastService struct {
	pool         *pgxpool.Pool
	productRepo  repository.ProductRepo
	tenantRepo   repository.TenantRepo
	orderRepo    repository.OrderRepo
	supplierRepo repository.SupplierRepo
}

// NewForecastService creates a new ForecastService.
func NewForecastService(
	pool *pgxpool.Pool,
	productRepo repository.ProductRepo,
	tenantRepo repository.TenantRepo,
	orderRepo repository.OrderRepo,
	supplierRepo repository.SupplierRepo,
) *ForecastService {
	return &ForecastService{
		pool:         pool,
		productRepo:  productRepo,
		tenantRepo:   tenantRepo,
		orderRepo:    orderRepo,
		supplierRepo: supplierRepo,
	}
}

// orderItem is a parsed item from the order's Items JSONB.
type orderItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
}

// dailySales is a helper for date-based aggregation.
type dailySales struct {
	Date     time.Time
	Quantity int
	Revenue  float64
}

// productSalesData holds aggregated sales history for a single product.
type productSalesData struct {
	ProductID   uuid.UUID
	ProductName string
	SKU         string
	Stock       int
	Price       float64
	DailySales  []dailySales
	TotalUnits  int
	TotalRev    float64
}

// getForecastConfig reads forecast config from tenant settings, or returns defaults.
func (s *ForecastService) getForecastConfig(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) model.ForecastConfig {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil || settings == nil {
		return model.DefaultForecastConfig()
	}

	var all map[string]json.RawMessage
	if err := json.Unmarshal(settings, &all); err != nil {
		return model.DefaultForecastConfig()
	}

	raw, ok := all["forecast"]
	if !ok {
		return model.DefaultForecastConfig()
	}

	var cfg model.ForecastConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return model.DefaultForecastConfig()
	}

	// Enforce sane minimums
	if cfg.DefaultLeadTimeDays <= 0 {
		cfg.DefaultLeadTimeDays = 14
	}
	if cfg.SafetyStockDays <= 0 {
		cfg.SafetyStockDays = 7
	}
	if cfg.ForecastDaysAhead <= 0 {
		cfg.ForecastDaysAhead = 30
	}
	return cfg
}

// fetchSalesHistory queries orders from the last 90 days and returns per-product daily sales.
func (s *ForecastService) fetchSalesHistory(ctx context.Context, tx pgx.Tx) (map[uuid.UUID]*productSalesData, error) {
	// Query orders from last 90 days with items (capped at 50k most recent to prevent OOM).
	// DESC + LIMIT ensures we keep the newest data if there are >50k orders.
	rows, err := tx.Query(ctx, `
		SELECT o.items, o.total_amount, o.created_at
		FROM orders o
		WHERE o.created_at >= NOW() - INTERVAL '90 days'
		  AND o.status NOT IN ('cancelled', 'refunded')
		ORDER BY o.created_at DESC
		LIMIT 50000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	productMap := make(map[uuid.UUID]*productSalesData)
	// dateMap: productID -> date -> aggregated quantity+revenue
	type dateBucket struct {
		qty int
		rev float64
	}
	dateMap := make(map[uuid.UUID]map[string]*dateBucket)

	for rows.Next() {
		var itemsJSON json.RawMessage
		var totalAmount float64
		var createdAt time.Time
		if err := rows.Scan(&itemsJSON, &totalAmount, &createdAt); err != nil {
			return nil, err
		}

		dateKey := createdAt.Format("2006-01-02")

		var items []orderItem
		if itemsJSON != nil {
			if err := json.Unmarshal(itemsJSON, &items); err != nil {
				// Skip orders with unparseable items
				continue
			}
		}

		for _, item := range items {
			if item.ProductID == uuid.Nil {
				continue
			}
			if _, ok := productMap[item.ProductID]; !ok {
				productMap[item.ProductID] = &productSalesData{
					ProductID:   item.ProductID,
					ProductName: item.Name,
					SKU:         item.SKU,
				}
				dateMap[item.ProductID] = make(map[string]*dateBucket)
			}

			pd := productMap[item.ProductID]
			pd.TotalUnits += item.Quantity
			pd.TotalRev += item.Price * float64(item.Quantity)

			if _, ok := dateMap[item.ProductID][dateKey]; !ok {
				dateMap[item.ProductID][dateKey] = &dateBucket{}
			}
			dateMap[item.ProductID][dateKey].qty += item.Quantity
			dateMap[item.ProductID][dateKey].rev += item.Price * float64(item.Quantity)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Convert dateMap to sorted dailySales slices
	for pid, dm := range dateMap {
		pd := productMap[pid]
		for dateStr, bucket := range dm {
			t, _ := time.Parse("2006-01-02", dateStr)
			pd.DailySales = append(pd.DailySales, dailySales{
				Date:     t,
				Quantity: bucket.qty,
				Revenue:  bucket.rev,
			})
		}
		sort.Slice(pd.DailySales, func(i, j int) bool {
			return pd.DailySales[i].Date.Before(pd.DailySales[j].Date)
		})
	}

	// Fetch product metadata (name/sku/price). Stock comes separately from the canonical
	// available store below, not the stale products.stock_quantity column.
	prodRows, err := tx.Query(ctx, `SELECT id, name, COALESCE(sku, ''), price FROM products`)
	if err != nil {
		return nil, err
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var id uuid.UUID
		var name, sku string
		var price float64
		if err := prodRows.Scan(&id, &name, &sku, &price); err != nil {
			return nil, err
		}
		if pd, ok := productMap[id]; ok {
			pd.ProductName = name
			pd.SKU = sku
			pd.Price = price
		}
	}
	if err := prodRows.Err(); err != nil {
		return nil, err
	}

	// Stock = canonical available (warehouse_stock quantity - reserved, with a
	// products.stock_quantity fallback for products that have no warehouse rows), not the stale
	// products.stock_quantity column which is never decremented on shipment.
	ids := make([]uuid.UUID, 0, len(productMap))
	for id := range productMap {
		ids = append(ids, id)
	}
	avail, err := s.productRepo.AvailableStockBatch(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	for id, pd := range productMap {
		pd.Stock = avail[id]
	}

	return productMap, nil
}

// computeForecast calculates a demand forecast for a given product's sales data.
func computeForecast(pd *productSalesData, daysAhead int) model.Forecast {
	now := time.Now()
	startDate := now.AddDate(0, 0, -90)
	totalDays := int(now.Sub(startDate).Hours() / 24)
	if totalDays <= 0 {
		totalDays = 1
	}

	// Build a full 90-day array (fill zeros for days with no sales)
	dailyArr := make([]float64, totalDays)
	for _, ds := range pd.DailySales {
		idx := int(ds.Date.Sub(startDate).Hours() / 24)
		if idx >= 0 && idx < totalDays {
			dailyArr[idx] += float64(ds.Quantity)
		}
	}

	// Average daily sales
	totalQty := 0.0
	for _, v := range dailyArr {
		totalQty += v
	}
	avgDaily := totalQty / float64(totalDays)

	// Standard deviation
	sumSqDiff := 0.0
	for _, v := range dailyArr {
		diff := v - avgDaily
		sumSqDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSqDiff / float64(totalDays))

	// Weekly seasonality index (day-of-week multiplier)
	// This is computed but used in the forecast via the trend factor only for simplicity.

	// Trend: linear regression on weekly totals
	// Group into weeks
	numWeeks := totalDays / 7
	if numWeeks < 2 {
		numWeeks = 0
	}
	trendDirection := "stable"
	trendPct := 0.0
	trendFactor := 1.0

	if numWeeks >= 2 {
		weeklyTotals := make([]float64, numWeeks)
		for i := 0; i < numWeeks; i++ {
			weekStart := i * 7
			weekEnd := min(weekStart+7, totalDays)
			for j := weekStart; j < weekEnd; j++ {
				weeklyTotals[i] += dailyArr[j]
			}
		}

		// Linear regression: y = a + b*x
		n := float64(numWeeks)
		sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0
		for i, y := range weeklyTotals {
			x := float64(i)
			sumX += x
			sumY += y
			sumXY += x * y
			sumXX += x * x
		}

		denominator := n*sumXX - sumX*sumX
		if denominator != 0 {
			slope := (n*sumXY - sumX*sumY) / denominator
			avgWeekly := sumY / n

			if avgWeekly > 0 {
				weeklyTrendPct := (slope / avgWeekly) * 100
				trendPct = weeklyTrendPct

				if weeklyTrendPct > 5 {
					trendDirection = "up"
					// Cap trend factor at 2x
					trendFactor = math.Min(1.0+(weeklyTrendPct/100)*float64(daysAhead)/30, 2.0)
				} else if weeklyTrendPct < -5 {
					trendDirection = "down"
					// Floor trend factor at 0.5x
					trendFactor = math.Max(1.0+(weeklyTrendPct/100)*float64(daysAhead)/30, 0.5)
				}
			}
		}
	}

	// Forecasted demand
	forecastedUnits := avgDaily * float64(daysAhead) * trendFactor
	confidenceLow := math.Max(0, forecastedUnits-stdDev*math.Sqrt(float64(daysAhead)))
	confidenceHigh := forecastedUnits + stdDev*math.Sqrt(float64(daysAhead))

	// Days of stock
	daysOfStock := 0.0
	if avgDaily > 0 {
		daysOfStock = float64(pd.Stock) / avgDaily
	} else if pd.Stock > 0 {
		daysOfStock = 999 // effectively infinite
	}

	lowData := len(pd.DailySales) < 14

	sku := pd.SKU
	return model.Forecast{
		ProductID:       pd.ProductID,
		ProductName:     pd.ProductName,
		SKU:             sku,
		CurrentStock:    pd.Stock,
		DaysAhead:       daysAhead,
		ForecastedUnits: math.Round(forecastedUnits*100) / 100,
		ConfidenceLow:   math.Round(confidenceLow*100) / 100,
		ConfidenceHigh:  math.Round(confidenceHigh*100) / 100,
		AvgDailySales:   math.Round(avgDaily*100) / 100,
		TrendDirection:  trendDirection,
		TrendPct:        math.Round(trendPct*100) / 100,
		DaysOfStock:     math.Round(daysOfStock*10) / 10,
		LowDataWarning:  lowData,
	}
}

// ForecastDemand returns a demand forecast for a single product.
func (s *ForecastService) ForecastDemand(ctx context.Context, tenantID, productID uuid.UUID, daysAhead int) (*model.Forecast, error) {
	var result *model.Forecast

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if daysAhead <= 0 {
			cfg := s.getForecastConfig(ctx, tx, tenantID)
			daysAhead = cfg.ForecastDaysAhead
		}

		productMap, err := s.fetchSalesHistory(ctx, tx)
		if err != nil {
			return err
		}

		pd, ok := productMap[productID]
		if !ok {
			// Product exists but has no sales — still produce a forecast with zero sales
			product, err := s.productRepo.FindByID(ctx, tx, productID)
			if err != nil {
				return err
			}
			if product == nil {
				return ErrProductNotFound
			}
			sku := ""
			if product.SKU != nil {
				sku = *product.SKU
			}
			pd = &productSalesData{
				ProductID:   product.ID,
				ProductName: product.Name,
				SKU:         sku,
				Stock:       product.StockQuantity,
				Price:       product.Price,
			}
		}

		fc := computeForecast(pd, daysAhead)
		result = &fc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForecastAll returns demand forecasts for all products with sales history.
func (s *ForecastService) ForecastAll(ctx context.Context, tenantID uuid.UUID, daysAhead int) ([]model.Forecast, error) {
	var results []model.Forecast

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if daysAhead <= 0 {
			cfg := s.getForecastConfig(ctx, tx, tenantID)
			daysAhead = cfg.ForecastDaysAhead
		}

		productMap, err := s.fetchSalesHistory(ctx, tx)
		if err != nil {
			return err
		}

		for _, pd := range productMap {
			fc := computeForecast(pd, daysAhead)
			results = append(results, fc)
		}

		// Sort by forecasted demand DESC
		sort.Slice(results, func(i, j int) bool {
			return results[i].ForecastedUnits > results[j].ForecastedUnits
		})

		return nil
	})
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []model.Forecast{}
	}
	return results, nil
}

// GetReorderRecommendations returns reorder suggestions for products at risk of stockout.
func (s *ForecastService) GetReorderRecommendations(ctx context.Context, tenantID uuid.UUID) ([]model.ReorderRecommendation, error) {
	var results []model.ReorderRecommendation

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		cfg := s.getForecastConfig(ctx, tx, tenantID)

		productMap, err := s.fetchSalesHistory(ctx, tx)
		if err != nil {
			return err
		}

		// Fetch supplier linkages (product -> supplier) via supplier_products table
		supplierLinks := make(map[uuid.UUID]struct {
			SupplierID   uuid.UUID
			SupplierName string
			UnitCost     float64
		})
		spRows, err := tx.Query(ctx, `
			SELECT sp.product_id, sp.supplier_id, s.name, sp.price
			FROM supplier_products sp
			JOIN suppliers s ON s.id = sp.supplier_id
			WHERE sp.product_id IS NOT NULL
		`)
		if err == nil {
			defer spRows.Close()
			for spRows.Next() {
				var productID, supplierID uuid.UUID
				var name string
				var price *float64
				if err := spRows.Scan(&productID, &supplierID, &name, &price); err != nil {
					continue
				}
				unitCost := 0.0
				if price != nil {
					unitCost = *price
				}
				supplierLinks[productID] = struct {
					SupplierID   uuid.UUID
					SupplierName string
					UnitCost     float64
				}{supplierID, name, unitCost}
			}
		}

		for _, pd := range productMap {
			// Calculate forecast for lead time period
			leadTimeDays := cfg.DefaultLeadTimeDays
			fc := computeForecast(pd, leadTimeDays)

			avgDaily := fc.AvgDailySales
			if avgDaily <= 0 {
				continue // No demand, no need to reorder
			}

			safetyStock := int(math.Ceil(avgDaily * float64(cfg.SafetyStockDays)))
			demandDuringLead := fc.ForecastedUnits
			reorderPoint := int(math.Ceil(demandDuringLead)) + safetyStock

			if pd.Stock >= reorderPoint {
				continue // Stock is sufficient
			}

			// Days until stockout
			daysUntilStockout := float64(pd.Stock) / avgDaily

			// Urgency
			urgency := "planned"
			if daysUntilStockout < 7 {
				urgency = "critical"
			} else if daysUntilStockout < 14 {
				urgency = "soon"
			}

			// Recommended quantity: enough to cover lead time + safety stock + buffer
			// Simplified EOQ: order enough for 2x lead time minus current stock
			recommendedQty := max(int(math.Ceil(avgDaily*float64(leadTimeDays)*2))+safetyStock-pd.Stock, 1)

			rec := model.ReorderRecommendation{
				ProductID:         pd.ProductID,
				ProductName:       pd.ProductName,
				SKU:               pd.SKU,
				CurrentStock:      pd.Stock,
				ForecastedDemand:  math.Round(demandDuringLead*100) / 100,
				SafetyStock:       safetyStock,
				ReorderPoint:      reorderPoint,
				RecommendedQty:    recommendedQty,
				Urgency:           urgency,
				DaysUntilStockout: math.Round(daysUntilStockout*10) / 10,
			}

			// Add supplier info
			if link, ok := supplierLinks[pd.ProductID]; ok {
				rec.SupplierID = &link.SupplierID
				rec.SupplierName = link.SupplierName
				rec.EstimatedCost = math.Round(float64(recommendedQty)*link.UnitCost*100) / 100
			} else {
				rec.EstimatedCost = math.Round(float64(recommendedQty)*pd.Price*0.5*100) / 100 // rough estimate: 50% of sell price
			}

			results = append(results, rec)
		}

		// Sort by urgency: critical > soon > planned, then by days until stockout ASC
		urgencyOrder := map[string]int{"critical": 0, "soon": 1, "planned": 2}
		sort.Slice(results, func(i, j int) bool {
			oi, oj := urgencyOrder[results[i].Urgency], urgencyOrder[results[j].Urgency]
			if oi != oj {
				return oi < oj
			}
			return results[i].DaysUntilStockout < results[j].DaysUntilStockout
		})

		return nil
	})
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []model.ReorderRecommendation{}
	}
	return results, nil
}

// GetSeasonalityAnalysis returns day-of-week and monthly sales patterns for a product.
func (s *ForecastService) GetSeasonalityAnalysis(ctx context.Context, tenantID, productID uuid.UUID) (*model.SeasonalityData, error) {
	var result *model.SeasonalityData

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify product exists
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotFound
		}

		productMap, err := s.fetchSalesHistory(ctx, tx)
		if err != nil {
			return err
		}

		pd, ok := productMap[productID]
		if !ok {
			// No sales history
			result = &model.SeasonalityData{
				ProductID:   productID,
				ProductName: product.Name,
				ByDayOfWeek: makeEmptyDayOfWeek(),
				ByMonth:     makeEmptyMonths(),
			}
			return nil
		}

		// Day-of-week analysis
		dayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		daySums := make([]float64, 7)
		dayCounts := make([]int, 7)

		for _, ds := range pd.DailySales {
			// Go: Sunday=0, Monday=1, ... but we want Monday=0
			dow := int(ds.Date.Weekday())
			idx := (dow + 6) % 7 // Monday=0, ..., Sunday=6
			daySums[idx] += float64(ds.Quantity)
			dayCounts[idx]++
		}

		totalDayAvg := 0.0
		dayAvgs := make([]float64, 7)
		validDays := 0
		for i := range 7 {
			if dayCounts[i] > 0 {
				dayAvgs[i] = daySums[i] / float64(dayCounts[i])
				totalDayAvg += dayAvgs[i]
				validDays++
			}
		}
		overallDayAvg := 0.0
		if validDays > 0 {
			overallDayAvg = totalDayAvg / float64(validDays)
		}

		byDayOfWeek := make([]model.DayOfWeekSales, 7)
		for i := range 7 {
			index := 0.0
			if overallDayAvg > 0 {
				index = dayAvgs[i] / overallDayAvg
			}
			byDayOfWeek[i] = model.DayOfWeekSales{
				Day:      dayNames[i],
				AvgSales: math.Round(dayAvgs[i]*100) / 100,
				Index:    math.Round(index*100) / 100,
			}
		}

		// Month analysis
		monthNames := []string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		}
		monthSums := make([]float64, 12)
		monthDays := make([]int, 12)

		for _, ds := range pd.DailySales {
			m := int(ds.Date.Month()) - 1
			monthSums[m] += float64(ds.Quantity)
			monthDays[m]++
		}

		totalMonthAvg := 0.0
		monthAvgs := make([]float64, 12)
		validMonths := 0
		for i := range 12 {
			if monthDays[i] > 0 {
				monthAvgs[i] = monthSums[i] / float64(monthDays[i])
				totalMonthAvg += monthAvgs[i]
				validMonths++
			}
		}
		overallMonthAvg := 0.0
		if validMonths > 0 {
			overallMonthAvg = totalMonthAvg / float64(validMonths)
		}

		byMonth := make([]model.MonthSales, 12)
		for i := range 12 {
			index := 0.0
			if overallMonthAvg > 0 {
				index = monthAvgs[i] / overallMonthAvg
			}
			byMonth[i] = model.MonthSales{
				Month:    monthNames[i],
				AvgSales: math.Round(monthAvgs[i]*100) / 100,
				Index:    math.Round(index*100) / 100,
			}
		}

		result = &model.SeasonalityData{
			ProductID:   productID,
			ProductName: pd.ProductName,
			ByDayOfWeek: byDayOfWeek,
			ByMonth:     byMonth,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetProductVelocity returns ABC classification for all products.
func (s *ForecastService) GetProductVelocity(ctx context.Context, tenantID uuid.UUID) ([]model.ProductVelocity, error) {
	var results []model.ProductVelocity

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		productMap, err := s.fetchSalesHistory(ctx, tx)
		if err != nil {
			return err
		}

		type productRevenue struct {
			pd  *productSalesData
			rev float64
		}
		var prods []productRevenue
		for _, pd := range productMap {
			prods = append(prods, productRevenue{pd: pd, rev: pd.TotalRev})
		}

		// Sort by revenue DESC
		sort.Slice(prods, func(i, j int) bool {
			return prods[i].rev > prods[j].rev
		})

		// Calculate total revenue for ABC thresholds
		totalRevenue := 0.0
		for _, p := range prods {
			totalRevenue += p.rev
		}

		// Assign ABC classes
		cumulative := 0.0
		for _, p := range prods {
			cumulative += p.rev

			abcClass := "C"
			if totalRevenue > 0 {
				pct := cumulative / totalRevenue
				if pct <= 0.80 {
					abcClass = "A"
				} else if pct <= 0.95 {
					abcClass = "B"
				}
			}

			// Calculate avg daily sales for days of stock
			now := time.Now()
			startDate := now.AddDate(0, 0, -90)
			totalDays := int(now.Sub(startDate).Hours() / 24)
			if totalDays <= 0 {
				totalDays = 1
			}
			avgDaily := float64(p.pd.TotalUnits) / float64(totalDays)

			daysOfStock := 0.0
			if avgDaily > 0 {
				daysOfStock = float64(p.pd.Stock) / avgDaily
			} else if p.pd.Stock > 0 {
				daysOfStock = 999
			}

			results = append(results, model.ProductVelocity{
				ProductID:    p.pd.ProductID,
				ProductName:  p.pd.ProductName,
				SKU:          p.pd.SKU,
				TotalRevenue: math.Round(p.rev*100) / 100,
				TotalUnits:   p.pd.TotalUnits,
				ABCClass:     abcClass,
				DaysOfStock:  math.Round(daysOfStock*10) / 10,
				CurrentStock: p.pd.Stock,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []model.ProductVelocity{}
	}
	return results, nil
}

// GetForecastConfig returns the tenant's forecast configuration.
func (s *ForecastService) GetForecastConfig(ctx context.Context, tenantID uuid.UUID) (*model.ForecastConfig, error) {
	var result model.ForecastConfig

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		result = s.getForecastConfig(ctx, tx, tenantID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateForecastConfig updates the tenant's forecast configuration.
func (s *ForecastService) UpdateForecastConfig(ctx context.Context, tenantID uuid.UUID, cfg model.ForecastConfig) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}

		allSettings := make(map[string]json.RawMessage)
		if settings != nil {
			if err := json.Unmarshal(settings, &allSettings); err != nil {
				return err
			}
		}

		cfgJSON, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		allSettings["forecast"] = cfgJSON

		merged, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		return s.tenantRepo.UpdateSettings(ctx, tx, tenantID, merged)
	})
}

func makeEmptyDayOfWeek() []model.DayOfWeekSales {
	names := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	result := make([]model.DayOfWeekSales, 7)
	for i, name := range names {
		result[i] = model.DayOfWeekSales{Day: name, AvgSales: 0, Index: 0}
	}
	return result
}

func makeEmptyMonths() []model.MonthSales {
	names := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	result := make([]model.MonthSales, 12)
	for i, name := range names {
		result[i] = model.MonthSales{Month: name, AvgSales: 0, Index: 0}
	}
	return result
}
