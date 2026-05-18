package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

var errInvalidProviderWebhookIntegrationID = errors.New("invalid provider webhook integration id")

type providerWebhookSecretResolver interface {
	Resolve(ctx context.Context, provider string, integrationID uuid.UUID) (string, *service.ProviderWebhookScope, error)
}

func resolveProviderWebhookSecret(
	ctx context.Context,
	r *http.Request,
	provider string,
	legacySecret string,
	resolver providerWebhookSecretResolver,
) (string, *service.ProviderWebhookScope, error) {
	integrationIDParam := chi.URLParam(r, "integrationID")
	if integrationIDParam == "" {
		return legacySecret, nil, nil
	}

	integrationID, err := uuid.Parse(integrationIDParam)
	if err != nil {
		return "", nil, errInvalidProviderWebhookIntegrationID
	}
	if resolver == nil {
		return "", nil, service.ErrWebhookSecretNotConfigured
	}

	return resolver.Resolve(ctx, provider, integrationID)
}
