CREATE TABLE tariff_schedules (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id        UUID         NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    granularity   VARCHAR(20)  NOT NULL DEFAULT 'monthly',
    is_active     BOOLEAN      NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Only one active schedule per EEG
CREATE UNIQUE INDEX idx_tariff_schedules_one_active
    ON tariff_schedules(eeg_id)
    WHERE is_active = true;

CREATE TABLE tariff_entries (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id    UUID          NOT NULL REFERENCES tariff_schedules(id) ON DELETE CASCADE,
    valid_from     TIMESTAMPTZ   NOT NULL,
    valid_until    TIMESTAMPTZ   NOT NULL,
    energy_price   NUMERIC(10,4) NOT NULL DEFAULT 0,
    producer_price NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT tariff_entries_valid_range CHECK (valid_until > valid_from),
    UNIQUE (schedule_id, valid_from)
);

CREATE INDEX idx_tariff_entries_lookup
    ON tariff_entries(schedule_id, valid_from, valid_until);
