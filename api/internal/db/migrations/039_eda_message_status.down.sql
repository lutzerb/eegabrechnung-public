-- Revert: remove 'processed' status (first reset any processed rows back to pending)
UPDATE eda_messages SET status = 'pending' WHERE status = 'processed';
ALTER TABLE eda_messages DROP CONSTRAINT IF EXISTS eda_messages_status_check;
ALTER TABLE eda_messages ADD CONSTRAINT eda_messages_status_check
    CHECK (status = ANY (ARRAY['pending','sent','ack','error']));
