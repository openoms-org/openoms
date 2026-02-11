package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTenantIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, uuid.Nil, TenantIDFromContext(ctx))
}
