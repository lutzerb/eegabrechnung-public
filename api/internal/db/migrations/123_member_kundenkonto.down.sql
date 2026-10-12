DROP INDEX IF EXISTS members_eeg_kundenkonto_uidx;
ALTER TABLE eegs    DROP COLUMN IF EXISTS kundenkonto_start;
ALTER TABLE members DROP COLUMN IF EXISTS kundenkonto;
