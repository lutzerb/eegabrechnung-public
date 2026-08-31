ALTER TABLE eegs DROP COLUMN IF EXISTS invoice_show_zero_fee_servicegebuehr_einspeisung;
ALTER TABLE eegs DROP COLUMN IF EXISTS invoice_show_zero_fee_servicegebuehr_bezug;
ALTER TABLE eegs DROP COLUMN IF EXISTS invoice_show_zero_fee_zaehlpunktsgebuehr;

ALTER TABLE eegs RENAME COLUMN invoice_show_zero_fee_fixgebuehr TO invoice_show_zero_fees;
