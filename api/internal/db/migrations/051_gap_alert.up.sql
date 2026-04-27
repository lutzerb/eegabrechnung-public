-- Migration 051: Gap alert settings on eegs + sent-at tracking on meter_points

ALTER TABLE eegs
  ADD COLUMN gap_alert_enabled boolean NOT NULL DEFAULT true,
  ADD COLUMN gap_alert_threshold_days smallint NOT NULL DEFAULT 5;

ALTER TABLE meter_points
  ADD COLUMN gap_alert_sent_at timestamptz;
