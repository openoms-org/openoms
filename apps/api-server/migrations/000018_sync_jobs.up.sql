-- Migration 000018: Sync jobs table (append-only, no updated_at)
CREATE TABLE sync_jobs (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id  UUID         NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    job_type        TEXT         NOT NULL,
    status          TEXT         NOT NULL DEFAULT 'pending',
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    items_processed INTEGER      NOT NULL DEFAULT 0,
    items_failed    INTEGER      NOT NULL DEFAULT 0,
    error_message   TEXT,
    metadata        JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_jobs_tenant ON sync_jobs(tenant_id, created_at DESC);
CREATE INDEX idx_sync_jobs_integration ON sync_jobs(tenant_id, integration_id, created_at DESC);
CREATE INDEX idx_sync_jobs_status ON sync_jobs(tenant_id, status)
    WHERE status IN ('pending', 'running');

ALTER TABLE sync_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sync_jobs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON sync_jobs TO openoms_app;
