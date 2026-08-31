ALTER TABLE organizations ADD COLUMN custom_domain text;

-- Best-effort: if an org gained more than one domain, only one arbitrary domain
-- survives the downgrade back to a single column.
UPDATE organizations o
SET custom_domain = od.domain
FROM organization_domains od
WHERE od.organization_id = o.id;

ALTER TABLE organizations ADD CONSTRAINT organizations_custom_domain_key UNIQUE (custom_domain);

DROP TABLE organization_domains;
