package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrCustomerNotFound is returned when a customer does not exist.
	ErrCustomerNotFound = errors.New("customer not found")
)

// CustomerService handles business logic for customer management.
type CustomerService struct {
	customerRepo    repository.CustomerRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
	logger          *slog.Logger
}

// NewCustomerService creates a new CustomerService.
func NewCustomerService(
	customerRepo repository.CustomerRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *CustomerService {
	return &CustomerService{
		customerRepo:    customerRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
		logger:          logger,
	}
}

// List returns a paginated list of customers for a tenant.
func (s *CustomerService) List(ctx context.Context, tenantID uuid.UUID, filter model.CustomerListFilter) (model.ListResponse[model.Customer], error) {
	var resp model.ListResponse[model.Customer]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		customers, total, err := s.customerRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(customers, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single customer by ID.
func (s *CustomerService) Get(ctx context.Context, tenantID, customerID uuid.UUID) (*model.Customer, error) {
	var customer *model.Customer
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		customer, err = s.customerRepo.FindByID(ctx, tx, customerID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

// Create inserts a new customer.
func (s *CustomerService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateCustomerRequest, actorID uuid.UUID, ip string) (*model.Customer, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	req.Name = model.StripHTMLTags(req.Name)

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	customer := &model.Customer{
		ID:                     uuid.New(),
		TenantID:               tenantID,
		Email:                  req.Email,
		Phone:                  req.Phone,
		Name:                   req.Name,
		CompanyName:            req.CompanyName,
		NIP:                    req.NIP,
		DefaultShippingAddress: req.DefaultShippingAddress,
		DefaultBillingAddress:  req.DefaultBillingAddress,
		Tags:                   tags,
		Notes:                  req.Notes,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.customerRepo.Create(ctx, tx, customer); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "customer.created",
			EntityType: "customer",
			EntityID:   customer.ID,
			Changes:    map[string]string{"name": req.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "customer.created", customer) })
	}
	return customer, nil
}

// Update modifies an existing customer.
func (s *CustomerService) Update(ctx context.Context, tenantID, customerID uuid.UUID, req model.UpdateCustomerRequest, actorID uuid.UUID, ip string) (*model.Customer, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var customer *model.Customer
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.customerRepo.FindByID(ctx, tx, customerID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrCustomerNotFound
		}

		if err := s.customerRepo.Update(ctx, tx, customerID, req); err != nil {
			return err
		}

		customer, err = s.customerRepo.FindByID(ctx, tx, customerID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "customer.updated",
			EntityType: "customer",
			EntityID:   customerID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	if customer != nil && s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "customer.updated", customer) })
	}
	return customer, err
}

// Delete removes a customer by ID.
func (s *CustomerService) Delete(ctx context.Context, tenantID, customerID uuid.UUID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		customer, err := s.customerRepo.FindByID(ctx, tx, customerID)
		if err != nil {
			return err
		}
		if customer == nil {
			return ErrCustomerNotFound
		}

		if err := s.customerRepo.Delete(ctx, tx, customerID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "customer.deleted",
			EntityType: "customer",
			EntityID:   customerID,
			Changes:    map[string]string{"name": customer.Name},
			IPAddress:  ip,
		})
	})
	if err == nil && s.webhookDispatch != nil {
		asyncutil.SafeGo(func() {
			s.webhookDispatch.Dispatch(context.Background(), tenantID, "customer.deleted", map[string]any{"customer_id": customerID.String()})
		})
	}
	return err
}

// ListOrders returns a paginated list of orders for a specific customer.
func (s *CustomerService) ListOrders(ctx context.Context, tenantID, customerID uuid.UUID, filter model.OrderListFilter) (model.ListResponse[model.Order], error) {
	var resp model.ListResponse[model.Order]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		customer, err := s.customerRepo.FindByID(ctx, tx, customerID)
		if err != nil {
			return err
		}
		if customer == nil {
			return ErrCustomerNotFound
		}

		orders, total, err := s.customerRepo.ListOrdersByCustomerID(ctx, tx, customerID, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(orders, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// IncrementOrderStats increments the customer's order count and total spent.
func (s *CustomerService) IncrementOrderStats(ctx context.Context, tenantID, customerID uuid.UUID, amount float64) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.customerRepo.IncrementOrderStats(ctx, tx, customerID, amount)
	})
}
