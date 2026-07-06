-- Invoice settings on EEG
ALTER TABLE eegs
  ADD COLUMN invoice_number_prefix text NOT NULL DEFAULT 'INV',
  ADD COLUMN invoice_number_digits  int  NOT NULL DEFAULT 5,
  ADD COLUMN invoice_pre_text       text NOT NULL DEFAULT '',
  ADD COLUMN invoice_post_text      text NOT NULL DEFAULT '',
  ADD COLUMN invoice_footer_text    text NOT NULL DEFAULT '';

-- Sequential invoice number per EEG
ALTER TABLE invoices
  ADD COLUMN invoice_number int,
  ADD COLUMN status text NOT NULL DEFAULT 'draft';

-- EDA messages table (link to EEG via meter point lookup)
CREATE TABLE IF NOT EXISTS eda_messages (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  direction    text NOT NULL,  -- 'inbound' | 'outbound'
  message_type text NOT NULL,
  subject      text NOT NULL DEFAULT '',
  body         text NOT NULL DEFAULT '',
  processed_at timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now()
);
