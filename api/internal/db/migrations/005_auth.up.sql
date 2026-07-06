CREATE TABLE organizations (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name       text NOT NULL,
  slug       text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           text NOT NULL UNIQUE,
  password_hash   text NOT NULL,
  name            text NOT NULL,
  role            text NOT NULL DEFAULT 'member',
  created_at      timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE eegs ADD COLUMN organization_id uuid REFERENCES organizations(id);

-- Default organization and admin user (password: admin)
INSERT INTO organizations (id, name, slug)
  VALUES ('00000000-0000-0000-0000-000000000001', 'Standard Organisation', 'default');

INSERT INTO users (organization_id, email, password_hash, name, role)
  VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@eeg.at',
    '$2b$10$M4dWKrtsZDKDm4rjaYA.zOMV1UjxPKl1i3yywaRignsFDSjhRmz9W',
    'Administrator',
    'admin'
  );

-- Assign existing EEGs to the default organization
UPDATE eegs SET organization_id = '00000000-0000-0000-0000-000000000001'
  WHERE organization_id IS NULL;

ALTER TABLE eegs ALTER COLUMN organization_id SET NOT NULL;
