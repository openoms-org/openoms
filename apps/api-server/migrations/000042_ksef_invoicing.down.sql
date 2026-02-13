DROP INDEX IF EXISTS idx_invoices_ksef_status;

ALTER TABLE invoices DROP COLUMN IF EXISTS ksef_response;
ALTER TABLE invoices DROP COLUMN IF EXISTS ksef_sent_at;
ALTER TABLE invoices DROP COLUMN IF EXISTS ksef_status;
ALTER TABLE invoices DROP COLUMN IF EXISTS ksef_number;
