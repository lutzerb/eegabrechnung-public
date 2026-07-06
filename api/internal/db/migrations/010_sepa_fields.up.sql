-- Add SEPA banking fields to EEGs table
ALTER TABLE eegs
  ADD COLUMN IF NOT EXISTS iban             TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS bic              TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS sepa_creditor_id TEXT NOT NULL DEFAULT '';
