ALTER TABLE eegs
  ADD COLUMN servicegebuehr_bezug_ct_kwh       NUMERIC(10,4) NOT NULL DEFAULT 0,
  ADD COLUMN servicegebuehr_einspeisung_ct_kwh NUMERIC(10,4) NOT NULL DEFAULT 0;

ALTER TABLE tariff_schedules
  ADD COLUMN servicegebuehr_bezug_ct_kwh_override       NUMERIC(10,4),
  ADD COLUMN servicegebuehr_einspeisung_ct_kwh_override NUMERIC(10,4);
