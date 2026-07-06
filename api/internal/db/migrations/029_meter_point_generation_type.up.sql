-- Add generation_type to meter_points for Einspeisung meters (PV, Windkraft, Wasserkraft).
ALTER TABLE meter_points ADD COLUMN generation_type text;

-- Default all existing GENERATION meter points to PV.
UPDATE meter_points SET generation_type = 'PV' WHERE energierichtung = 'GENERATION';
