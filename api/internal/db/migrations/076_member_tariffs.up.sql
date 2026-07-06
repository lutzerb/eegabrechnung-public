ALTER TABLE tariff_schedules
  ADD COLUMN member_id                      UUID REFERENCES members(id) ON DELETE CASCADE,
  ADD COLUMN free_kwh_override               NUMERIC(10,4),
  ADD COLUMN discount_pct_override            NUMERIC(5,2),
  ADD COLUMN meter_fee_eur_override           NUMERIC(10,4),
  ADD COLUMN participation_fee_eur_override   NUMERIC(10,4),
  ADD COLUMN zaehlpunkts_gebuehr_eur_override NUMERIC(10,4);

DROP INDEX idx_tariff_schedules_one_active;

CREATE UNIQUE INDEX idx_tariff_schedules_one_active_default
  ON tariff_schedules(eeg_id)
  WHERE is_active = true AND member_id IS NULL;

CREATE UNIQUE INDEX idx_tariff_schedules_one_active_member
  ON tariff_schedules(eeg_id, member_id)
  WHERE is_active = true AND member_id IS NOT NULL;

ALTER TABLE eegs
  ADD COLUMN zaehlpunkts_gebuehr_eur NUMERIC(10,4) NOT NULL DEFAULT 0;
