DROP TABLE IF EXISTS eda_worker_status;
DROP TABLE IF EXISTS eda_errors;
DROP INDEX IF EXISTS idx_eda_messages_message_id;
ALTER TABLE eda_messages DROP COLUMN IF EXISTS message_id;
ALTER TABLE energy_readings DROP COLUMN IF EXISTS source;
