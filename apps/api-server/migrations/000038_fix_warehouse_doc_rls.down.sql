-- Revert to the original (vulnerable) COALESCE-based RLS policies
DROP POLICY IF EXISTS tenant_isolation ON warehouse_documents;
CREATE POLICY tenant_isolation ON warehouse_documents
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::uuid, tenant_id));

DROP POLICY IF EXISTS tenant_isolation ON warehouse_document_items;
CREATE POLICY tenant_isolation ON warehouse_document_items
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::uuid, tenant_id));
