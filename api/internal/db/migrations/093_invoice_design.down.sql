ALTER TABLE eegs DROP CONSTRAINT eegs_invoice_font_size_check;
ALTER TABLE eegs DROP COLUMN invoice_font_size;

ALTER TABLE eegs DROP CONSTRAINT eegs_invoice_font_family_check;
ALTER TABLE eegs DROP COLUMN invoice_font_family;

ALTER TABLE eegs DROP COLUMN invoice_logo_left;

ALTER TABLE eegs DROP CONSTRAINT eegs_invoice_accent_color_check;
ALTER TABLE eegs DROP COLUMN invoice_accent_color;

ALTER TABLE eegs DROP CONSTRAINT eegs_invoice_design_check;
ALTER TABLE eegs DROP COLUMN invoice_design;
