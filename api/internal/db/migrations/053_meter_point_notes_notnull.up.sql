UPDATE meter_points SET notes = '' WHERE notes IS NULL;
ALTER TABLE meter_points ALTER COLUMN notes SET NOT NULL;
ALTER TABLE meter_points ALTER COLUMN notes SET DEFAULT '';
