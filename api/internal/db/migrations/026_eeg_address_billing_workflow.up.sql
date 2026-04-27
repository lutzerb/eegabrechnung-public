-- Add address and UID fields to EEGs (Rechnungssteller block, §11 UStG)
ALTER TABLE eegs
    ADD COLUMN IF NOT EXISTS strasse    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS plz        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ort        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS uid_nummer TEXT NOT NULL DEFAULT '';

-- Rename billing_run status 'completed' → 'finalized'
-- Draft runs can be deleted; finalized runs require formal storno
UPDATE billing_runs SET status = 'finalized' WHERE status = 'completed';

-- Add storno PDF path to invoices (set when a finalized run is cancelled)
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS storno_pdf_path TEXT NOT NULL DEFAULT '';
