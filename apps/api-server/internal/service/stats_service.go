package service

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type StatsService struct {
	statsRepo repository.StatsRepo
	pool      *pgxpool.Pool
}

func NewStatsService(
	statsRepo repository.StatsRepo,
	pool *pgxpool.Pool,
) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		pool:      pool,
	}
}

func (s *StatsService) GetDashboardStats(ctx context.Context, tenantID uuid.UUID) (*model.DashboardStats, error) {
	var stats model.DashboardStats
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var (
			byStatus     map[string]int
			bySource     map[string]int
			totalRevenue float64
			dailyRevenue []model.DailyRevenue
			recentOrders []model.OrderSummary
			currency     string
			mu           sync.Mutex
			wg           sync.WaitGroup
			firstErr     error
		)

		setErr := func(err error) {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}

		wg.Add(6)

		go func() {
			defer wg.Done()
			result, err := s.statsRepo.GetOrderCountByStatus(ctx, tx)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			byStatus = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.statsRepo.GetOrderCountBySource(ctx, tx)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			bySource = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.statsRepo.GetTotalRevenue(ctx, tx)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			totalRevenue = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.statsRepo.GetDailyRevenue(ctx, tx, 30)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			dailyRevenue = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.statsRepo.GetRecentOrders(ctx, tx, 10)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			recentOrders = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.statsRepo.GetMostCommonCurrency(ctx, tx)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			currency = result
			mu.Unlock()
		}()

		wg.Wait()

		if firstErr != nil {
			return firstErr
		}

		total := 0
		for _, count := range byStatus {
			total += count
		}

		if dailyRevenue == nil {
			dailyRevenue = []model.DailyRevenue{}
		}
		if recentOrders == nil {
			recentOrders = []model.OrderSummary{}
		}
		// Default to PLN if no orders exist yet
		if currency == "" {
			currency = "PLN"
		}

		stats = model.DashboardStats{
			OrderCounts: model.OrderCounts{
				Total:    total,
				ByStatus: byStatus,
				BySource: bySource,
			},
			Revenue: model.Revenue{
				Total:    totalRevenue,
				Currency: currency,
				Daily:    dailyRevenue,
			},
			RecentOrders: recentOrders,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
