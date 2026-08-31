-- Split the single "Fixgebühr/Teilnahmegebühr und Zählpunktsgebühr auch bei 0,00 €
-- anzeigen" flag into one independently selectable flag per fee-type line item, and
-- extend it to also cover the Servicegebühr Bezug/Einspeisung lines (migration 112).
-- The Fixgebühr flag keeps its existing values via rename; the three new flags are
-- backfilled from it so an EEG that already opted in keeps showing all four line
-- items at 0,00 € until an admin narrows it down.
ALTER TABLE eegs RENAME COLUMN invoice_show_zero_fees TO invoice_show_zero_fee_fixgebuehr;

ALTER TABLE eegs ADD COLUMN invoice_show_zero_fee_zaehlpunktsgebuehr boolean NOT NULL DEFAULT false;
ALTER TABLE eegs ADD COLUMN invoice_show_zero_fee_servicegebuehr_bezug boolean NOT NULL DEFAULT false;
ALTER TABLE eegs ADD COLUMN invoice_show_zero_fee_servicegebuehr_einspeisung boolean NOT NULL DEFAULT false;

UPDATE eegs SET
  invoice_show_zero_fee_zaehlpunktsgebuehr = true,
  invoice_show_zero_fee_servicegebuehr_bezug = true,
  invoice_show_zero_fee_servicegebuehr_einspeisung = true
WHERE invoice_show_zero_fee_fixgebuehr = true;
