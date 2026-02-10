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
	statsRepo *repository.StatsRepository
	pool      *pgxpool.Pool
}

func NewStatsService(
	statsRepo *repository.StatsRepository,
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

		stats = model.DashboardStats{
			OrderCounts: model.OrderCounts{
				Total:    total,
				ByStatus: byStatus,
				BySource: bySource,
			},
			Revenue: model.Revenue{
				Total:    totalRevenue,
				Currency: "PLN",
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
