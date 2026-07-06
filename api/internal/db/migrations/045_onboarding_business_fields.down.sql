ALTER TABLE onboarding_requests
    DROP COLUMN IF EXISTS business_role,
    DROP COLUMN IF EXISTS uid_nummer,
    DROP COLUMN IF EXISTS use_vat;
