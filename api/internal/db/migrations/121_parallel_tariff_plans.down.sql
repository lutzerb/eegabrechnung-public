DROP INDEX IF EXISTS idx_members_tariff_schedule;
ALTER TABLE members DROP COLUMN IF EXISTS tariff_schedule_id;

DROP INDEX IF EXISTS idx_tariff_schedules_one_default;

-- Restore "one active EEG-wide plan": keep only the default plan active.
UPDATE tariff_schedules SET is_active = false WHERE member_id IS NULL AND NOT is_default;
CREATE UNIQUE INDEX idx_tariff_schedules_one_active_default
    ON tariff_schedules (eeg_id) WHERE is_active AND member_id IS NULL;

ALTER TABLE tariff_schedules DROP COLUMN IF EXISTS is_default;
