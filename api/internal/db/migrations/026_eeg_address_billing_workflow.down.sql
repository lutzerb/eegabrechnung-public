ALTER TABLE invoices DROP COLUMN IF EXISTS storno_pdf_path;
UPDATE billing_runs SET status = 'completed' WHERE status = 'finalized';
ALTER TABLE eegs DROP COLUMN IF EXISTS uid_nummer;
ALTER TABLE eegs DROP COLUMN IF EXISTS ort;
ALTER TABLE eegs DROP COLUMN IF EXISTS plz;
ALTER TABLE eegs DROP COLUMN IF EXISTS strasse;
