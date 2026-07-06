ALTER TABLE invoices
  DROP COLUMN IF EXISTS consumption_kwh,
  DROP COLUMN IF EXISTS generation_kwh;
