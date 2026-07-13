-- Rename EC_REQ_PT → CR_REQ_PT throughout.
-- CR_REQ_PT is the canonical name per ebutilities.at process list; EC_REQ_PT was an
-- internal misnomer — the wire format (Subject Prozess-Id CR_REQ_PT_04.10, MessageCode
-- ANFORDERUNG_PT) was always correct. Mirrors migration 054 (EC_EINZEL_ANM → EC_REQ_ONL).

UPDATE eda_processes
SET process_type = 'CR_REQ_PT'
WHERE process_type = 'EC_REQ_PT';

-- Historical message log entries: message_type mirrors the process name for outbound messages.
UPDATE eda_messages
SET message_type = 'CR_REQ_PT'
WHERE message_type = 'EC_REQ_PT';

-- Rename pending jobs (not yet claimed by the worker). Both the job type and the
-- payload's process field must change — the worker looks up the edanet Subject
-- Prozess-Id by payload->>'process'.
UPDATE jobs
SET type = 'eda.CR_REQ_PT',
    payload = jsonb_set(payload, '{process}', '"CR_REQ_PT"')
WHERE type = 'eda.EC_REQ_PT'
  AND status = 'pending';
