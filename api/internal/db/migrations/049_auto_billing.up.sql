ALTER TABLE eegs
  ADD COLUMN auto_billing_enabled boolean NOT NULL DEFAULT false,
  ADD COLUMN auto_billing_day_of_month smallint CHECK (auto_billing_day_of_month BETWEEN 1 AND 28),
  ADD COLUMN auto_billing_period text CHECK (auto_billing_period IN ('monthly','quarterly')),
  ADD COLUMN auto_billing_last_run_at timestamptz;
