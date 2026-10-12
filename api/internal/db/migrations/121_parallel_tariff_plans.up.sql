-- Parallel tariff plans: several EEG-wide plans may be active at once, exactly one is the
-- default (Standardtarif); members can be assigned to one of the active plans.
ALTER TABLE tariff_schedules ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT false;

UPDATE tariff_schedules SET is_default = true WHERE member_id IS NULL AND is_active = true;

DROP INDEX IF EXISTS idx_tariff_schedules_one_active_default;
CREATE UNIQUE INDEX idx_tariff_schedules_one_default
    ON tariff_schedules (eeg_id) WHERE is_default AND member_id IS NULL;

ALTER TABLE members
    ADD COLUMN tariff_schedule_id UUID REFERENCES tariff_schedules(id) ON DELETE SET NULL;
CREATE INDEX idx_members_tariff_schedule ON members (tariff_schedule_id) WHERE tariff_schedule_id IS NOT NULL;
