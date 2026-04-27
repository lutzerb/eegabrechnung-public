ALTER TABLE invoices
  DROP COLUMN IF EXISTS consumption_net_amount,
  DROP COLUMN IF EXISTS generation_net_amount;
