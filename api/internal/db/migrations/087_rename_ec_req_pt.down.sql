UPDATE eda_processes
SET process_type = 'EC_REQ_PT'
WHERE process_type = 'CR_REQ_PT';

UPDATE eda_messages
SET message_type = 'EC_REQ_PT'
WHERE message_type = 'CR_REQ_PT';

UPDATE jobs
SET type = 'eda.EC_REQ_PT',
    payload = jsonb_set(payload, '{process}', '"EC_REQ_PT"')
WHERE type = 'eda.CR_REQ_PT'
  AND status = 'pending';
