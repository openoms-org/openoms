package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrRepricingRuleNotFound = errors.New("repricing rule not found")
)

type RepricingService struct {
	repricingRepo repository.RepricingRepo
	productRepo   repository.ProductRepo
	auditRepo     repository.AuditRepo
	pool          *pgxpool.Pool
	logger        *slog.Logger
}

func NewRepricingService(
	repricingRepo repository.RepricingRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) *RepricingService {
	return &RepricingService{
		repricingRepo: repricingRepo,
		productRepo:   productRepo,
		auditRepo:     auditRepo,
		pool:          pool,
		logger:        logger,
	}
}

func (s *RepricingService) List(ctx context.Context, tenantID uuid.UUID, filter model.RepricingRuleListFilter) (model.ListResponse[model.RepricingRule], error) {
	var resp model.ListResponse[model.RepricingRule]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rules, total, err := s.repricingRepo.ListRules(ctx, tx, filter)
		if err != nil {
			return err
		}
		if rules == nil {
			rules = []model.RepricingRule{}
		}
		resp = model.ListResponse[model.RepricingRule]{
			Items:  rules,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *RepricingService) Get(ctx context.Context, tenantID, ruleID uuid.UUID) (*model.RepricingRule, error) {
	var rule *model.RepricingRule
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rule, err = s.repricingRepo.FindRuleByID(ctx, tx, ruleID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrRepricingRuleNotFound
	}
	return rule, nil
}

func (s *RepricingService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateRepricingRuleRequest, actorID uuid.UUID, ip string) (*model.RepricingRule, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	params := req.Params
	if params == nil {
		params = json.RawMessage("{}")
	}
	channels := req.Channels
	if channels == nil {
		channels = json.RawMessage(`["internal"]`)
	}

	rule := &model.RepricingRule{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       req.Name,
		Status:     "active",
		Strategy:   req.Strategy,
		Priority:   req.Priority,
		ScopeType:  req.ScopeType,
		ScopeValue: req.ScopeValue,
		Params:     params,
		MinPrice:   req.MinPrice,
		MaxPrice:   req.MaxPrice,
		Channels:   channels,
		CreatedBy:  &actorID,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.repricingRepo.CreateRule(ctx, tx, rule); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "repricing_rule.created",
			EntityType: "repricing_rule",
			EntityID:   rule.ID,
			Changes:    map[string]string{"name": rule.Name, "strategy": rule.Strategy},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *RepricingService) Update(ctx context.Context, tenantID, ruleID uuid.UUID, req model.UpdateRepricingRuleRequest, actorID uuid.UUID, ip string) (*model.RepricingRule, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var rule *model.RepricingRule
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.repricingRepo.FindRuleByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRepricingRuleNotFound
		}

		if err := s.repricingRepo.UpdateRule(ctx, tx, ruleID, req); err != nil {
			return err
		}

		rule, err = s.repricingRepo.FindRuleByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "repricing_rule.updated",
			EntityType: "repricing_rule",
			EntityID:   ruleID,
			IPAddress:  ip,
		})
	})
	return rule, err
}

func (s *RepricingService) Delete(ctx context.Context, tenantID, ruleID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.repricingRepo.FindRuleByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRepricingRuleNotFound
		}

		if err := s.repricingRepo.DeleteRule(ctx, tx, ruleID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "repricing_rule.deleted",
			EntityType: "repricing_rule",
			EntityID:   ruleID,
			Changes:    map[string]string{"name": existing.Name},
			IPAddress:  ip,
		})
	})
}

// ApplyRules executes all active repricing rules for a tenant.
func (s *RepricingService) ApplyRules(ctx context.Context, tenantID uuid.UUID) (*model.ApplyResult, error) {
	result := &model.ApplyResult{}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rules, err := s.repricingRepo.ListActiveRules(ctx, tx)
		if err != nil {
			return err
		}
		result.RulesEvaluated = len(rules)

		for _, rule := range rules {
			affected, changes, applyErr := s.applyRule(ctx, tx, tenantID, rule)
			if applyErr != nil {
				s.logger.Error("repricing: apply rule failed",
					"rule_id", rule.ID,
					"tenant_id", tenantID,
					"error", applyErr,
				)
				continue
			}
			result.ProductsAffected += affected
			result.PriceChanges += changes

			if err := s.repricingRepo.UpdateRuleApplied(ctx, tx, rule.ID, affected); err != nil {
				s.logger.Error("repricing: update rule applied stats failed",
					"rule_id", rule.ID,
					"error", err,
				)
			}
		}
		return nil
	})
	return result, err
}

func (s *RepricingService) applyRule(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, rule model.RepricingRule) (affected, changes int, err error) {
	products, err := s.getMatchingProducts(ctx, tx, rule)
	if err != nil {
		return 0, 0, err
	}
	affected = len(products)

	for _, product := range products {
		newPrice, reason, skip := s.calculatePrice(product, rule)
		if skip {
			continue
		}

		// Clamp to bounds
		newPrice = s.clampPrice(newPrice, rule.MinPrice, rule.MaxPrice)

		// Round to 2 decimals
		newPrice = math.Round(newPrice*100) / 100

		if newPrice == product.Price {
			continue
		}
		if newPrice <= 0 {
			continue
		}

		// Update product price
		priceVal := newPrice
		updateReq := model.UpdateProductRequest{Price: &priceVal}
		if err := s.productRepo.Update(ctx, tx, product.ID, updateReq); err != nil {
			s.logger.Error("repricing: update product price failed",
				"product_id", product.ID,
				"error", err,
			)
			continue
		}

		// Log the change
		logEntry := &model.RepricingLog{
			ID:        uuid.New(),
			TenantID:  tenantID,
			RuleID:    rule.ID,
			ProductID: product.ID,
			OldPrice:  product.Price,
			NewPrice:  newPrice,
			Reason:    &reason,
			Channel:   "internal",
		}
		if err := s.repricingRepo.CreateLog(ctx, tx, logEntry); err != nil {
			s.logger.Error("repricing: create log failed",
				"product_id", product.ID,
				"error", err,
			)
			continue
		}

		changes++
	}

	return affected, changes, nil
}

func (s *RepricingService) getMatchingProducts(ctx context.Context, tx pgx.Tx, rule model.RepricingRule) ([]model.Product, error) {
	switch rule.ScopeType {
	case "all":
		products, _, err := s.productRepo.List(ctx, tx, model.ProductListFilter{
			PaginationParams: model.PaginationParams{Limit: 10000, Offset: 0},
		})
		return products, err

	case "category":
		var category string
		if err := json.Unmarshal(rule.ScopeValue, &category); err != nil {
			return nil, fmt.Errorf("parse category scope: %w", err)
		}
		products, _, err := s.productRepo.List(ctx, tx, model.ProductListFilter{
			Category:         &category,
			PaginationParams: model.PaginationParams{Limit: 10000, Offset: 0},
		})
		return products, err

	case "tag":
		var tag string
		if err := json.Unmarshal(rule.ScopeValue, &tag); err != nil {
			return nil, fmt.Errorf("parse tag scope: %w", err)
		}
		products, _, err := s.productRepo.List(ctx, tx, model.ProductListFilter{
			Tag:              &tag,
			PaginationParams: model.PaginationParams{Limit: 10000, Offset: 0},
		})
		return products, err

	case "product_ids":
		var ids []string
		if err := json.Unmarshal(rule.ScopeValue, &ids); err != nil {
			return nil, fmt.Errorf("parse product_ids scope: %w", err)
		}
		var products []model.Product
		for _, idStr := range ids {
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			p, err := s.productRepo.FindByID(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			if p != nil {
				products = append(products, *p)
			}
		}
		return products, nil

	default:
		return nil, fmt.Errorf("unknown scope type: %s", rule.ScopeType)
	}
}

func (s *RepricingService) calculatePrice(product model.Product, rule model.RepricingRule) (newPrice float64, reason string, skip bool) {
	switch rule.Strategy {
	case "margin":
		return s.applyMarginStrategy(product, rule)
	case "stock_based":
		return s.applyStockStrategy(product, rule)
	case "time_based":
		return s.applyTimeStrategy(product, rule)
	case "competitive":
		return 0, "", true // Placeholder: competitive data not yet available
	default:
		return 0, "", true
	}
}

func (s *RepricingService) applyMarginStrategy(product model.Product, rule model.RepricingRule) (float64, string, bool) {
	var params model.MarginParams
	if err := json.Unmarshal(rule.Params, &params); err != nil {
		s.logger.Error("repricing: parse margin params", "error", err)
		return 0, "", true
	}

	// Get cost from product metadata
	costPrice := s.getCostFromMetadata(product, params.CostField)
	if costPrice <= 0 {
		return 0, "", true // No cost data, skip
	}

	newPrice := costPrice * (1 + params.TargetMarginPct/100)
	reason := fmt.Sprintf("Margin strategy: cost=%.2f, target margin=%.1f%%", costPrice, params.TargetMarginPct)
	return newPrice, reason, false
}

func (s *RepricingService) getCostFromMetadata(product model.Product, costField string) float64 {
	if costField == "" {
		costField = "supplier_cost"
	}
	if product.Metadata == nil {
		return 0
	}
	var meta map[string]any
	if err := json.Unmarshal(product.Metadata, &meta); err != nil {
		return 0
	}
	val, ok := meta[costField]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func (s *RepricingService) applyStockStrategy(product model.Product, rule model.RepricingRule) (float64, string, bool) {
	var params model.StockBasedParams
	if err := json.Unmarshal(rule.Params, &params); err != nil {
		s.logger.Error("repricing: parse stock params", "error", err)
		return 0, "", true
	}

	qty := product.StockQuantity

	if params.LowStockThreshold > 0 && qty < params.LowStockThreshold {
		newPrice := product.Price * (1 + params.LowStockMarkupPct/100)
		reason := fmt.Sprintf("Stock-based: low stock (%d < %d), markup %.1f%%", qty, params.LowStockThreshold, params.LowStockMarkupPct)
		return newPrice, reason, false
	}

	if params.HighStockThreshold > 0 && qty > params.HighStockThreshold {
		newPrice := product.Price * (1 - params.HighStockDiscountPct/100)
		reason := fmt.Sprintf("Stock-based: high stock (%d > %d), discount %.1f%%", qty, params.HighStockThreshold, params.HighStockDiscountPct)
		return newPrice, reason, false
	}

	return 0, "", true // Stock in normal range, no change
}

func (s *RepricingService) applyTimeStrategy(product model.Product, rule model.RepricingRule) (float64, string, bool) {
	var params model.TimeBasedParams
	if err := json.Unmarshal(rule.Params, &params); err != nil {
		s.logger.Error("repricing: parse time params", "error", err)
		return 0, "", true
	}

	now := time.Now()
	currentDay := strings.ToLower(now.Weekday().String()[:3])
	currentHour := now.Hour()

	for _, schedule := range params.Schedule {
		days := strings.Split(strings.ToLower(schedule.Days), ",")
		dayMatch := false
		for _, d := range days {
			if strings.TrimSpace(d) == currentDay {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			continue
		}

		// Check hours if specified
		if schedule.Hours != "" {
			parts := strings.Split(schedule.Hours, "-")
			if len(parts) == 2 {
				startHour := 0
				endHour := 23
				fmt.Sscanf(parts[0], "%d", &startHour)
				fmt.Sscanf(parts[1], "%d", &endHour)
				if currentHour < startHour || currentHour > endHour {
					continue
				}
			}
		}

		newPrice := product.Price * (1 - schedule.DiscountPct/100)
		reason := fmt.Sprintf("Time-based: day=%s, discount %.1f%%", currentDay, schedule.DiscountPct)
		return newPrice, reason, false
	}

	return 0, "", true // No matching schedule
}

func (s *RepricingService) clampPrice(price float64, minPrice, maxPrice *float64) float64 {
	if minPrice != nil && price < *minPrice {
		price = *minPrice
	}
	if maxPrice != nil && price > *maxPrice {
		price = *maxPrice
	}
	return price
}

// SimulateRule performs a dry-run of a repricing rule without applying changes.
func (s *RepricingService) SimulateRule(ctx context.Context, tenantID, ruleID uuid.UUID) ([]model.SimulationResult, error) {
	var results []model.SimulationResult

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rule, err := s.repricingRepo.FindRuleByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}
		if rule == nil {
			return ErrRepricingRuleNotFound
		}

		if rule.Strategy == "competitive" {
			results = []model.SimulationResult{}
			return nil
		}

		products, err := s.getMatchingProducts(ctx, tx, *rule)
		if err != nil {
			return err
		}

		for _, product := range products {
			newPrice, reason, skip := s.calculatePrice(product, *rule)
			if skip {
				continue
			}
			newPrice = s.clampPrice(newPrice, rule.MinPrice, rule.MaxPrice)
			newPrice = math.Round(newPrice*100) / 100

			if newPrice == product.Price || newPrice <= 0 {
				continue
			}

			changePct := 0.0
			if product.Price > 0 {
				changePct = math.Round(((newPrice-product.Price)/product.Price*100)*100) / 100
			}

			results = append(results, model.SimulationResult{
				ProductID:   product.ID,
				ProductName: product.Name,
				SKU:         product.SKU,
				OldPrice:    product.Price,
				NewPrice:    newPrice,
				ChangePct:   changePct,
				Reason:      reason,
			})
		}
		return nil
	})
	if results == nil {
		results = []model.SimulationResult{}
	}
	return results, err
}

// GetSummary returns dashboard-level repricing stats.
func (s *RepricingService) GetSummary(ctx context.Context, tenantID uuid.UUID) (*model.RepricingSummary, error) {
	var summary *model.RepricingSummary
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		summary, err = s.repricingRepo.GetSummary(ctx, tx)
		return err
	})
	return summary, err
}

// GetLogForProduct returns price history for a product.
func (s *RepricingService) GetLogForProduct(ctx context.Context, tenantID, productID uuid.UUID, limit, offset int) ([]model.RepricingLog, int, error) {
	var logs []model.RepricingLog
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		logs, total, err = s.repricingRepo.ListLogByProduct(ctx, tx, productID, limit, offset)
		return err
	})
	if logs == nil {
		logs = []model.RepricingLog{}
	}
	return logs, total, err
}

// GetLogForRule returns price changes for a rule.
func (s *RepricingService) GetLogForRule(ctx context.Context, tenantID, ruleID uuid.UUID, limit, offset int) ([]model.RepricingLog, int, error) {
	var logs []model.RepricingLog
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		logs, total, err = s.repricingRepo.ListLogByRule(ctx, tx, ruleID, limit, offset)
		return err
	})
	if logs == nil {
		logs = []model.RepricingLog{}
	}
	return logs, total, err
}

// ListLog returns all repricing log entries.
func (s *RepricingService) ListLog(ctx context.Context, tenantID uuid.UUID, limit, offset int) (model.ListResponse[model.RepricingLog], error) {
	var resp model.ListResponse[model.RepricingLog]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		logs, total, err := s.repricingRepo.ListLog(ctx, tx, limit, offset)
		if err != nil {
			return err
		}
		if logs == nil {
			logs = []model.RepricingLog{}
		}
		resp = model.ListResponse[model.RepricingLog]{
			Items:  logs,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		}
		return nil
	})
	return resp, err
}
