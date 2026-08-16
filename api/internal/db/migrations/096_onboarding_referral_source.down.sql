ALTER TABLE onboarding_requests
    DROP COLUMN IF EXISTS referral_source,
    DROP COLUMN IF EXISTS referral_source_note;
