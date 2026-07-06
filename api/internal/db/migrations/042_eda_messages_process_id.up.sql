-- Add optional link from eda_messages back to the eda_process that triggered it.
-- Used to correlate EDASendError gateway responses to their originating process.
ALTER TABLE eda_messages
    ADD COLUMN IF NOT EXISTS eda_process_id uuid REFERENCES eda_processes(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_eda_messages_process_id ON eda_messages(eda_process_id) WHERE eda_process_id IS NOT NULL;
