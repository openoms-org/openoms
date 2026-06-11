-- OPE-536 rollback: drop the TOTP replay-protection column.
ALTER TABLE public.users DROP COLUMN IF EXISTS totp_last_used_step;
