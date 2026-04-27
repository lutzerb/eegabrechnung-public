-- Invoice VAT breakdown — stored at billing time so it remains correct even if EEG settings change later
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS net_amount     NUMERIC(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS vat_amount     NUMERIC(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS vat_pct_applied NUMERIC(5,2) NOT NULL DEFAULT 0;

-- Backfill existing invoices using current EEG VAT settings (best approximation)
UPDATE invoices i
SET
  vat_pct_applied = CASE WHEN e.use_vat THEN e.vat_pct ELSE 0 END,
  vat_amount = CASE
    WHEN e.use_vat AND e.vat_pct > 0
      THEN ROUND(i.total_amount - (i.total_amount / (1 + e.vat_pct / 100)), 4)
    ELSE 0
  END,
  net_amount = CASE
    WHEN e.use_vat AND e.vat_pct > 0
      THEN ROUND(i.total_amount / (1 + e.vat_pct / 100), 4)
    ELSE i.total_amount
  END
FROM eegs e
WHERE i.eeg_id = e.id;

-- DATEV / accounting configuration per EEG
ALTER TABLE eegs
  ADD COLUMN IF NOT EXISTS accounting_revenue_account  INT  NOT NULL DEFAULT 4000,
  ADD COLUMN IF NOT EXISTS accounting_expense_account  INT  NOT NULL DEFAULT 5000,
  ADD COLUMN IF NOT EXISTS accounting_debitor_prefix   INT  NOT NULL DEFAULT 10000,
  ADD COLUMN IF NOT EXISTS datev_consultant_nr         TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS datev_client_nr             TEXT NOT NULL DEFAULT '';
