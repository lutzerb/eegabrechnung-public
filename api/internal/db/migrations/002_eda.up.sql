-- EDA message store
CREATE TABLE eda_messages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id      uuid REFERENCES eegs(id) ON DELETE SET NULL,
    process     text NOT NULL,
    direction   text NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'ack', 'error')),
    xml_payload text NOT NULL,
    error_msg   text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Index for worker polling of pending jobs
CREATE INDEX idx_jobs_pending ON jobs(status, created_at) WHERE status = 'pending';

-- Index for eda_messages polling
CREATE INDEX idx_eda_messages_status ON eda_messages(status, created_at);
