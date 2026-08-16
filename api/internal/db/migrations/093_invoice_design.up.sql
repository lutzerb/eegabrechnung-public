-- Per-EEG invoice visual design: "standard" (existing fpdf layout, unchanged) or
-- "individuell" (alternate Oikos-style layout, see internal/invoice/pdf_theme.go).
-- Accent color / logo position / font family / font size are only used when
-- invoice_design = 'individuell'; they carry sensible defaults so existing rows
-- don't need backfilling.

ALTER TABLE eegs ADD COLUMN invoice_design text NOT NULL DEFAULT 'standard';
ALTER TABLE eegs ADD CONSTRAINT eegs_invoice_design_check
  CHECK (invoice_design IN ('standard', 'individuell'));

ALTER TABLE eegs ADD COLUMN invoice_accent_color text NOT NULL DEFAULT '#c9b89a';
ALTER TABLE eegs ADD CONSTRAINT eegs_invoice_accent_color_check
  CHECK (invoice_accent_color ~ '^#[0-9a-fA-F]{6}$');

ALTER TABLE eegs ADD COLUMN invoice_logo_left boolean NOT NULL DEFAULT true;

ALTER TABLE eegs ADD COLUMN invoice_font_family text NOT NULL DEFAULT 'dejavu';
ALTER TABLE eegs ADD CONSTRAINT eegs_invoice_font_family_check
  CHECK (invoice_font_family IN ('dejavu', 'roboto', 'opensans', 'ptserif'));

-- Fixed tables have fixed mm column widths (see the header-overflow issue found
-- while prototyping) — keep the selectable range narrow enough that cells don't
-- overflow rather than letting the UI offer arbitrary point sizes.
ALTER TABLE eegs ADD COLUMN invoice_font_size int NOT NULL DEFAULT 10;
ALTER TABLE eegs ADD CONSTRAINT eegs_invoice_font_size_check
  CHECK (invoice_font_size BETWEEN 8 AND 12);
