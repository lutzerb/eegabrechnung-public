ALTER TABLE invoices DROP COLUMN IF EXISTS billing_run_id;
DROP TABLE IF EXISTS billing_runs;
