package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// LoyaltyRepository handles persistence for loyalty programs and points.
type LoyaltyRepository struct{}

// NewLoyaltyRepository creates a new LoyaltyRepository.
func NewLoyaltyRepository() *LoyaltyRepository {
	return &LoyaltyRepository{}
}

var loyaltyProgramColumns = `id, tenant_id, name, status, program_type, config, created_at, updated_at`

func scanLoyaltyProgram(row interface{ Scan(dest ...any) error }) (*model.LoyaltyProgram, error) {
	var p model.LoyaltyProgram
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Status, &p.ProgramType,
		&p.Config, &p.CreatedAt, &p.UpdatedAt,
	)
	return &p, err
}

// ListPrograms returns a paginated list of loyalty programs.
func (r *LoyaltyRepository) ListPrograms(ctx context.Context, tx pgx.Tx, filter model.LoyaltyProgramListFilter) ([]model.LoyaltyProgram, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM loyalty_programs").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count loyalty programs: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"status":     "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT %s FROM loyalty_programs %s LIMIT $1 OFFSET $2`,
		loyaltyProgramColumns, orderByClause,
	)

	rows, err := tx.Query(ctx, query, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list loyalty programs: %w", err)
	}
	defer rows.Close()

	var programs []model.LoyaltyProgram
	for rows.Next() {
		p, err := scanLoyaltyProgram(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan loyalty program: %w", err)
		}
		programs = append(programs, *p)
	}
	return programs, total, rows.Err()
}

// FindProgramByID returns a loyalty program by its ID.
func (r *LoyaltyRepository) FindProgramByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.LoyaltyProgram, error) {
	p, err := scanLoyaltyProgram(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM loyalty_programs WHERE id = $1", loyaltyProgramColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find loyalty program by id: %w", err)
	}
	return p, nil
}

// CreateProgram inserts a new loyalty program.
func (r *LoyaltyRepository) CreateProgram(ctx context.Context, tx pgx.Tx, program *model.LoyaltyProgram) error {
	return tx.QueryRow(ctx,
		`INSERT INTO loyalty_programs (id, tenant_id, name, status, program_type, config)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		program.ID, program.TenantID, program.Name, program.Status,
		program.ProgramType, program.Config,
	).Scan(&program.CreatedAt, &program.UpdatedAt)
}

// UpdateProgram applies partial updates to a loyalty program.
func (r *LoyaltyRepository) UpdateProgram(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateLoyaltyProgramRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Config != nil {
		setClauses = append(setClauses, fmt.Sprintf("config = $%d", argIdx))
		args = append(args, req.Config)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE loyalty_programs SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update loyalty program: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("loyalty program not found")
	}
	return nil
}

// DeleteProgram removes a loyalty program by its ID.
func (r *LoyaltyRepository) DeleteProgram(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM loyalty_programs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete loyalty program: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("loyalty program not found")
	}
	return nil
}

// GetCustomerLoyalty retrieves a customer's loyalty record for a specific program.
func (r *LoyaltyRepository) GetCustomerLoyalty(ctx context.Context, tx pgx.Tx, customerID, programID uuid.UUID) (*model.CustomerLoyalty, error) {
	var cl model.CustomerLoyalty
	err := tx.QueryRow(ctx,
		`SELECT tenant_id, customer_id, program_id, points_balance, total_points_earned,
		        total_points_redeemed, current_tier, total_spent, order_count,
		        last_activity_at, created_at, updated_at
		 FROM customer_loyalty WHERE customer_id = $1 AND program_id = $2`,
		customerID, programID,
	).Scan(
		&cl.TenantID, &cl.CustomerID, &cl.ProgramID, &cl.PointsBalance,
		&cl.TotalPointsEarned, &cl.TotalPointsRedeemed, &cl.CurrentTier,
		&cl.TotalSpent, &cl.OrderCount, &cl.LastActivityAt,
		&cl.CreatedAt, &cl.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer loyalty: %w", err)
	}
	return &cl, nil
}

// UpsertCustomerLoyalty creates or updates a customer loyalty record.
func (r *LoyaltyRepository) UpsertCustomerLoyalty(ctx context.Context, tx pgx.Tx, cl *model.CustomerLoyalty) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO customer_loyalty (tenant_id, customer_id, program_id, points_balance,
		 total_points_earned, total_points_redeemed, current_tier, total_spent, order_count, last_activity_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (customer_id, program_id)
		 DO UPDATE SET
		   points_balance = $4,
		   total_points_earned = $5,
		   total_points_redeemed = $6,
		   current_tier = $7,
		   total_spent = $8,
		   order_count = $9,
		   last_activity_at = $10,
		   updated_at = NOW()`,
		cl.TenantID, cl.CustomerID, cl.ProgramID, cl.PointsBalance,
		cl.TotalPointsEarned, cl.TotalPointsRedeemed, cl.CurrentTier,
		cl.TotalSpent, cl.OrderCount, cl.LastActivityAt,
	)
	if err != nil {
		return fmt.Errorf("upsert customer loyalty: %w", err)
	}
	return nil
}

// AddPoints adds points to a customer's loyalty balance.
func (r *LoyaltyRepository) AddPoints(ctx context.Context, tx pgx.Tx, customerID, programID uuid.UUID, points int) error {
	_, err := tx.Exec(ctx,
		`UPDATE customer_loyalty SET
		 points_balance = points_balance + $1,
		 total_points_earned = total_points_earned + $1,
		 last_activity_at = NOW(),
		 updated_at = NOW()
		 WHERE customer_id = $2 AND program_id = $3`,
		points, customerID, programID,
	)
	return err
}

// RedeemPoints redeems points from a customer's loyalty balance.
func (r *LoyaltyRepository) RedeemPoints(ctx context.Context, tx pgx.Tx, customerID, programID uuid.UUID, points int) error {
	ct, err := tx.Exec(ctx,
		`UPDATE customer_loyalty SET
		 points_balance = points_balance - $1,
		 total_points_redeemed = total_points_redeemed + $1,
		 last_activity_at = NOW(),
		 updated_at = NOW()
		 WHERE customer_id = $2 AND program_id = $3 AND points_balance >= $1`,
		points, customerID, programID,
	)
	if err != nil {
		return fmt.Errorf("redeem points: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("insufficient points or record not found")
	}
	return nil
}

// GetLeaderboard returns top customers by points for a program.
func (r *LoyaltyRepository) GetLeaderboard(ctx context.Context, tx pgx.Tx, programID uuid.UUID, limit int) ([]model.LeaderboardEntry, error) {
	rows, err := tx.Query(ctx,
		`SELECT cl.customer_id, c.name, cl.points_balance, cl.total_spent, cl.order_count, cl.current_tier
		 FROM customer_loyalty cl
		 JOIN customers c ON c.id = cl.customer_id
		 WHERE cl.program_id = $1
		 ORDER BY cl.total_points_earned DESC
		 LIMIT $2`,
		programID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []model.LeaderboardEntry
	for rows.Next() {
		var e model.LeaderboardEntry
		if err := rows.Scan(&e.CustomerID, &e.CustomerName, &e.Points,
			&e.TotalSpent, &e.OrderCount, &e.CurrentTier); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetCustomerLoyaltyStatus returns all loyalty program memberships for a customer.
func (r *LoyaltyRepository) GetCustomerLoyaltyStatus(ctx context.Context, tx pgx.Tx, customerID uuid.UUID) ([]model.CustomerLoyaltyWithProgram, error) {
	rows, err := tx.Query(ctx,
		`SELECT cl.tenant_id, cl.customer_id, cl.program_id, cl.points_balance,
		        cl.total_points_earned, cl.total_points_redeemed, cl.current_tier,
		        cl.total_spent, cl.order_count, cl.last_activity_at,
		        cl.created_at, cl.updated_at,
		        lp.name, lp.program_type
		 FROM customer_loyalty cl
		 JOIN loyalty_programs lp ON lp.id = cl.program_id
		 WHERE cl.customer_id = $1
		 ORDER BY lp.name`,
		customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get customer loyalty status: %w", err)
	}
	defer rows.Close()

	var results []model.CustomerLoyaltyWithProgram
	for rows.Next() {
		var r model.CustomerLoyaltyWithProgram
		if err := rows.Scan(
			&r.TenantID, &r.CustomerID, &r.ProgramID, &r.PointsBalance,
			&r.TotalPointsEarned, &r.TotalPointsRedeemed, &r.CurrentTier,
			&r.TotalSpent, &r.OrderCount, &r.LastActivityAt,
			&r.CreatedAt, &r.UpdatedAt,
			&r.ProgramName, &r.ProgramType,
		); err != nil {
			return nil, fmt.Errorf("scan customer loyalty status: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
