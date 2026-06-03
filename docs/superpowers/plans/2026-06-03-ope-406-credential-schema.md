# OPE-406 Credential & Settings Schema Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** A versioned credential/settings field-schema attached to a provider version — the single source of truth that (later) generates admin validation forms and customer setup forms. This task delivers the backend contract: model, storage, validation, and platform endpoints.

**Architecture:** A `provider_field_schemas` table (1:1 with `provider_versions`, no RLS), a typed schema model (field groups → fields with validation/secret/env-scope/help/capability/test-dependency), schema-structure validation, and `SetSchema`/`GetSchema` on `ProviderRegistryService` (reusing the version frozen-check), exposed as `PATCH`/`GET /v1/platform/providers/{id}/versions/{version_id}/schema`.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, golang-migrate, testify. Builds on OPE-405 (provider_versions, lifecycle, `IsPublishedState`).

---

## Scope

In scope: `provider_field_schemas` table + schema model + structural validation + Set/Get (frozen once published) + 2 endpoints + tests.

Out of scope: frontend dynamic form rendering (Studio UI, OPE-410), the customer-facing tenant-setup read path, capability profiles (OPE-407), validation probes (OPE-408). The schema only *declares* which fields are secret; tenant credential VALUES continue to live AES-encrypted in `integrations.credentials`.

## Field groups (design spec §118-137)

`secret_credentials`, `settings`, `environment`, `sync`, `feature_toggles`, `provider_options`.

## Per-field attributes (design spec §554)

key, label, type (`string|password|number|boolean|enum|url|textarea`), required, secret (storage target), environment_scope (`all|production|sandbox`), help_text, validation (enum/regex/min/max/min_length/max_length), capability_enabled (optional), test_connection_dependency.

## Data model (migration 000033) — no RLS, platform-managed

```sql
CREATE TABLE public.provider_field_schemas (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL UNIQUE REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    schema              jsonb NOT NULL DEFAULT '{"groups":[]}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
```
Plus dual-role grants DO-block (openoms_app/openoms). `.down.sql`: `DROP TABLE IF EXISTS public.provider_field_schemas;`

## Implementation Tasks (TDD)

### Task 1: Migration + model
- [ ] `migrations/000033_provider_field_schema.up.sql` / `.down.sql`.
- [ ] `internal/model/provider_schema.go`: `ProviderFieldSchema{ID, ProviderVersionID, Groups []ProviderFieldGroup, CreatedAt, UpdatedAt}`, `ProviderFieldGroup{Key, Label, Fields}`, `ProviderField{Key,Label,Type,Required,Secret,EnvironmentScope,HelpText,Validation,CapabilityEnabled,TestConnectionDependency}`, `ProviderFieldValidation{Enum,Regex,Min,Max,MinLength,MaxLength}`. Group-key + field-type + env-scope constants. `ValidateFieldSchema(groups) error` — valid group keys, valid field types, valid env scopes, unique field keys across the whole schema, non-empty enum when type=enum, compilable regex, min<=max. Returns a descriptive error.
- [ ] `model/provider_schema_test.go`: valid schema passes; duplicate field key, unknown group, unknown type, enum-without-values, bad regex, min>max each fail.

### Task 2: Repository
- [ ] `repository/provider_schema_repository.go`: `Upsert(ctx, versionID, schema []byte) (*model.ProviderFieldSchema, error)` (INSERT ... ON CONFLICT(provider_version_id) DO UPDATE SET schema, updated_at RETURNING ...), `GetByVersion(ctx, versionID) (*model.ProviderFieldSchema, error)` (pgx.ErrNoRows when absent). Store/scan `schema` as jsonb (json.RawMessage marshal/unmarshal of Groups).

### Task 3: Service (on ProviderRegistryService)
- [ ] Add `schemas *repository.ProviderSchemaRepository` to the service + constructor.
- [ ] `SetSchema(ctx, versionID, groups []model.ProviderFieldGroup, actorID *uuid.UUID) (*model.ProviderFieldSchema, error)`: GetVersion (404); if `IsPublishedState` → `ErrProviderVersionFrozen`; `ValidateFieldSchema` (→ `ErrInvalidFieldSchema`); marshal + Upsert. (Single write; no tx needed.)
- [ ] `GetSchema(ctx, versionID) (*model.ProviderFieldSchema, error)`: GetVersion (404); GetByVersion; if absent return an empty schema (Groups: []) rather than 404 (a version legitimately may have no schema yet).
- [ ] New error `ErrInvalidFieldSchema`.

### Task 4: Handler + router
- [ ] Add `UpdateSchema`/`GetSchema` to `ProviderHandler` + the `ProviderRegistry` interface. Request body = `{"groups":[...]}`. Map `ErrInvalidFieldSchema` → 422 (extend `writeServiceError`). Audit `platform.provider.version.schema_updated`.
- [ ] Router: under `/versions/{version_id}`: `r.With(rpp(read)).Get("/schema", ...)`, `r.With(rpp(write)).Patch("/schema", ...)`.
- [ ] Handler tests: invalid schema → 422, frozen → 422, get returns groups, success audits.

### Task 5: Integration test
- [ ] `tests/integration/provider_schema_test.go`: create def+version → SetSchema (valid, with a secret_credentials group + an enum field) → GetSchema returns it → transition to private_beta → SetSchema now → `ErrProviderVersionFrozen` → invalid schema (dup key) → `ErrInvalidFieldSchema`. Via appPool (grants + no-RLS).

### Task 6: Validate + docs
- [ ] `go test ./...`, vet, gofmt, golangci-lint clean; integration green; migrate down/up.
- [ ] `docs/system-documentation.md` — add provider_field_schemas + the schema endpoints to the Studio registry block.

## Risks
- Schema validation completeness — cover the structural rules with table-driven tests; regex compiled with `regexp.Compile`.
- Migration Safety CI — no indexes here (UNIQUE constraint only); no CONCURRENTLY concern.

## Self-Review
Covers OPE-420 Task 4 backend items: field groups; field validation (required/type/enum/regex/min-max/redaction(secret flag)/env-scope); schema is the generation source (render contract returned by GetSchema); secret values never in the schema (only field flags). Frontend form generation + rotation-of-stored-values are later tasks (UI / tenant-setup path).
