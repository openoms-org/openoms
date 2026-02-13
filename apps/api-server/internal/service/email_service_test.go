package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestRenderEmailTemplate_BasicStatus(t *testing.T) {
	order := &model.Order{
		ID:           uuid.New(),
		CustomerName: "Jan Kowalski",
		TotalAmount:  99.99,
		Currency:     "PLN",
	}

	subject, body := renderEmailTemplate(order, "confirmed", "TestShop", nil)

	assert.Contains(t, subject, order.ID.String()[:8])
	assert.Contains(t, subject, "confirmed")
	assert.Contains(t, body, "Jan Kowalski")
	assert.Contains(t, body, "99.99 PLN")
	assert.Contains(t, body, "CONFIRMED")
}

func TestRenderEmailTemplate_ShippedStatus(t *testing.T) {
	order := &model.Order{
		ID:           uuid.New(),
		CustomerName: "Anna Nowak",
		TotalAmount:  50.00,
		Currency:     "PLN",
	}

	_, body := renderEmailTemplate(order, "shipped", "TestShop", nil)

	assert.Contains(t, body, "w drodze")
}

func TestRenderEmailTemplate_CancelledStatus(t *testing.T) {
	order := &model.Order{
		ID:           uuid.New(),
		CustomerName: "User",
		TotalAmount:  10.00,
		Currency:     "PLN",
	}

	_, body := renderEmailTemplate(order, "cancelled", "Shop", nil)

	assert.Contains(t, body, "anulowania")
}

func TestRenderEmailTemplate_RefundedStatus(t *testing.T) {
	order := &model.Order{
		ID:           uuid.New(),
		CustomerName: "User",
		TotalAmount:  25.00,
		Currency:     "PLN",
	}

	_, body := renderEmailTemplate(order, "refunded", "Shop", nil)

	assert.Contains(t, body, "Zwrot")
}

func TestRenderEmailTemplate_WithStatusConfig(t *testing.T) {
	order := &model.Order{
		ID:           uuid.New(),
		CustomerName: "User",
		TotalAmount:  10.00,
		Currency:     "PLN",
	}

	cfg := &model.OrderStatusConfig{
		Statuses: []model.StatusDef{
			{Key: "new", Label: "Now", Color: "blue", Position: 1},
			{Key: "confirmed", Label: "Potwierdzone", Color: "green", Position: 2},
		},
	}

	subject, body := renderEmailTemplate(order, "confirmed", "Shop", cfg)

	assert.Contains(t, subject, "Potwierdzone")
	assert.Contains(t, body, "POTWIERDZONE")
}

func TestRenderEmailTemplate_CompanyName(t *testing.T) {
	order := &model.Order{
		ID:           uuid.New(),
		CustomerName: "User",
		TotalAmount:  10.00,
		Currency:     "PLN",
	}

	_, body := renderEmailTemplate(order, "new", "MojSklep", nil)

	assert.Contains(t, body, "MojSklep")
}

func TestSendMail_SMTPNotConfigured(t *testing.T) {
	err := sendMail(model.EmailSettings{}, "to@test.com", "Subject", "<p>Body</p>")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP not configured")
}

func TestSendMail_MissingFromEmail(t *testing.T) {
	err := sendMail(model.EmailSettings{
		SMTPHost: "localhost",
	}, "to@test.com", "Subject", "<p>Body</p>")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP not configured")
}
