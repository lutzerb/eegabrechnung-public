-- Gap 1: member lifecycle status
ALTER TABLE members ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'ACTIVE';

-- Gap 2: configurable starting invoice number
ALTER TABLE eegs ADD COLUMN IF NOT EXISTS invoice_number_start int NOT NULL DEFAULT 1;
