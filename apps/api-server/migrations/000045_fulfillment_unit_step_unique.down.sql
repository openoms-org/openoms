-- OPE-523 rollback: drop the fulfillment unit/step dedupe unique indexes.
--
-- NOTE: CreateUnit/CreateStep use INSERT ... ON CONFLICT ... DO NOTHING, which
-- REQUIRES these unique indexes for arbiter inference. Apply this down migration
-- only together with reverting the application to a build that does not use those
-- ON CONFLICT clauses (i.e. a full rollback of both schema and image) — otherwise
-- unit/step creation will error with "no unique or exclusion constraint matching
-- the ON CONFLICT". A normal `helm rollback` (image reverted, this down NOT
-- auto-run) is unaffected: the old plain-INSERT code works fine with the unique
-- indexes still present.
DROP INDEX IF EXISTS public.uq_fulfillment_units_dedupe;
DROP INDEX IF EXISTS public.uq_fulfillment_steps_unit_step;
