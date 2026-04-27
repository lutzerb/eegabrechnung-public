-- Align eda_messages columns with what the repository expects, and add
-- EDA communication configuration to eegs.

-- Add display-friendly columns to eda_messages.
ALTER TABLE eda_messages
    ADD COLUMN IF NOT EXISTS message_type text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS subject      text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS body         text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS processed_at timestamptz;

-- Backfill message_type from the existing process column.
UPDATE eda_messages SET message_type = process WHERE message_type = '';

-- EDA communication configuration on eegs.
ALTER TABLE eegs
    ADD COLUMN IF NOT EXISTS eda_transition_date  date,
    ADD COLUMN IF NOT EXISTS eda_marktpartner_id  text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS eda_netzbetreiber_id text NOT NULL DEFAULT '';
