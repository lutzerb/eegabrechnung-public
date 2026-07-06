ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS consumption_kwh NUMERIC(12,3) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS generation_kwh  NUMERIC(12,3) NOT NULL DEFAULT 0;

-- Backfill: treat existing total_kwh as consumption
UPDATE invoices SET consumption_kwh = total_kwh WHERE consumption_kwh = 0;
