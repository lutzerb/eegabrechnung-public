ALTER TABLE eda_messages
    DROP COLUMN IF EXISTS message_type,
    DROP COLUMN IF EXISTS subject,
    DROP COLUMN IF EXISTS body,
    DROP COLUMN IF EXISTS processed_at;

ALTER TABLE eegs
    DROP COLUMN IF EXISTS eda_transition_date,
    DROP COLUMN IF EXISTS eda_marktpartner_id,
    DROP COLUMN IF EXISTS eda_netzbetreiber_id;
