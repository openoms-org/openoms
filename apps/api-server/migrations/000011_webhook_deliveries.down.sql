-- Migration 000011 down: Remove webhook deliveries
DROP TABLE IF EXISTS webhook_deliveries;
