package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrLoyaltyProgramNotFound is returned when a loyalty program does not exist.
	ErrLoyaltyProgramNotFound = errors.New("loyalty program not found")
	// ErrInsufficientPoints is returned when a customer has too few loyalty points.
	ErrInsufficientPoints     = errors.New("insufficient points")
)

// LoyaltyService handles business logic for loyalty programs and points.
type LoyaltyService struct {
	loyaltyRepo *repository.LoyaltyRepository
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
	logger      *slog.Logger
}

// NewLoyaltyService creates a new LoyaltyService.
func NewLoyaltyService(
	loyaltyRepo *repository.LoyaltyRepository,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) *LoyaltyService {
	return &LoyaltyService{
		loyaltyRepo: loyaltyRepo,
		auditRepo:   auditRepo,
		pool:        pool,
		logger:      logger,
	}
}

// ListPrograms returns a paginated list of loyalty programs.
func (s *LoyaltyService) ListPrograms(ctx context.Context, tenantID uuid.UUID, filter model.LoyaltyProgramListFilter) (model.ListResponse[model.LoyaltyProgram], error) {
	var resp model.ListResponse[model.LoyaltyProgram]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		programs, total, err := s.loyaltyRepo.ListPrograms(ctx, tx, filter)
		if err != nil {
			return err
		}
		if programs == nil {
			programs = []model.LoyaltyProgram{}
		}
		resp = model.ListResponse[model.LoyaltyProgram]{
			Items:  programs,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// GetProgram returns a single loyalty program by ID.
func (s *LoyaltyService) GetProgram(ctx context.Context, tenantID, programID uuid.UUID) (*model.LoyaltyProgram, error) {
	var program *model.LoyaltyProgram
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		program, err = s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if program == nil {
		return nil, ErrLoyaltyProgramNotFound
	}
	return program, nil
}

// CreateProgram inserts a new loyalty program.
func (s *LoyaltyService) CreateProgram(ctx context.Context, tenantID uuid.UUID, req model.CreateLoyaltyProgramRequest, actorID uuid.UUID, ip string) (*model.LoyaltyProgram, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	program := &model.LoyaltyProgram{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        model.StripHTMLTags(req.Name),
		Status:      "active",
		ProgramType: req.ProgramType,
		Config:      req.Config,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.loyaltyRepo.CreateProgram(ctx, tx, program); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "loyalty_program.created",
			EntityType: "loyalty_program",
			EntityID:   program.ID,
			Changes:    map[string]string{"name": program.Name, "type": program.ProgramType},
			IPAddress:  ip,
		})
	})
	return program, err
}

// UpdateProgram modifies an existing loyalty program.
func (s *LoyaltyService) UpdateProgram(ctx context.Context, tenantID, programID uuid.UUID, req model.UpdateLoyaltyProgramRequest, actorID uuid.UUID, ip string) (*model.LoyaltyProgram, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var program *model.LoyaltyProgram
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrLoyaltyProgramNotFound
		}

		if err := s.loyaltyRepo.UpdateProgram(ctx, tx, programID, req); err != nil {
			return err
		}

		program, err = s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "loyalty_program.updated",
			EntityType: "loyalty_program",
			EntityID:   programID,
			IPAddress:  ip,
		})
	})
	return program, err
}

// DeleteProgram removes a loyalty program by ID.
func (s *LoyaltyService) DeleteProgram(ctx context.Context, tenantID, programID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrLoyaltyProgramNotFound
		}

		if err := s.loyaltyRepo.DeleteProgram(ctx, tx, programID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "loyalty_program.deleted",
			EntityType: "loyalty_program",
			EntityID:   programID,
			Changes:    map[string]string{"name": existing.Name},
			IPAddress:  ip,
		})
	})
}

// AwardPoints manually awards points to a customer in a program.
func (s *LoyaltyService) AwardPoints(ctx context.Context, tenantID uuid.UUID, programID uuid.UUID, req model.AwardPointsRequest, actorID uuid.UUID, ip string) error {
	if err := req.Validate(); err != nil {
		return NewValidationError(err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		program, err := s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}
		if program == nil {
			return ErrLoyaltyProgramNotFound
		}

		// Ensure customer has a loyalty record
		cl, err := s.loyaltyRepo.GetCustomerLoyalty(ctx, tx, req.CustomerID, programID)
		if err != nil {
			return err
		}
		if cl == nil {
			now := time.Now()
			cl = &model.CustomerLoyalty{
				TenantID:       tenantID,
				CustomerID:     req.CustomerID,
				ProgramID:      programID,
				LastActivityAt: &now,
			}
			if err := s.loyaltyRepo.UpsertCustomerLoyalty(ctx, tx, cl); err != nil {
				return err
			}
		}

		if err := s.loyaltyRepo.AddPoints(ctx, tx, req.CustomerID, programID, req.Points); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "loyalty.points_awarded",
			EntityType: "loyalty_program",
			EntityID:   programID,
			Changes: map[string]string{
				"customer_id": req.CustomerID.String(),
				"points":      itoa(req.Points),
				"reason":      req.Reason,
			},
			IPAddress: ip,
		})
	})
}

// RedeemPoints redeems points from a customer's loyalty balance.
func (s *LoyaltyService) RedeemPoints(ctx context.Context, tenantID uuid.UUID, programID uuid.UUID, req model.RedeemPointsRequest, actorID uuid.UUID, ip string) error {
	if err := req.Validate(); err != nil {
		return NewValidationError(err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		program, err := s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}
		if program == nil {
			return ErrLoyaltyProgramNotFound
		}

		cl, err := s.loyaltyRepo.GetCustomerLoyalty(ctx, tx, req.CustomerID, programID)
		if err != nil {
			return err
		}
		if cl == nil || cl.PointsBalance < req.Points {
			return ErrInsufficientPoints
		}

		if err := s.loyaltyRepo.RedeemPoints(ctx, tx, req.CustomerID, programID, req.Points); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "loyalty.points_redeemed",
			EntityType: "loyalty_program",
			EntityID:   programID,
			Changes: map[string]string{
				"customer_id": req.CustomerID.String(),
				"points":      itoa(req.Points),
			},
			IPAddress: ip,
		})
	})
}

// GetCustomerLoyaltyStatus returns all loyalty program memberships for a customer.
func (s *LoyaltyService) GetCustomerLoyaltyStatus(ctx context.Context, tenantID, customerID uuid.UUID) ([]model.CustomerLoyaltyWithProgram, error) {
	var results []model.CustomerLoyaltyWithProgram
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		results, err = s.loyaltyRepo.GetCustomerLoyaltyStatus(ctx, tx, customerID)
		return err
	})
	if results == nil {
		results = []model.CustomerLoyaltyWithProgram{}
	}
	return results, err
}

// GetLeaderboard returns the top customers for a loyalty program.
func (s *LoyaltyService) GetLeaderboard(ctx context.Context, tenantID, programID uuid.UUID, limit int) ([]model.LeaderboardEntry, error) {
	var entries []model.LeaderboardEntry
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		program, err := s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}
		if program == nil {
			return ErrLoyaltyProgramNotFound
		}

		entries, err = s.loyaltyRepo.GetLeaderboard(ctx, tx, programID, limit)
		return err
	})
	if entries == nil {
		entries = []model.LeaderboardEntry{}
	}
	return entries, err
}

// UpdateTier recalculates a customer's tier in a tier-based program.
func (s *LoyaltyService) UpdateTier(ctx context.Context, tenantID, customerID, programID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		program, err := s.loyaltyRepo.FindProgramByID(ctx, tx, programID)
		if err != nil {
			return err
		}
		if program == nil {
			return ErrLoyaltyProgramNotFound
		}
		if program.ProgramType != "tier" {
			return nil // only applicable for tier programs
		}

		cl, err := s.loyaltyRepo.GetCustomerLoyalty(ctx, tx, customerID, programID)
		if err != nil {
			return err
		}
		if cl == nil {
			return nil
		}

		// Parse tier config
		type TierDef struct {
			Name     string  `json:"name"`
			MinSpent float64 `json:"min_spent"`
		}
		type TierConfig struct {
			Tiers []TierDef `json:"tiers"`
		}
		var cfg TierConfig
		if err := json.Unmarshal(program.Config, &cfg); err != nil {
			s.logger.Warn("invalid tier config", "program_id", programID, "error", err)
			return nil
		}

		// Find the highest tier the customer qualifies for
		var tier string
		for _, t := range cfg.Tiers {
			if cl.TotalSpent >= t.MinSpent {
				tier = t.Name
			}
		}

		if tier != "" && (cl.CurrentTier == nil || *cl.CurrentTier != tier) {
			cl.CurrentTier = &tier
			return s.loyaltyRepo.UpsertCustomerLoyalty(ctx, tx, cl)
		}
		return nil
	})
}

// AwardPointsForOrder awards loyalty points based on order amount (called automatically).
func (s *LoyaltyService) AwardPointsForOrder(ctx context.Context, tenantID, customerID uuid.UUID, orderAmount float64) {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Find all active programs
		programs, _, err := s.loyaltyRepo.ListPrograms(ctx, tx, model.LoyaltyProgramListFilter{
			PaginationParams: model.PaginationParams{Limit: 100, Offset: 0},
		})
		if err != nil {
			return err
		}

		now := time.Now()
		for _, program := range programs {
			if program.Status != "active" {
				continue
			}

			// Ensure customer has a loyalty record
			cl, err := s.loyaltyRepo.GetCustomerLoyalty(ctx, tx, customerID, program.ID)
			if err != nil {
				return err
			}
			if cl == nil {
				cl = &model.CustomerLoyalty{
					TenantID:       tenantID,
					CustomerID:     customerID,
					ProgramID:      program.ID,
					LastActivityAt: &now,
				}
				if err := s.loyaltyRepo.UpsertCustomerLoyalty(ctx, tx, cl); err != nil {
					return err
				}
			}

			// Update spending and order count
			cl.TotalSpent += orderAmount
			cl.OrderCount++
			cl.LastActivityAt = &now

			switch program.ProgramType {
			case "points":
				type PointsConfig struct {
					PointsPerPLN int `json:"points_per_pln"`
				}
				var cfg PointsConfig
				if err := json.Unmarshal(program.Config, &cfg); err != nil {
					s.logger.Warn("invalid points config", "program_id", program.ID, "error", err)
					continue
				}
				if cfg.PointsPerPLN <= 0 {
					cfg.PointsPerPLN = 1
				}
				points := int(orderAmount) * cfg.PointsPerPLN
				cl.PointsBalance += points
				cl.TotalPointsEarned += points

			case "tier":
				// Just update spending/count, tier will be recalculated
				type TierDef struct {
					Name     string  `json:"name"`
					MinSpent float64 `json:"min_spent"`
				}
				type TierConfig struct {
					Tiers []TierDef `json:"tiers"`
				}
				var cfg TierConfig
				if err := json.Unmarshal(program.Config, &cfg); err == nil {
					for _, t := range cfg.Tiers {
						if cl.TotalSpent >= t.MinSpent {
							cl.CurrentTier = &t.Name
						}
					}
				}

			case "discount_after_n":
				// Track order count, the discount eligibility is checked at order time
			}

			if err := s.loyaltyRepo.UpsertCustomerLoyalty(ctx, tx, cl); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error("failed to award loyalty points", "error", err, "customer_id", customerID)
	}
}
