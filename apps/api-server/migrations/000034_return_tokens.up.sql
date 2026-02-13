ALTER TABLE returns ADD COLUMN return_token TEXT;
ALTER TABLE returns ADD COLUMN customer_email TEXT;
ALTER TABLE returns ADD COLUMN customer_notes TEXT;
CREATE UNIQUE INDEX idx_returns_token ON returns(return_token) WHERE return_token IS NOT NULL;
