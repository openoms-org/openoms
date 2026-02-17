package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrSegmentNotFound = errors.New("segment not found")
	ErrMemberNotFound  = errors.New("member not found in segment")
)

type SegmentService struct {
	segmentRepo *repository.CustomerSegmentRepository
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
	logger      *slog.Logger
}

func NewSegmentService(
	segmentRepo *repository.CustomerSegmentRepository,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) *SegmentService {
	return &SegmentService{
		segmentRepo: segmentRepo,
		auditRepo:   auditRepo,
		pool:        pool,
		logger:      logger,
	}
}

func (s *SegmentService) List(ctx context.Context, tenantID uuid.UUID, filter model.SegmentListFilter) (model.ListResponse[model.CustomerSegment], error) {
	var resp model.ListResponse[model.CustomerSegment]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		segments, total, err := s.segmentRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if segments == nil {
			segments = []model.CustomerSegment{}
		}
		resp = model.ListResponse[model.CustomerSegment]{
			Items:  segments,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *SegmentService) Get(ctx context.Context, tenantID, segmentID uuid.UUID) (*model.CustomerSegment, error) {
	var segment *model.CustomerSegment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		segment, err = s.segmentRepo.FindByID(ctx, tx, segmentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if segment == nil {
		return nil, ErrSegmentNotFound
	}
	return segment, nil
}

func (s *SegmentService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateSegmentRequest, actorID uuid.UUID, ip string) (*model.CustomerSegment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	segment := &model.CustomerSegment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        model.StripHTMLTags(req.Name),
		Description: req.Description,
		Color:       req.Color,
		SegmentType: req.SegmentType,
		Rules:       req.Rules,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.segmentRepo.Create(ctx, tx, segment); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "segment.created",
			EntityType: "segment",
			EntityID:   segment.ID,
			Changes:    map[string]string{"name": segment.Name},
			IPAddress:  ip,
		})
	})
	return segment, err
}

func (s *SegmentService) Update(ctx context.Context, tenantID, segmentID uuid.UUID, req model.UpdateSegmentRequest, actorID uuid.UUID, ip string) (*model.CustomerSegment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var segment *model.CustomerSegment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.segmentRepo.FindByID(ctx, tx, segmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSegmentNotFound
		}

		if err := s.segmentRepo.Update(ctx, tx, segmentID, req); err != nil {
			return err
		}

		segment, err = s.segmentRepo.FindByID(ctx, tx, segmentID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "segment.updated",
			EntityType: "segment",
			EntityID:   segmentID,
			IPAddress:  ip,
		})
	})
	return segment, err
}

func (s *SegmentService) Delete(ctx context.Context, tenantID, segmentID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.segmentRepo.FindByID(ctx, tx, segmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSegmentNotFound
		}

		if err := s.segmentRepo.Delete(ctx, tx, segmentID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "segment.deleted",
			EntityType: "segment",
			EntityID:   segmentID,
			Changes:    map[string]string{"name": existing.Name},
			IPAddress:  ip,
		})
	})
}

func (s *SegmentService) ListMembers(ctx context.Context, tenantID, segmentID uuid.UUID, limit, offset int) (model.ListResponse[model.SegmentMember], error) {
	var resp model.ListResponse[model.SegmentMember]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.segmentRepo.FindByID(ctx, tx, segmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSegmentNotFound
		}

		members, total, err := s.segmentRepo.ListMembers(ctx, tx, segmentID, limit, offset)
		if err != nil {
			return err
		}
		if members == nil {
			members = []model.SegmentMember{}
		}
		resp = model.ListResponse[model.SegmentMember]{
			Items:  members,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		}
		return nil
	})
	return resp, err
}

func (s *SegmentService) AddMember(ctx context.Context, tenantID, segmentID uuid.UUID, req model.AddSegmentMemberRequest, actorID uuid.UUID, ip string) error {
	if err := req.Validate(); err != nil {
		return NewValidationError(err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.segmentRepo.FindByID(ctx, tx, segmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSegmentNotFound
		}

		if err := s.segmentRepo.AddMember(ctx, tx, tenantID, segmentID, req.CustomerID); err != nil {
			return err
		}

		if err := s.segmentRepo.UpdateCustomerCount(ctx, tx, segmentID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "segment.member_added",
			EntityType: "segment",
			EntityID:   segmentID,
			Changes:    map[string]string{"customer_id": req.CustomerID.String()},
			IPAddress:  ip,
		})
	})
}

func (s *SegmentService) RemoveMember(ctx context.Context, tenantID, segmentID, customerID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.segmentRepo.FindByID(ctx, tx, segmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSegmentNotFound
		}

		if err := s.segmentRepo.RemoveMember(ctx, tx, segmentID, customerID); err != nil {
			return err
		}

		if err := s.segmentRepo.UpdateCustomerCount(ctx, tx, segmentID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "segment.member_removed",
			EntityType: "segment",
			EntityID:   segmentID,
			Changes:    map[string]string{"customer_id": customerID.String()},
			IPAddress:  ip,
		})
	})
}

// GetCustomerSegments returns all segments a customer belongs to.
func (s *SegmentService) GetCustomerSegments(ctx context.Context, tenantID, customerID uuid.UUID) ([]model.CustomerSegment, error) {
	var segments []model.CustomerSegment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		segments, err = s.segmentRepo.GetCustomerSegments(ctx, tx, customerID)
		return err
	})
	if segments == nil {
		segments = []model.CustomerSegment{}
	}
	return segments, err
}

// RunRFMAnalysis calculates RFM scores for all customers and returns results.
func (s *SegmentService) RunRFMAnalysis(ctx context.Context, tenantID uuid.UUID) ([]model.CustomerRFM, error) {
	var results []model.CustomerRFM

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rawData, err := s.segmentRepo.GetRFMData(ctx, tx)
		if err != nil {
			return err
		}

		if len(rawData) == 0 {
			return nil
		}

		// Calculate quintile breakpoints for each dimension
		recencyValues := make([]float64, len(rawData))
		frequencyValues := make([]float64, len(rawData))
		monetaryValues := make([]float64, len(rawData))
		for i, d := range rawData {
			recencyValues[i] = float64(d.DaysSince)
			frequencyValues[i] = float64(d.OrderCount)
			monetaryValues[i] = d.TotalSpent
		}

		recencyBreaks := quintileBreaks(recencyValues)
		frequencyBreaks := quintileBreaks(frequencyValues)
		monetaryBreaks := quintileBreaks(monetaryValues)

		for i := range rawData {
			// Recency: lower days = higher score (inverse)
			rawData[i].RFM.Recency = min(max(6-quintileScore(float64(rawData[i].DaysSince), recencyBreaks), 1), 5)

			// Frequency: more orders = higher score
			rawData[i].RFM.Frequency = quintileScore(float64(rawData[i].OrderCount), frequencyBreaks)

			// Monetary: more spent = higher score
			rawData[i].RFM.Monetary = quintileScore(rawData[i].TotalSpent, monetaryBreaks)

			// Assign segment label
			rawData[i].SegmentLabel = rfmSegmentLabel(rawData[i].RFM)
		}

		results = rawData

		// Auto-assign to RFM segments
		rfmSegments := map[string][]model.CustomerRFM{
			"Champions":           {},
			"Lojalni klienci":     {},
			"Potencjalni lojalni": {},
			"Zagrożeni":           {},
			"Utraceni":            {},
			"Pozostali":           {},
		}

		for _, r := range results {
			rfmSegments[r.SegmentLabel] = append(rfmSegments[r.SegmentLabel], r)
		}

		// Colors for auto-segments
		segmentColors := map[string]string{
			"Champions":           "#10b981",
			"Lojalni klienci":     "#3b82f6",
			"Potencjalni lojalni": "#8b5cf6",
			"Zagrożeni":           "#f59e0b",
			"Utraceni":            "#ef4444",
			"Pozostali":           "#6b7280",
		}

		for segName, customers := range rfmSegments {
			// Find or create the auto-segment
			seg, err := s.segmentRepo.FindByTypeAndName(ctx, tx, "rfm_auto", segName)
			if err != nil {
				return err
			}
			if seg == nil {
				desc := rfmSegmentDescription(segName)
				seg = &model.CustomerSegment{
					ID:          uuid.New(),
					TenantID:    tenantID,
					Name:        segName,
					Description: &desc,
					Color:       segmentColors[segName],
					SegmentType: "rfm_auto",
				}
				if err := s.segmentRepo.Create(ctx, tx, seg); err != nil {
					return err
				}
			}

			// Clear and re-populate members
			if err := s.segmentRepo.ClearSegmentMembers(ctx, tx, seg.ID); err != nil {
				return err
			}
			for _, c := range customers {
				if err := s.segmentRepo.AddMember(ctx, tx, tenantID, seg.ID, c.CustomerID); err != nil {
					return err
				}
			}
			if err := s.segmentRepo.UpdateCustomerCount(ctx, tx, seg.ID); err != nil {
				return err
			}
		}

		return nil
	})

	if results == nil {
		results = []model.CustomerRFM{}
	}
	return results, err
}

// quintileBreaks returns 4 breakpoints dividing sorted values into 5 equal groups.
func quintileBreaks(values []float64) [4]float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n == 0 {
		return [4]float64{0, 0, 0, 0}
	}

	var breaks [4]float64
	for i := range 4 {
		idx := int(math.Round(float64((i+1)*n) / 5.0))
		if idx >= n {
			idx = n - 1
		}
		breaks[i] = sorted[idx]
	}
	return breaks
}

// quintileScore returns 1-5 based on where value falls in the breaks.
func quintileScore(value float64, breaks [4]float64) int {
	for i := range 4 {
		if value <= breaks[i] {
			return i + 1
		}
	}
	return 5
}

// rfmSegmentLabel assigns a human-readable label based on RFM scores.
func rfmSegmentLabel(rfm model.RFMScores) string {
	r, f := rfm.Recency, rfm.Frequency

	// Champions: high recency, high frequency, high monetary
	if r >= 4 && f >= 4 {
		return "Champions"
	}
	// Loyal: frequency >= 4
	if f >= 4 {
		return "Lojalni klienci"
	}
	// Potential loyalists: high recency, moderate frequency
	if r >= 4 && f >= 2 {
		return "Potencjalni lojalni"
	}
	// At risk: low recency, moderate+ frequency
	if r <= 2 && f >= 3 {
		return "Zagrożeni"
	}
	// Lost: low recency, low frequency
	if r == 1 && f == 1 {
		return "Utraceni"
	}
	return "Pozostali"
}

func rfmSegmentDescription(name string) string {
	descriptions := map[string]string{
		"Champions":           "Najlepsi klienci — kupują często, ostatnio i za duże kwoty",
		"Lojalni klienci":     "Klienci kupujący regularnie",
		"Potencjalni lojalni": "Ostatnio kupili, mogą stać się lojalnymi",
		"Zagrożeni":           "Kiedyś kupowali regularnie, ale dawno ich nie było",
		"Utraceni":            "Dawno nie kupowali i kupowali rzadko",
		"Pozostali":           "Klienci nie pasujący do innych segmentów",
	}
	if d, ok := descriptions[name]; ok {
		return d
	}
	return ""
}

// RefreshRuleBasedSegments evaluates rule-based segments and updates membership.
func (s *SegmentService) RefreshRuleBasedSegments(ctx context.Context, tenantID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		filter := model.SegmentListFilter{
			PaginationParams: model.PaginationParams{Limit: 100, Offset: 0},
		}
		segments, _, err := s.segmentRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}

		for _, seg := range segments {
			if seg.SegmentType != "rule_based" || seg.Rules == nil {
				continue
			}

			var rules model.SegmentRules
			if err := json.Unmarshal(seg.Rules, &rules); err != nil {
				s.logger.Warn("invalid segment rules", "segment_id", seg.ID, "error", err)
				continue
			}

			// Build query based on rules
			query := `SELECT c.id FROM customers c
				 LEFT JOIN (
					SELECT customer_id, COUNT(*) AS order_count,
						   MAX(created_at) AS last_order_at,
						   COALESCE(SUM(total_amount), 0) AS total_spent
					FROM orders GROUP BY customer_id
				 ) o ON o.customer_id = c.id
				 WHERE 1=1`
			var args []any
			argIdx := 1

			if rules.MinOrders != nil {
				query += " AND COALESCE(o.order_count, 0) >= $" + itoa(argIdx)
				args = append(args, *rules.MinOrders)
				argIdx++
			}
			if rules.MaxOrders != nil {
				query += " AND COALESCE(o.order_count, 0) <= $" + itoa(argIdx)
				args = append(args, *rules.MaxOrders)
				argIdx++
			}
			if rules.MinTotalSpent != nil {
				query += " AND COALESCE(o.total_spent, 0) >= $" + itoa(argIdx)
				args = append(args, *rules.MinTotalSpent)
				argIdx++
			}
			if rules.MaxTotalSpent != nil {
				query += " AND COALESCE(o.total_spent, 0) <= $" + itoa(argIdx)
				args = append(args, *rules.MaxTotalSpent)
				argIdx++
			}
			if rules.LastOrderWithinDay != nil {
				query += " AND o.last_order_at >= NOW() - INTERVAL '1 day' * $" + itoa(argIdx)
				args = append(args, *rules.LastOrderWithinDay)
				argIdx++
			}

			rows, err := tx.Query(ctx, query, args...)
			if err != nil {
				s.logger.Error("rule-based segment query failed", "segment_id", seg.ID, "error", err)
				continue
			}

			var customerIDs []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return err
				}
				customerIDs = append(customerIDs, id)
			}
			rows.Close()

			// Clear and re-populate
			if err := s.segmentRepo.ClearSegmentMembers(ctx, tx, seg.ID); err != nil {
				return err
			}
			for _, cid := range customerIDs {
				if err := s.segmentRepo.AddMember(ctx, tx, tenantID, seg.ID, cid); err != nil {
					return err
				}
			}
			if err := s.segmentRepo.UpdateCustomerCount(ctx, tx, seg.ID); err != nil {
				return err
			}
		}

		return nil
	})
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
