-- Add business type and VAT fields to onboarding_requests
ALTER TABLE onboarding_requests
    ADD COLUMN IF NOT EXISTS business_role text NOT NULL DEFAULT 'privat',
    ADD COLUMN IF NOT EXISTS uid_nummer    text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS use_vat       boolean NOT NULL DEFAULT false;
