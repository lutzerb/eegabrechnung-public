-- Scale factor for the "individuell" invoice design's logo height (1.0 =
-- unscaled, auto-matches the EEG address block height as before). Range
-- enforced in the handler is 0.5-2.5: the recipient/"Rechnung an" block in
-- the themed renderer sits at a fixed y=45 and does not move when the logo
-- grows, so scaling towards the top of that range risks the logo overlapping
-- it on EEGs with a long (4-line) address and high invoice_row_spacing.
ALTER TABLE eegs ADD COLUMN invoice_logo_scale numeric(3,2) NOT NULL DEFAULT 1.0;
