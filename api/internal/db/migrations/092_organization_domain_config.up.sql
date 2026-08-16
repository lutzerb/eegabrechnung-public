-- Per-organization base URL for links in outbound emails (invoices, onboarding,
-- portal magic links) and, where applicable, the dedicated public domain the
-- organization is reached under. Both are explicit per-org config — the existing
-- default organization (eegwn) is backfilled here just like any future customer
-- organization, not treated as an implicit fallback.

ALTER TABLE organizations ADD COLUMN portal_base_url text;
ALTER TABLE organizations ADD COLUMN custom_domain text;

UPDATE organizations SET portal_base_url = 'https://onboarding.eegwn.at', custom_domain = 'onboarding.eegwn.at'
  WHERE id = '00000000-0000-0000-0000-000000000001';

-- Demo organization: used only for the internal product demo, never sends real
-- outbound email (is_demo blocks sending) and has no dedicated public domain of its
-- own — custom_domain intentionally stays NULL rather than colliding with eegwn's.
UPDATE organizations SET portal_base_url = 'https://onboarding.eegwn.at'
  WHERE id = '00000000-0000-0000-0000-000000000002';

ALTER TABLE organizations ALTER COLUMN portal_base_url SET NOT NULL;
ALTER TABLE organizations ADD CONSTRAINT organizations_custom_domain_key UNIQUE (custom_domain);
