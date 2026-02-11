CREATE TABLE warehouse_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_number TEXT NOT NULL,
    document_type TEXT NOT NULL, -- 'PZ', 'WZ', 'MM'
    status TEXT NOT NULL DEFAULT 'draft', -- draft, confirmed, cancelled
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    target_warehouse_id UUID REFERENCES warehouses(id), -- for MM transfers
    supplier_id UUID REFERENCES suppliers(id), -- for PZ
    order_id UUID REFERENCES orders(id), -- for WZ linked to order
    notes TEXT,
    confirmed_at TIMESTAMPTZ,
    confirmed_by UUID REFERENCES users(id),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE warehouse_document_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES warehouse_documents(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    variant_id UUID REFERENCES product_variants(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(12,2),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RLS
ALTER TABLE warehouse_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE warehouse_documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON warehouse_documents
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::uuid, tenant_id));

ALTER TABLE warehouse_document_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE warehouse_document_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON warehouse_document_items
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::uuid, tenant_id));

-- Unique document number per tenant
CREATE UNIQUE INDEX idx_warehouse_docs_number ON warehouse_documents(tenant_id, document_number);

-- Indexes
CREATE INDEX idx_warehouse_docs_tenant ON warehouse_documents(tenant_id);
CREATE INDEX idx_warehouse_docs_type ON warehouse_documents(document_type);
CREATE INDEX idx_warehouse_docs_status ON warehouse_documents(status);
CREATE INDEX idx_warehouse_docs_warehouse ON warehouse_documents(warehouse_id);
CREATE INDEX idx_warehouse_doc_items_document ON warehouse_document_items(document_id);
CREATE INDEX idx_warehouse_doc_items_product ON warehouse_document_items(product_id);

-- Triggers
CREATE TRIGGER update_warehouse_documents_updated_at BEFORE UPDATE ON warehouse_documents FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON warehouse_documents TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON warehouse_document_items TO openoms_app;
