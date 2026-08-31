-- Replaces the single organizations.custom_domain column with a one-to-many
-- organization_domains table. Two problems drove this:
--   1. An org can need more than one public domain that resolves for
--      member-portal login (magic link / password) — a single UNIQUE column
--      can't hold two.
--   2. A customer org's custom_domain was never set during onboarding (only
--      portal_base_url was), so their member-portal login had been hard-failing
--      with a 400 since day one (resolveTenantOrg has no fallback, by design —
--      see member_portal.go).

CREATE TABLE organization_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Carry over every org that already had a custom_domain set (eegwn today).
INSERT INTO organization_domains (organization_id, domain)
SELECT id, LOWER(custom_domain) FROM organizations WHERE custom_domain IS NOT NULL;

-- Backfill orgs whose custom_domain was left NULL even though portal_base_url
-- implies a real domain (the onboarding bug above). The demo org is excluded:
-- its portal_base_url deliberately borrows eegwn's onboarding domain rather than
-- owning one, and must not register itself under it.
INSERT INTO organization_domains (organization_id, domain)
SELECT id, LOWER(regexp_replace(portal_base_url, '^https?://', ''))
FROM organizations
WHERE custom_domain IS NULL
  AND id <> '00000000-0000-0000-0000-000000000002'
ON CONFLICT (domain) DO NOTHING;

ALTER TABLE organizations DROP CONSTRAINT organizations_custom_domain_key;
ALTER TABLE organizations DROP COLUMN custom_domain;
