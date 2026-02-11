-- Migration 000038: Fix RLS policies on warehouse_documents and warehouse_document_items
--
-- The original policies used a COALESCE pattern that falls back to
-- tenant_id = tenant_id (always true) when app.current_tenant_id is not set.
-- This is a security vulnerability — replace with the strict check used by
-- all other tables, which denies access when the setting is absent.

DROP POLICY IF EXISTS tenant_isolation ON warehouse_documents;
CREATE POLICY tenant_isolation ON warehouse_documents
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation ON warehouse_document_items;
CREATE POLICY tenant_isolation ON warehouse_document_items
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
