-- OPE-526: scope/discriminator integrity for supplier_availability_policy. The four
-- discriminator columns (supplier_id, product_id, listing_id, channel) are nullable and
-- the per-scope unique indexes from 000041 are PARTIAL, so a row like scope='supplier'
-- with supplier_id NULL escapes its unique index entirely (NULL keys never conflict).
-- This CHECK makes "row not covered by the scope's unique index" impossible:
--   supplier -> supplier_id set, others null
--   product  -> supplier_id AND product_id set (uq_sap_product spans both), others null
--   listing  -> listing_id set, others null
--   channel  -> channel set, others null
-- num_nonnulls() is used instead of IS [NOT] NULL chains so the predicate stays compact
-- and column-count exact (n set + rest 0). The table is new (000041, gated feature) and
-- only written through the audited SetPolicy path, so no existing rows can violate it.
ALTER TABLE public.supplier_availability_policy
    ADD CONSTRAINT chk_sap_scope_discriminator CHECK (
        (scope = 'supplier' AND num_nonnulls(supplier_id) = 1 AND num_nonnulls(product_id, listing_id, channel) = 0)
        OR (scope = 'product' AND num_nonnulls(supplier_id, product_id) = 2 AND num_nonnulls(listing_id, channel) = 0)
        OR (scope = 'listing' AND num_nonnulls(listing_id) = 1 AND num_nonnulls(supplier_id, product_id, channel) = 0)
        OR (scope = 'channel' AND num_nonnulls(channel) = 1 AND num_nonnulls(supplier_id, product_id, listing_id) = 0)
    );
