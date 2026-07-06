-- Add 'processed' as a valid status for inbound EDA messages that have been
-- successfully parsed and imported (CR_MSG, CPDocument, etc.).
ALTER TABLE eda_messages DROP CONSTRAINT IF EXISTS eda_messages_status_check;
ALTER TABLE eda_messages ADD CONSTRAINT eda_messages_status_check
    CHECK (status = ANY (ARRAY['pending','sent','ack','error','processed']));
