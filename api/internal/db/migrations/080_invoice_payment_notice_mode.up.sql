ALTER TABLE eegs
  ADD COLUMN IF NOT EXISTS invoice_payment_notice_mode text NOT NULL DEFAULT 'sepa_lastschrift';
