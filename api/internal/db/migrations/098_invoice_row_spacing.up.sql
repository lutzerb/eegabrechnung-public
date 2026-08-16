-- Configurable line/row spacing scale factor for the "individuell" invoice
-- design (see InvoiceTheme.RowSpacing / theme.h() in pdf_theme.go). Multiplies
-- every table row height and inter-section gap; default 1.0 keeps the current
-- (unscaled) layout unchanged. UI range: 0.7-1.3.
ALTER TABLE eegs ADD COLUMN invoice_row_spacing numeric(3,2) NOT NULL DEFAULT 1.0;
