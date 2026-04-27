ALTER TABLE invoices
  ADD COLUMN sepa_return_at timestamptz,
  ADD COLUMN sepa_return_reason text,
  ADD COLUMN sepa_return_note text;
