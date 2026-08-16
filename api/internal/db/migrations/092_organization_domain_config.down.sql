ALTER TABLE organizations DROP CONSTRAINT organizations_custom_domain_key;
ALTER TABLE organizations DROP COLUMN custom_domain;
ALTER TABLE organizations DROP COLUMN portal_base_url;
