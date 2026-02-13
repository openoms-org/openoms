DROP INDEX IF EXISTS idx_returns_token;
ALTER TABLE returns DROP COLUMN IF EXISTS customer_notes;
ALTER TABLE returns DROP COLUMN IF EXISTS customer_email;
ALTER TABLE returns DROP COLUMN IF EXISTS return_token;
