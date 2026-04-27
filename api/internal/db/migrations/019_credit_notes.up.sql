-- Credit note support for VAT-liable producers.
ALTER TABLE eegs
  ADD COLUMN IF NOT EXISTS generate_credit_notes bool NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS credit_note_number_prefix text NOT NULL DEFAULT 'GS',
  ADD COLUMN IF NOT EXISTS credit_note_number_digits int NOT NULL DEFAULT 5;

ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS document_type text NOT NULL DEFAULT 'invoice';
