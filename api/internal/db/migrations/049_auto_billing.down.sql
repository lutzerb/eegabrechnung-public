ALTER TABLE eegs
  DROP COLUMN IF EXISTS auto_billing_enabled,
  DROP COLUMN IF EXISTS auto_billing_day_of_month,
  DROP COLUMN IF EXISTS auto_billing_period,
  DROP COLUMN IF EXISTS auto_billing_last_run_at;
