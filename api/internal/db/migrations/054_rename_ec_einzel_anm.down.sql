UPDATE eda_processes
SET process_type = 'EC_EINZEL_ANM'
WHERE process_type = 'EC_REQ_ONL';

UPDATE jobs
SET type = 'eda.EC_EINZEL_ANM'
WHERE type = 'eda.EC_REQ_ONL'
  AND status = 'pending';
