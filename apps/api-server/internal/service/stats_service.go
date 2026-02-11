package service

import (
	"context"

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
		byStatus, err := s.statsRepo.GetOrderCountByStatus(ctx, tx)
		if err != nil {
			return err
		}

		bySource, err := s.statsRepo.GetOrderCountBySource(ctx, tx)
		if err != nil {
			return err
		}

		totalRevenue, err := s.statsRepo.GetTotalRevenue(ctx, tx)
		if err != nil {
			return err
		}

		dailyRevenue, err := s.statsRepo.GetDailyRevenue(ctx, tx, 30)
		if err != nil {
			return err
		}

		recentOrders, err := s.statsRepo.GetRecentOrders(ctx, tx, 10)
		if err != nil {
			return err
		}

		currency, err := s.statsRepo.GetMostCommonCurrency(ctx, tx)
		if err != nil {
			return err
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

func (s *StatsService) GetTopProducts(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.TopProduct, error) {
	var result []model.TopProduct
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		products, err := s.statsRepo.GetTopProducts(ctx, tx, limit)
		if err != nil {
			return err
		}
		result = products
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []model.TopProduct{}
	}
	return result, nil
}

func (s *StatsService) GetRevenueBySource(ctx context.Context, tenantID uuid.UUID, days int) ([]model.SourceRevenue, error) {
	var result []model.SourceRevenue
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		revenue, err := s.statsRepo.GetRevenueBySource(ctx, tx, days)
		if err != nil {
			return err
		}
		result = revenue
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []model.SourceRevenue{}
	}
	return result, nil
}

func (s *StatsService) GetOrderTrends(ctx context.Context, tenantID uuid.UUID, days int) ([]model.DailyOrderTrend, error) {
	var result []model.DailyOrderTrend
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		trends, err := s.statsRepo.GetOrderTrends(ctx, tx, days)
		if err != nil {
			return err
		}
		result = trends
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []model.DailyOrderTrend{}
	}
	return result, nil
}

func (s *StatsService) GetPaymentMethodStats(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	var result map[string]int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		stats, err := s.statsRepo.GetPaymentMethodStats(ctx, tx)
		if err != nil {
			return err
		}
		result = stats
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = make(map[string]int)
	}
	return result, nil
}
