package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	allegroint "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
)

func attachAllegroTokenPersist(provider any, ctx context.Context, pool *pgxpool.Pool, encryptionKey []byte, integrationID uuid.UUID, existing []byte) {
	if pool == nil || len(encryptionKey) == 0 || len(existing) == 0 {
		return
	}
	allegroint.AttachTokenRefreshPersist(provider, allegroint.PersistFn(existing, func(newJSON []byte) error {
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		encrypted, err := crypto.Encrypt(newJSON, encryptionKey)
		if err != nil {
			return err
		}
		credsJSONB, err := json.Marshal(encrypted)
		if err != nil {
			return err
		}
		_, err = pool.Exec(persistCtx,
			`UPDATE integrations SET credentials = $1::jsonb, updated_at = NOW() WHERE id = $2`,
			credsJSONB, integrationID,
		)
		return err
	}))
}
