//go:build integration

// Package integration holds DB-bound integration tests that exercise the real
// PostgreSQL schema, Row-Level Security policies and tenant isolation against a
// live database. They are gated behind the `integration` build tag and only run
// when DATABASE_URL points at a disposable Postgres (the CI postgres service, or
// a local `docker compose -f docker-compose.dev.yml up -d postgres`). See OPE-507.
//
// RLS model: production/staging connect as a NON-superuser, NOBYPASSRLS app role
// (openoms_app / openoms). The migrations FORCE ROW LEVEL SECURITY, so even the
// table owner is subject to RLS — but a SUPERUSER always bypasses it. The CI/dev
// Postgres superuser therefore cannot exercise RLS. This harness reproduces the
// production setup: it creates a non-superuser `openoms_app` role BEFORE running
// migrations (so the migrations' dynamic grants — incl. 000028 — apply to it),
// then connects the "app" pool as that role so RLS genuinely applies. The
// "super" pool stays superuser for cross-tenant seeding/inspection.
//
// Both pools are opened via the app's own database.Connect and use
// default_query_exec_mode=simple_protocol — the same mode production uses behind
// the Supabase transaction pooler. This matters: with the default extended
// protocol + statement caching, set_config('app.current_tenant_id', ...) does not
// reliably reach the RLS policy's current_setting() at execution, which is exactly
// why production runs in simple_protocol. Tests therefore mirror prod precisely.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const appRolePassword = "integration_test_app"

var (
	superPool *pgxpool.Pool // superuser: runs migrations, seeds across tenants (bypasses RLS)
	appPool   *pgxpool.Pool // openoms_app: NOSUPERUSER NOBYPASSRLS — RLS applies
)

func TestMain(m *testing.M) {
	rawDSN := os.Getenv("DATABASE_URL")
	if rawDSN == "" {
		fmt.Println("DATABASE_URL not set — skipping DB integration tests (run with -tags integration against a disposable Postgres)")
		os.Exit(0)
	}
	if err := setup(rawDSN); err != nil {
		fmt.Fprintf(os.Stderr, "integration harness setup failed: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if superPool != nil {
		superPool.Close()
	}
	if appPool != nil {
		appPool.Close()
	}
	os.Exit(code)
}

func setup(rawDSN string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Superuser pool (simple_protocol, mirroring prod) for migrations + cross-tenant seeding.
	var err error
	superPool, err = connect(ctx, rawDSN, "")
	if err != nil {
		return fmt.Errorf("connect superuser pool: %w", err)
	}

	// 1. Ensure a non-superuser, NOBYPASSRLS app role exists BEFORE migrations so the
	//    migrations' dynamic role grants (e.g. 000028) apply to it, mirroring prod.
	if _, err := superPool.Exec(ctx, fmt.Sprintf(`
		DO $$
		BEGIN
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
		    CREATE ROLE openoms_app LOGIN PASSWORD '%[1]s' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		  ELSE
		    ALTER ROLE openoms_app WITH LOGIN PASSWORD '%[1]s' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		  END IF;
		END $$;`, appRolePassword)); err != nil {
		return fmt.Errorf("ensure openoms_app role: %w", err)
	}

	// 2. Apply all migrations as the superuser via the golang-migrate CLI (the same
	//    tool the Taskfile/deploy use), against the raw DSN. "no change" is fine.
	if err := runMigrations(rawDSN); err != nil {
		return err
	}

	// 3. Re-assert grants to openoms_app idempotently (fresh DBs already get this from
	//    000028; re-runs against an existing DB do not re-run migrations). Mirrors the
	//    deploy's post-migrate GRANT step.
	if _, err := superPool.Exec(ctx, `
		GRANT USAGE ON SCHEMA public TO openoms_app;
		GRANT ALL ON ALL TABLES IN SCHEMA public TO openoms_app;
		GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO openoms_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO openoms_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO openoms_app;`); err != nil {
		return fmt.Errorf("grant to openoms_app: %w", err)
	}

	// 4. App pool: same target DB, authenticated as openoms_app so RLS applies.
	appPool, err = connect(ctx, rawDSN, "openoms_app")
	if err != nil {
		return fmt.Errorf("connect app pool (openoms_app): %w", err)
	}
	return nil
}

// connect opens a pgxpool mirroring the production connection: simple_protocol
// (set_config + RLS does not work reliably under the extended protocol's statement
// caching — production runs simple_protocol behind the Supabase pooler) and the
// json.RawMessage->jsonb type registration that simple_protocol requires. If asUser
// is non-empty the userinfo is replaced with that role + the harness app password.
func connect(ctx context.Context, rawDSN, asUser string) (*pgxpool.Pool, error) {
	u, err := url.Parse(rawDSN)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if asUser != "" {
		u.User = url.UserPassword(asUser, appRolePassword)
	}
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		tm := conn.TypeMap()
		tm.RegisterType(&pgtype.Type{
			Name:  "jsonb",
			OID:   pgtype.JSONBOID,
			Codec: &pgtype.JSONBCodec{Marshal: json.Marshal, Unmarshal: json.Unmarshal},
		})
		tm.RegisterDefaultPgType(json.RawMessage{}, "jsonb")
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func runMigrations(rawDSN string) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("cannot resolve migrations path")
	}
	migrationsDir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	out, err := exec.Command("golang-migrate", "-path", migrationsDir, "-database", rawDSN, "up").CombinedOutput() //nolint:gosec // fixed args, test-only
	if err != nil {
		if strings.Contains(string(out), "no change") {
			return nil
		}
		return fmt.Errorf("golang-migrate up failed: %v\n%s", err, out)
	}
	return nil
}

// seedTenant inserts a tenant via the superuser pool (bypassing RLS) and returns its id.
// The tenant is removed on test cleanup.
func seedTenant(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	id := uuid.New()
	slug := "it-" + id.String()[:8]
	_, err := superPool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, plan) VALUES ($1, $2, $3, 'free')`,
		id, "integration "+slug, slug)
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM tenants WHERE id = $1", id)
	})
	return id
}
