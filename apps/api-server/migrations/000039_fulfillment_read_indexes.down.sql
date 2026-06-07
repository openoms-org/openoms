-- OPE-419 rollback: drop the operations / fulfillment read API status index.
DROP INDEX IF EXISTS public.idx_fulfillment_processes_status;
