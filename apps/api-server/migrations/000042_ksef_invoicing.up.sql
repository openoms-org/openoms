-- KSeF (Krajowy System e-Faktur) integration columns on invoices table.
-- KSeF configuration is stored in the tenant settings JSONB under the "ksef" key.

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS ksef_number TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS ksef_status TEXT NOT NULL DEFAULT 'not_sent';
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS ksef_sent_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS ksef_response JSONB;

-- Index for background worker polling pending KSeF statuses.
CREATE INDEX IF NOT EXISTS idx_invoices_ksef_status ON invoices(ksef_status) WHERE ksef_status != 'not_sent';
