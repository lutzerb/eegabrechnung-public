ALTER TABLE eda_messages
    DROP COLUMN IF EXISTS from_address,
    DROP COLUMN IF EXISTS to_address;
