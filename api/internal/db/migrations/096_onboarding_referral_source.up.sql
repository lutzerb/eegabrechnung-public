-- Optional "how did you hear about us" answer captured on the public onboarding form.
ALTER TABLE onboarding_requests
    ADD COLUMN IF NOT EXISTS referral_source      text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS referral_source_note text NOT NULL DEFAULT '';
