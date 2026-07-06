ALTER TABLE eegs DROP COLUMN IF EXISTS zaehlpunkts_gebuehr_eur;

DROP INDEX IF EXISTS idx_tariff_schedules_one_active_member;
DROP INDEX IF EXISTS idx_tariff_schedules_one_active_default;

CREATE UNIQUE INDEX idx_tariff_schedules_one_active
  ON tariff_schedules(eeg_id)
  WHERE is_active = true;

ALTER TABLE tariff_schedules
  DROP COLUMN IF EXISTS zaehlpunkts_gebuehr_eur_override,
  DROP COLUMN IF EXISTS participation_fee_eur_override,
  DROP COLUMN IF EXISTS meter_fee_eur_override,
  DROP COLUMN IF EXISTS discount_pct_override,
  DROP COLUMN IF EXISTS free_kwh_override,
  DROP COLUMN IF EXISTS member_id;
