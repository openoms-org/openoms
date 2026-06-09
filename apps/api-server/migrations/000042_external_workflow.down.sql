-- OPE-421/Phase-13 rollback (additive tables/columns).
ALTER TABLE public.orchestration_attempts DROP COLUMN IF EXISTS external_execution_id;
DROP TABLE IF EXISTS public.external_workflow_tokens;
