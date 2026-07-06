ALTER TABLE eda_processes
  ADD COLUMN response_codes       TEXT[]     NOT NULL DEFAULT '{}',
  ADD COLUMN meter_owner_name     TEXT       NOT NULL DEFAULT '',
  ADD COLUMN portal_approval_url  TEXT       NOT NULL DEFAULT '',
  ADD COLUMN customer_notified_at TIMESTAMPTZ;
