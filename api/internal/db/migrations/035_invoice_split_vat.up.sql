-- Split VAT tracking: separate columns for consumption (EEG-level) and
-- generation (member-level) VAT, so DATEV and PDF can show independent treatment.
ALTER TABLE invoices
  ADD COLUMN consumption_vat_pct    NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN consumption_vat_amount NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN generation_vat_pct     NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN generation_vat_amount  NUMERIC NOT NULL DEFAULT 0;
