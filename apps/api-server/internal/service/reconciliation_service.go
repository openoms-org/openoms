package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strconv"
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
	// ErrSettlementNotFound is returned when a payment settlement does not exist.
	ErrSettlementNotFound  = errors.New("settlement not found")
	// ErrTransactionNotFound is returned when a payment transaction does not exist.
	ErrTransactionNotFound = errors.New("transaction not found")
	// ErrAlreadyMatched is returned when a transaction has already been reconciled.
	ErrAlreadyMatched      = errors.New("transaction is already matched")
)

// ReconciliationService handles payment reconciliation and transaction matching.
type ReconciliationService struct {
	paymentRepo *repository.PaymentRepository
	orderRepo   repository.OrderRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

// NewReconciliationService creates a new ReconciliationService.
func NewReconciliationService(
	paymentRepo *repository.PaymentRepository,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *ReconciliationService {
	return &ReconciliationService{
		paymentRepo: paymentRepo,
		orderRepo:   orderRepo,
		auditRepo:   auditRepo,
		pool:        pool,
	}
}

// ImportSettlement creates a settlement with its transactions.
func (s *ReconciliationService) ImportSettlement(ctx context.Context, tenantID uuid.UUID, req model.CreateSettlementRequest) (*model.PaymentSettlement, error) {
	currency := req.Currency
	if currency == "" {
		currency = "PLN"
	}

	settlement := &model.PaymentSettlement{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Provider:       req.Provider,
		SettlementID:   req.SettlementID,
		SettlementDate: req.SettlementDate,
		TotalAmount:    req.TotalAmount,
		FeeAmount:      req.FeeAmount,
		NetAmount:      req.NetAmount,
		Currency:       currency,
		Status:         "pending",
		Notes:          req.Notes,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.paymentRepo.CreateSettlement(ctx, tx, settlement); err != nil {
			return err
		}

		for _, txReq := range req.Transactions {
			txDate, err := time.Parse(time.RFC3339, txReq.TransactionDate)
			if err != nil {
				txDate, err = time.Parse("2006-01-02", txReq.TransactionDate)
				if err != nil {
					return fmt.Errorf("invalid transaction_date: %s", txReq.TransactionDate)
				}
			}

			txCurrency := txReq.Currency
			if txCurrency == "" {
				txCurrency = currency
			}
			txType := txReq.TransactionType
			if txType == "" {
				txType = "payment"
			}

			txn := &model.PaymentTransaction{
				ID:                    uuid.New(),
				TenantID:              tenantID,
				SettlementID:          &settlement.ID,
				Provider:              req.Provider,
				ExternalTransactionID: txReq.ExternalTransactionID,
				Amount:                txReq.Amount,
				Fee:                   txReq.Fee,
				NetAmount:             txReq.NetAmount,
				Currency:              txCurrency,
				TransactionType:       txType,
				TransactionDate:       txDate,
				MatchStatus:           "unmatched",
			}

			// Dedup: skip if external_transaction_id already exists
			if txReq.ExternalTransactionID != nil && *txReq.ExternalTransactionID != "" {
				existing, err := s.paymentRepo.FindTransactionByExternalID(ctx, tx, req.Provider, *txReq.ExternalTransactionID)
				if err != nil {
					return err
				}
				if existing != nil {
					continue // skip duplicate
				}
			}

			if err := s.paymentRepo.CreateTransaction(ctx, tx, txn); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return settlement, nil
}

// ImportCSV parses a CSV file and creates a settlement with transactions.
func (s *ReconciliationService) ImportCSV(ctx context.Context, tenantID uuid.UUID, provider string, reader io.Reader) (*model.PaymentSettlement, error) {
	csvReader := csv.NewReader(reader)

	// PayU uses semicolons
	if provider == "payu" || provider == "przelewy24" {
		csvReader.Comma = ';'
	}

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, errors.New("CSV file must have a header row and at least one data row")
	}

	header := records[0]
	colMap := buildColumnMap(header)

	var transactions []model.CreateTransactionRequest
	var totalAmount, totalFee, totalNet float64
	var earliestDate, latestDate string

	for i, row := range records[1:] {
		txn, err := parseCSVRow(provider, colMap, row, i+1)
		if err != nil {
			slog.Warn("skipping CSV row", "row", i+1, "error", err)
			continue
		}
		transactions = append(transactions, txn)
		totalAmount += txn.Amount
		totalFee += txn.Fee
		totalNet += txn.NetAmount

		if earliestDate == "" || txn.TransactionDate < earliestDate {
			earliestDate = txn.TransactionDate
		}
		if latestDate == "" || txn.TransactionDate > latestDate {
			latestDate = txn.TransactionDate
		}
	}

	if len(transactions) == 0 {
		return nil, errors.New("no valid transactions found in CSV")
	}

	settlementDate := latestDate
	if len(settlementDate) > 10 {
		settlementDate = settlementDate[:10]
	}

	req := model.CreateSettlementRequest{
		Provider:       provider,
		SettlementDate: settlementDate,
		TotalAmount:    math.Round(totalAmount*100) / 100,
		FeeAmount:      math.Round(totalFee*100) / 100,
		NetAmount:      math.Round(totalNet*100) / 100,
		Currency:       "PLN",
		Transactions:   transactions,
	}

	return s.ImportSettlement(ctx, tenantID, req)
}

// AutoMatch performs automatic matching of transactions within a settlement against orders.
func (s *ReconciliationService) AutoMatch(ctx context.Context, tenantID uuid.UUID, settlementID uuid.UUID) (*model.AutoMatchResponse, error) {
	response := &model.AutoMatchResponse{
		SettlementID: settlementID,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settlement, err := s.paymentRepo.FindSettlementByID(ctx, tx, settlementID)
		if err != nil {
			return err
		}
		if settlement == nil {
			return ErrSettlementNotFound
		}

		transactions, err := s.paymentRepo.FindTransactionsBySettlement(ctx, tx, settlementID)
		if err != nil {
			return err
		}

		for _, txn := range transactions {
			if txn.MatchStatus == "matched" || txn.MatchStatus == "manual_match" {
				response.Results = append(response.Results, model.MatchResult{
					TransactionID: txn.ID,
					OrderID:       txn.OrderID,
					Status:        txn.MatchStatus,
					Notes:         "already matched",
				})
				response.Matched++
				continue
			}

			// Strategy 1: Match by external_transaction_id -> order.external_id
			if txn.ExternalTransactionID != nil && *txn.ExternalTransactionID != "" {
				order, err := s.orderRepo.FindByExternalID(ctx, tx, txn.Provider, *txn.ExternalTransactionID)
				if err == nil && order != nil {
					// Check amount discrepancy
					diff := math.Abs(order.TotalAmount - txn.Amount)
					status := "matched"
					notes := fmt.Sprintf("matched by external_id: %s", *txn.ExternalTransactionID)
					if diff > 0.01 {
						status = "discrepancy"
						notes = fmt.Sprintf("amount discrepancy: order=%.2f, txn=%.2f (diff=%.2f)", order.TotalAmount, txn.Amount, diff)
					}

					orderID := order.ID
					if err := s.paymentRepo.UpdateTransactionMatch(ctx, tx, txn.ID, &orderID, status, &notes); err != nil {
						return err
					}

					result := model.MatchResult{
						TransactionID: txn.ID,
						OrderID:       &orderID,
						Status:        status,
						Notes:         notes,
					}
					response.Results = append(response.Results, result)
					if status == "matched" {
						response.Matched++
					} else {
						response.Discrepancy++
					}
					continue
				}
			}

			// Strategy 2: Fuzzy match by amount + date range (+-1 day)
			matched := false
			dateFrom := txn.TransactionDate.Add(-24 * time.Hour)
			dateTo := txn.TransactionDate.Add(24 * time.Hour)

			// Query orders with matching amount in date range
			filter := model.OrderListFilter{
				PaginationParams: model.PaginationParams{Limit: 100, Offset: 0},
			}
			orders, _, err := s.orderRepo.List(ctx, tx, filter)
			if err != nil {
				return err
			}

			for _, order := range orders {
				if order.OrderedAt == nil {
					continue
				}
				if order.OrderedAt.Before(dateFrom) || order.OrderedAt.After(dateTo) {
					continue
				}
				diff := math.Abs(order.TotalAmount - txn.Amount)
				if diff <= 0.01 {
					// Exact amount match within date range
					orderID := order.ID
					status := "matched"
					notes := fmt.Sprintf("fuzzy match: amount=%.2f, date=%s", txn.Amount, txn.TransactionDate.Format("2006-01-02"))

					if err := s.paymentRepo.UpdateTransactionMatch(ctx, tx, txn.ID, &orderID, status, &notes); err != nil {
						return err
					}

					response.Results = append(response.Results, model.MatchResult{
						TransactionID: txn.ID,
						OrderID:       &orderID,
						Status:        status,
						Notes:         notes,
					})
					response.Matched++
					matched = true
					break
				}
			}

			if !matched {
				notes := "no matching order found"
				response.Results = append(response.Results, model.MatchResult{
					TransactionID: txn.ID,
					Status:        "unmatched",
					Notes:         notes,
				})
				response.Unmatched++
			}
		}

		// Update settlement status based on match results
		newStatus := "matched"
		if response.Unmatched > 0 && response.Matched > 0 {
			newStatus = "partial_match"
		} else if response.Unmatched > 0 {
			newStatus = "unmatched"
		}
		if response.Discrepancy > 0 {
			newStatus = "partial_match"
		}

		return s.paymentRepo.UpdateSettlementStatus(ctx, tx, settlementID, newStatus)
	})

	if err != nil {
		return nil, err
	}
	return response, nil
}

// ManualMatch manually links a transaction to an order.
func (s *ReconciliationService) ManualMatch(ctx context.Context, tenantID uuid.UUID, transactionID uuid.UUID, req model.ManualMatchRequest) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		txn, err := s.paymentRepo.FindTransactionByID(ctx, tx, transactionID)
		if err != nil {
			return err
		}
		if txn == nil {
			return ErrTransactionNotFound
		}

		// Verify order exists
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFound
		}

		notes := "manually matched"
		if req.Notes != nil {
			notes = *req.Notes
		}

		// Check for amount discrepancy
		diff := math.Abs(order.TotalAmount - txn.Amount)
		status := "manual_match"
		if diff > 0.01 {
			notes += fmt.Sprintf(" (amount discrepancy: order=%.2f, txn=%.2f)", order.TotalAmount, txn.Amount)
		}

		return s.paymentRepo.UpdateTransactionMatch(ctx, tx, transactionID, &req.OrderID, status, &notes)
	})
}

// Unmatch removes a match from a transaction.
func (s *ReconciliationService) Unmatch(ctx context.Context, tenantID uuid.UUID, transactionID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		txn, err := s.paymentRepo.FindTransactionByID(ctx, tx, transactionID)
		if err != nil {
			return err
		}
		if txn == nil {
			return ErrTransactionNotFound
		}

		notes := "match removed"
		return s.paymentRepo.UpdateTransactionMatch(ctx, tx, transactionID, nil, "unmatched", &notes)
	})
}

// GetSummary returns reconciliation summary statistics.
func (s *ReconciliationService) GetSummary(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo *string) (*model.ReconciliationSummary, error) {
	var summary *model.ReconciliationSummary
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		summary, err = s.paymentRepo.GetReconciliationSummary(ctx, tx, dateFrom, dateTo)
		return err
	})
	return summary, err
}

// ListSettlements lists settlements with pagination and filters.
func (s *ReconciliationService) ListSettlements(ctx context.Context, tenantID uuid.UUID, filter model.SettlementListFilter) (model.ListResponse[model.PaymentSettlement], error) {
	var resp model.ListResponse[model.PaymentSettlement]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settlements, total, err := s.paymentRepo.ListSettlements(ctx, tx, filter)
		if err != nil {
			return err
		}
		if settlements == nil {
			settlements = []model.PaymentSettlement{}
		}
		resp = model.ListResponse[model.PaymentSettlement]{
			Items:  settlements,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// GetSettlement returns a settlement by ID with its transactions.
func (s *ReconciliationService) GetSettlement(ctx context.Context, tenantID uuid.UUID, settlementID uuid.UUID) (*model.PaymentSettlementWithTransactions, error) {
	var result *model.PaymentSettlementWithTransactions
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settlement, err := s.paymentRepo.FindSettlementByID(ctx, tx, settlementID)
		if err != nil {
			return err
		}
		if settlement == nil {
			return ErrSettlementNotFound
		}

		transactions, err := s.paymentRepo.FindTransactionsBySettlement(ctx, tx, settlementID)
		if err != nil {
			return err
		}
		if transactions == nil {
			transactions = []model.PaymentTransaction{}
		}

		result = &model.PaymentSettlementWithTransactions{
			PaymentSettlement: *settlement,
			Transactions:      transactions,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListTransactions lists transactions with pagination and filters.
func (s *ReconciliationService) ListTransactions(ctx context.Context, tenantID uuid.UUID, filter model.TransactionListFilter) (model.ListResponse[model.PaymentTransaction], error) {
	var resp model.ListResponse[model.PaymentTransaction]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		transactions, total, err := s.paymentRepo.ListTransactions(ctx, tx, filter)
		if err != nil {
			return err
		}
		if transactions == nil {
			transactions = []model.PaymentTransaction{}
		}
		resp = model.ListResponse[model.PaymentTransaction]{
			Items:  transactions,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// --- CSV Parsing Helpers ---

// buildColumnMap normalizes CSV header names to lookup keys.
func buildColumnMap(header []string) map[string]int {
	colMap := make(map[string]int)
	for i, h := range header {
		normalized := strings.ToLower(strings.TrimSpace(h))
		// Remove BOM if present
		normalized = strings.TrimPrefix(normalized, "\ufeff")
		colMap[normalized] = i
	}
	return colMap
}

// parseCSVRow parses a single CSV row into a CreateTransactionRequest based on provider format.
func parseCSVRow(provider string, colMap map[string]int, row []string, rowNum int) (model.CreateTransactionRequest, error) {
	switch provider {
	case "payu":
		return parsePayURow(colMap, row, rowNum)
	case "przelewy24":
		return parsePrzelewy24Row(colMap, row, rowNum)
	default:
		return parseGenericRow(colMap, row, rowNum)
	}
}

func parsePayURow(colMap map[string]int, row []string, rowNum int) (model.CreateTransactionRequest, error) {
	var txn model.CreateTransactionRequest

	// PayU CSV columns (Polish headers): id transakcji, numer zamówienia, kwota, prowizja, data, status
	txIDIdx := findColumn(colMap, "id transakcji", "transaction_id", "id")
	amountIdx := findColumn(colMap, "kwota", "amount", "wartosc")
	feeIdx := findColumn(colMap, "prowizja", "fee", "commission")
	dateIdx := findColumn(colMap, "data", "date", "data transakcji")

	if txIDIdx >= 0 && txIDIdx < len(row) {
		val := strings.TrimSpace(row[txIDIdx])
		if val != "" {
			txn.ExternalTransactionID = &val
		}
	}

	amount, err := parseDecimal(row, amountIdx)
	if err != nil {
		return txn, fmt.Errorf("row %d: invalid amount: %w", rowNum, err)
	}
	txn.Amount = amount

	fee, _ := parseDecimal(row, feeIdx)
	txn.Fee = math.Abs(fee)
	txn.NetAmount = math.Round((amount-math.Abs(fee))*100) / 100

	date, err := parseDate(row, dateIdx)
	if err != nil {
		return txn, fmt.Errorf("row %d: invalid date: %w", rowNum, err)
	}
	txn.TransactionDate = date

	txn.TransactionType = "payment"
	if amount < 0 {
		txn.TransactionType = "refund"
	}

	return txn, nil
}

func parsePrzelewy24Row(colMap map[string]int, row []string, rowNum int) (model.CreateTransactionRequest, error) {
	var txn model.CreateTransactionRequest

	txIDIdx := findColumn(colMap, "id transakcji", "numer transakcji", "transaction_id")
	amountIdx := findColumn(colMap, "kwota", "amount", "wartosc")
	feeIdx := findColumn(colMap, "prowizja", "fee", "commission")
	dateIdx := findColumn(colMap, "data", "date", "data transakcji")

	if txIDIdx >= 0 && txIDIdx < len(row) {
		val := strings.TrimSpace(row[txIDIdx])
		if val != "" {
			txn.ExternalTransactionID = &val
		}
	}

	amount, err := parseDecimal(row, amountIdx)
	if err != nil {
		return txn, fmt.Errorf("row %d: invalid amount: %w", rowNum, err)
	}
	txn.Amount = amount

	fee, _ := parseDecimal(row, feeIdx)
	txn.Fee = math.Abs(fee)
	txn.NetAmount = math.Round((amount-math.Abs(fee))*100) / 100

	date, err := parseDate(row, dateIdx)
	if err != nil {
		return txn, fmt.Errorf("row %d: invalid date: %w", rowNum, err)
	}
	txn.TransactionDate = date

	txn.TransactionType = "payment"
	if amount < 0 {
		txn.TransactionType = "refund"
	}

	return txn, nil
}

func parseGenericRow(colMap map[string]int, row []string, rowNum int) (model.CreateTransactionRequest, error) {
	var txn model.CreateTransactionRequest

	txIDIdx := findColumn(colMap, "transaction_id", "id", "external_id")
	amountIdx := findColumn(colMap, "amount", "kwota", "value")
	feeIdx := findColumn(colMap, "fee", "commission", "prowizja")
	dateIdx := findColumn(colMap, "date", "transaction_date", "data")
	typeIdx := findColumn(colMap, "type", "transaction_type", "typ")

	if txIDIdx >= 0 && txIDIdx < len(row) {
		val := strings.TrimSpace(row[txIDIdx])
		if val != "" {
			txn.ExternalTransactionID = &val
		}
	}

	amount, err := parseDecimal(row, amountIdx)
	if err != nil {
		return txn, fmt.Errorf("row %d: invalid amount: %w", rowNum, err)
	}
	txn.Amount = amount

	fee, _ := parseDecimal(row, feeIdx)
	txn.Fee = math.Abs(fee)
	txn.NetAmount = math.Round((amount-math.Abs(fee))*100) / 100

	date, err := parseDate(row, dateIdx)
	if err != nil {
		return txn, fmt.Errorf("row %d: invalid date: %w", rowNum, err)
	}
	txn.TransactionDate = date

	txn.TransactionType = "payment"
	if typeIdx >= 0 && typeIdx < len(row) {
		val := strings.ToLower(strings.TrimSpace(row[typeIdx]))
		if val != "" {
			txn.TransactionType = val
		}
	}
	if amount < 0 {
		txn.TransactionType = "refund"
	}

	return txn, nil
}

// findColumn returns the index of the first matching column name, or -1.
func findColumn(colMap map[string]int, names ...string) int {
	for _, name := range names {
		if idx, ok := colMap[name]; ok {
			return idx
		}
	}
	return -1
}

// parseDecimal parses a decimal value from a CSV column, handling Polish format (comma as decimal separator).
func parseDecimal(row []string, idx int) (float64, error) {
	if idx < 0 || idx >= len(row) {
		return 0, fmt.Errorf("column index %d out of range", idx)
	}
	val := strings.TrimSpace(row[idx])
	if val == "" {
		return 0, nil
	}
	// Replace comma with dot for Polish format
	val = strings.ReplaceAll(val, ",", ".")
	// Remove spaces (thousand separators)
	val = strings.ReplaceAll(val, " ", "")
	return strconv.ParseFloat(val, 64)
}

// parseDate parses a date string trying multiple formats.
func parseDate(row []string, idx int) (string, error) {
	if idx < 0 || idx >= len(row) {
		return "", fmt.Errorf("column index %d out of range", idx)
	}
	val := strings.TrimSpace(row[idx])
	if val == "" {
		return "", fmt.Errorf("empty date")
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02.01.2006 15:04:05",
		"02.01.2006",
		"02-01-2006",
		"02/01/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, val); err == nil {
			return t.Format(time.RFC3339), nil
		}
	}

	return "", fmt.Errorf("unrecognized date format: %s", val)
}
