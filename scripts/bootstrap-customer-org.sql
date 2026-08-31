-- Bootstraps a new customer organization (shared-multi-tenant model — see
-- /home/bernhard/.claude/plans/so-n-chste-woche-haben-glimmering-penguin.md for
-- context). Run manually per new customer; org creation is intentionally not
-- exposed via UI/API since it happens a handful of times a year, not something
-- worth building self-service tooling for.
--
-- Steps:
--   1. Pick org name/slug, the customer's dedicated public domain, and an admin
--      email/password for the first login.
--   2. Add a Cloudflare Tunnel Public Hostname for the domain, pointing at the
--      same eegabrechnung-web:3000 service as onboarding.eegwn.at. If the
--      domain's DNS zone lives in a Cloudflare account you don't control (a
--      customer's own account), a Public Hostname on your tunnel can't reach it —
--      instead create a tunnel *in that account* and add it as another
--      cloudflared-secondary-N service in docker-compose.yaml (copy the
--      cloudflared-secondary-1 block), still pointed at eegabrechnung-web:3000.
--   3. Generate the bcrypt password hash:
--        cd api && go run ./cmd/hashpw '<plaintext-password>'
--   4. Fill in the placeholders below and run against the production DB:
--        docker exec -i eegabrechnung-eegabrechnung-postgres-1 \
--          psql -U eegabrechnung -d eegabrechnung < bootstrap-customer-org.sql
--   5. Verify: log in as the new admin, confirm only the new org's (empty) EEG
--      list is visible.

INSERT INTO organizations (id, name, slug, portal_base_url)
VALUES (
  gen_random_uuid(),
  'CHANGE_ME Kunde',            -- e.g. 'Muster Energie GmbH'
  'change-me',                  -- unique slug, e.g. 'muster-energie'
  'https://change-me.example.at'  -- must match the Cloudflare Public Hostname
)
RETURNING id \gset org_

-- Domain(s) this org resolves for on member-portal login (magic link / password).
-- Add more rows here if the org needs to be reachable under multiple domains.
INSERT INTO organization_domains (organization_id, domain)
VALUES (
  :'org_id',
  'change-me.example.at'        -- host only, no scheme/port; must match portal_base_url above
);

INSERT INTO users (organization_id, email, password_hash, name, role)
VALUES (
  :'org_id',
  'admin@change-me.example.at',
  'CHANGE_ME_BCRYPT_HASH',      -- output of `go run ./cmd/hashpw '<password>'`
  'Administrator',
  'admin'
);
