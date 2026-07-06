-- Migration 016: EDA completeness gaps
-- Covers: source tracking on readings, message dedup key, dead-letter errors table,
--         worker status singleton.

-- 1. Source column on energy_readings (xlsx | eda)
ALTER TABLE energy_readings
    ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'xlsx';

-- 2. MessageID deduplication on eda_messages
ALTER TABLE eda_messages
    ADD COLUMN IF NOT EXISTS message_id text NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_eda_messages_message_id
    ON eda_messages(message_id) WHERE message_id != '';

-- 3. Dead-letter table for failed inbound EDA messages
CREATE TABLE IF NOT EXISTS eda_errors (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id       uuid        REFERENCES eegs(id) ON DELETE SET NULL,
    direction    text        NOT NULL DEFAULT 'inbound',
    message_type text        NOT NULL DEFAULT '',
    raw_content  text        NOT NULL DEFAULT '',
    error_msg    text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_eda_errors_eeg ON eda_errors(eeg_id, created_at DESC);

-- 4. Worker status singleton (id=1 always; UPSERT from worker on each poll)
CREATE TABLE IF NOT EXISTS eda_worker_status (
    id             integer     PRIMARY KEY DEFAULT 1,
    transport_mode text        NOT NULL DEFAULT '',
    last_poll_at   timestamptz,
    last_error     text        NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);
INSERT INTO eda_worker_status (id) VALUES (1) ON CONFLICT DO NOTHING;
