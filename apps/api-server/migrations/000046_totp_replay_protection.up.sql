-- OPE-536 TOTP replay protection. Records the highest TOTP time-step (Unix/period
-- counter) that was successfully accepted for a user during 2FA login. A code whose
-- matching time-step is <= the stored value is rejected as a replay, which closes the
-- "reuse an observed code within the validity window" gap. Additive, nullable column:
-- NULL means "no successful 2FA login recorded yet" (treated as no prior step). Existing
-- table-level grants cover the new column, so no GRANT is required.
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS totp_last_used_step bigint;
