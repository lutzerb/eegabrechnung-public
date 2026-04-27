ALTER TABLE invoices
  DROP COLUMN IF EXISTS consumption_vat_pct,
  DROP COLUMN IF EXISTS consumption_vat_amount,
  DROP COLUMN IF EXISTS generation_vat_pct,
  DROP COLUMN IF EXISTS generation_vat_amount;
