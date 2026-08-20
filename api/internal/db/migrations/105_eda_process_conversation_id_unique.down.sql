DROP INDEX IF EXISTS idx_eda_processes_conv;
CREATE INDEX idx_eda_processes_conv ON eda_processes(conversation_id) WHERE conversation_id != '';
