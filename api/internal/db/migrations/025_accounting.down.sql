ALTER TABLE invoices
  DROP COLUMN IF EXISTS net_amount,
  DROP COLUMN IF EXISTS vat_amount,
  DROP COLUMN IF EXISTS vat_pct_applied;

ALTER TABLE eegs
  DROP COLUMN IF EXISTS accounting_revenue_account,
  DROP COLUMN IF EXISTS accounting_expense_account,
  DROP COLUMN IF EXISTS accounting_debitor_prefix,
  DROP COLUMN IF EXISTS datev_consultant_nr,
  DROP COLUMN IF EXISTS datev_client_nr;
