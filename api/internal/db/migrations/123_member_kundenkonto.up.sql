-- Kundenkonto: freely editable per-member customer account number (e.g. for a
-- customer's own external bookkeeping), plus a per-EEG starting value used only
-- as a suggestion/floor. No cross-EEG or org-wide sequence — each EEG is numbered
-- independently, mirroring invoice_number_start.
ALTER TABLE members ADD COLUMN IF NOT EXISTS kundenkonto text NOT NULL DEFAULT '';
ALTER TABLE eegs    ADD COLUMN IF NOT EXISTS kundenkonto_start int NOT NULL DEFAULT 0;

-- Unique per EEG only (not org-wide); empty string ("not yet assigned") is excluded
-- so any number of members can remain without a Kundenkonto.
CREATE UNIQUE INDEX IF NOT EXISTS members_eeg_kundenkonto_uidx
  ON members (eeg_id, kundenkonto) WHERE kundenkonto <> '';
