ALTER TABLE jobs ADD COLUMN next_attempt_at timestamptz NOT NULL DEFAULT now();
