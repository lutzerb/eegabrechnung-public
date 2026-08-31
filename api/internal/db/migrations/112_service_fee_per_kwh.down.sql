ALTER TABLE tariff_schedules
  DROP COLUMN IF EXISTS servicegebuehr_einspeisung_ct_kwh_override,
  DROP COLUMN IF EXISTS servicegebuehr_bezug_ct_kwh_override;

ALTER TABLE eegs
  DROP COLUMN IF EXISTS servicegebuehr_einspeisung_ct_kwh,
  DROP COLUMN IF EXISTS servicegebuehr_bezug_ct_kwh;
