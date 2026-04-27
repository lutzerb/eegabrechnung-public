-- Add retry counter to jobs table for SMTP/outbound retry logic.
ALTER TABLE jobs ADD COLUMN retry_count int NOT NULL DEFAULT 0;
