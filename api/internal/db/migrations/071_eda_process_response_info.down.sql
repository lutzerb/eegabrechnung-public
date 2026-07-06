ALTER TABLE eda_processes
  DROP COLUMN IF EXISTS response_codes,
  DROP COLUMN IF EXISTS meter_owner_name,
  DROP COLUMN IF EXISTS portal_approval_url,
  DROP COLUMN IF EXISTS customer_notified_at;
