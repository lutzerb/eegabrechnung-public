-- Track when (and if) the 72-hour follow-up reminder was sent.
ALTER TABLE onboarding_requests
    ADD COLUMN IF NOT EXISTS reminder_sent_at timestamptz;
