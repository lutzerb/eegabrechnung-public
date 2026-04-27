-- Gap 12: Mehrfachteilnahme — a single meter point may participate in up to 5 EEGs
-- simultaneously (Austrian EAG as of April 2024).
--
-- This table records which meter point participates in which EEG, with what factor
-- and for which time range.  It is the source of truth for EDA registration messages.
-- Billing itself still uses wh_community from energy_readings (set by grid operator).

CREATE TABLE IF NOT EXISTS eeg_meter_participations (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id               uuid NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    meter_point_id       uuid NOT NULL REFERENCES meter_points(id) ON DELETE CASCADE,
    participation_factor numeric(7,4) NOT NULL DEFAULT 100.0 CHECK (participation_factor > 0 AND participation_factor <= 100),
    share_type           text NOT NULL DEFAULT 'GC',   -- GC | RC_R | RC_L | CC
    valid_from           date NOT NULL,
    valid_until          date,                          -- NULL = open-ended (active)
    notes                text NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_eeg_meter_participations_eeg     ON eeg_meter_participations(eeg_id);
CREATE INDEX IF NOT EXISTS idx_eeg_meter_participations_mp      ON eeg_meter_participations(meter_point_id);
CREATE INDEX IF NOT EXISTS idx_eeg_meter_participations_active  ON eeg_meter_participations(eeg_id, meter_point_id, valid_from);
