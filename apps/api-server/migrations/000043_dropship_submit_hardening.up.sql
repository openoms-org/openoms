-- OPE-517/OPE-518 supplier-order submit hardening (additive).
--
-- submit_attempted_at is the two-phase submit-intent marker: it is set and COMMITTED
-- before provider.CreateOrder is called, and kept afterwards as historical evidence.
-- A retry that finds the marker with supplier_reference still NULL must NOT resubmit
-- automatically — it opens a supplier_manual_submission_required blocker so an operator
-- verifies the order at the supplier first (prevents a duplicate purchase order when the
-- recording transaction failed after a successful CreateOrder).
ALTER TABLE public.dropship_orders ADD COLUMN IF NOT EXISTS submit_attempted_at timestamptz;

-- v1 backstop — at most one SUBMITTED dropship order (supplier_reference set) per
-- (order, supplier); the planned multi-document-split feature must relax this index.
CREATE UNIQUE INDEX uq_dropship_orders_submitted_once ON public.dropship_orders (tenant_id, order_id, supplier_id) WHERE supplier_reference IS NOT NULL; -- migrate:index-lock-ok
