-- Per-supplier configurable sync interval (default 60 minutes)
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS sync_interval_minutes INTEGER NOT NULL DEFAULT 60;

-- Grant (already has full grants, but be explicit for new column)
COMMENT ON COLUMN suppliers.sync_interval_minutes IS 'How often to sync this supplier feed, in minutes. Default 60.';
