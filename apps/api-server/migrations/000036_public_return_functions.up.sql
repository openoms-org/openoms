-- Migration 000036: SECURITY DEFINER functions for public return self-service.
-- These bypass RLS so the unauthenticated public return endpoints can look up
-- orders and returns without needing a tenant context set first.

CREATE OR REPLACE FUNCTION public.find_order_tenant_id(p_order_id UUID)
RETURNS TABLE(tenant_id UUID, customer_email TEXT)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT o.tenant_id, o.customer_email
    FROM orders o
    WHERE o.id = p_order_id;
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION public.find_return_by_token(p_token TEXT)
RETURNS TABLE(
    id UUID, tenant_id UUID, order_id UUID, status VARCHAR,
    reason TEXT, items JSONB, refund_amount DECIMAL,
    notes TEXT, return_token TEXT, customer_email TEXT,
    customer_notes TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT r.id, r.tenant_id, r.order_id, r.status,
           r.reason, r.items, r.refund_amount,
           r.notes, r.return_token, r.customer_email,
           r.customer_notes, r.created_at, r.updated_at
    FROM returns r
    WHERE r.return_token = p_token;
$$ LANGUAGE sql STABLE;

GRANT EXECUTE ON FUNCTION public.find_order_tenant_id(UUID) TO openoms_app;
GRANT EXECUTE ON FUNCTION public.find_return_by_token(TEXT) TO openoms_app;
