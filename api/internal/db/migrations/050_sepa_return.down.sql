ALTER TABLE invoices
  DROP COLUMN IF EXISTS sepa_return_at,
  DROP COLUMN IF EXISTS sepa_return_reason,
  DROP COLUMN IF EXISTS sepa_return_note;
