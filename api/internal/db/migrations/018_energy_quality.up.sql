-- Add quality level column to energy_readings.
-- L0 = total/synthetic, L1 = measured (Messwert), L2 = substitute (Ersatzwert), L3 = faulty (Fehlwert).
-- Only L0, L1, and L2 are included in billing; L3 is excluded.
ALTER TABLE energy_readings ADD COLUMN IF NOT EXISTS quality text NOT NULL DEFAULT 'L0';
