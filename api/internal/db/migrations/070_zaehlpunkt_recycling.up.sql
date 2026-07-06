-- Allow the same Zählpunkt to be reused across members over time (e.g. tenant change).
-- Replace the global UNIQUE constraint with a partial unique index that only enforces
-- uniqueness among *active* (not yet deregistered) meter points per EEG.
ALTER TABLE meter_points DROP CONSTRAINT meter_points_zaehlpunkt_key;

CREATE UNIQUE INDEX meter_points_active_zaehlpunkt_uq
  ON meter_points (eeg_id, zaehlpunkt)
  WHERE abgemeldet_am IS NULL;

-- Allow admins to manually set abgemeldet_am (needed for non-EDA EEGs where the
-- EDA worker never fires CM_REV_SP to set this date automatically).
-- No-op for EDA EEGs where the worker already sets it; the column already exists.
