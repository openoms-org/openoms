-- OPE-526 rollback: drop the scope/discriminator CHECK.
ALTER TABLE public.supplier_availability_policy
    DROP CONSTRAINT IF EXISTS chk_sap_scope_discriminator;
