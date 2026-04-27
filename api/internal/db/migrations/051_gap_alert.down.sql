ALTER TABLE meter_points DROP COLUMN IF EXISTS gap_alert_sent_at;

ALTER TABLE eegs
  DROP COLUMN IF EXISTS gap_alert_enabled,
  DROP COLUMN IF EXISTS gap_alert_threshold_days;
