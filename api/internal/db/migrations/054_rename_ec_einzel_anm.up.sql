-- Rename EC_EINZEL_ANM → EC_REQ_ONL throughout.
-- Both names produced identical XML (CMRequest ANFORDERUNG_ECON, Subject EC_REQ_ONL_02.30).
-- EC_REQ_ONL is the canonical name per ebutilities.at process list.

UPDATE eda_processes
SET process_type = 'EC_REQ_ONL'
WHERE process_type = 'EC_EINZEL_ANM';

-- Rename pending jobs (not yet sent — worker hasn't claimed them yet).
-- Sent/done jobs are no longer processed by the worker, so no update needed.
UPDATE jobs
SET type = 'eda.EC_REQ_ONL'
WHERE type = 'eda.EC_EINZEL_ANM'
  AND status = 'pending';
