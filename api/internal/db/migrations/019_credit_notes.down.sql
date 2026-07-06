ALTER TABLE invoices DROP COLUMN IF EXISTS document_type;
ALTER TABLE eegs
  DROP COLUMN IF EXISTS generate_credit_notes,
  DROP COLUMN IF EXISTS credit_note_number_prefix,
  DROP COLUMN IF EXISTS credit_note_number_digits;
