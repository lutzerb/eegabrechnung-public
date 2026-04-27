CREATE UNIQUE INDEX IF NOT EXISTS invoices_eeg_document_number_uidx
  ON invoices (eeg_id, document_type, invoice_number)
  WHERE invoice_number IS NOT NULL;
