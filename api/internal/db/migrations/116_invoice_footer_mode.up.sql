-- Controls how the "individuell" invoice design (pdf_theme.go) renders
-- invoice_footer_text: 'inline' (default, unchanged behavior — drawn once after
-- the content, on whichever page that lands on) or 'page' (registered as a real
-- fpdf page footer, repeated at a fixed position at the bottom of every page).
-- Not used by the standard design, which always renders the footer inline.
ALTER TABLE eegs ADD COLUMN invoice_footer_mode text NOT NULL DEFAULT 'inline';
