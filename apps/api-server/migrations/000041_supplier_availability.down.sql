-- OPE-418 rollback: drop the supplier-availability read-model tables (additive).
DROP TABLE IF EXISTS public.supplier_availability_policy;
DROP TABLE IF EXISTS public.supplier_availability;
