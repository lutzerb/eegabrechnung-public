-- Add sender/receiver address columns to EDA messages for display in the protocol.
ALTER TABLE eda_messages
    ADD COLUMN IF NOT EXISTS from_address TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS to_address   TEXT NOT NULL DEFAULT '';
