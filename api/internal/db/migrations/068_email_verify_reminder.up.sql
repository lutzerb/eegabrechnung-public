ALTER TABLE onboarding_email_verifications
    ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMPTZ;
