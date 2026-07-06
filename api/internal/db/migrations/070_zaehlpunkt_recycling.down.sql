DROP INDEX IF EXISTS meter_points_active_zaehlpunkt_uq;
ALTER TABLE meter_points ADD CONSTRAINT meter_points_zaehlpunkt_key UNIQUE (zaehlpunkt);
