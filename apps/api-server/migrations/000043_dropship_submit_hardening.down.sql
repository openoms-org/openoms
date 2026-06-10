-- OPE-517/OPE-518 rollback (additive marker column + backstop index).
DROP INDEX IF EXISTS public.uq_dropship_orders_submitted_once;
ALTER TABLE public.dropship_orders DROP COLUMN IF EXISTS submit_attempted_at;
