CREATE TABLE onboarding_email_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  eeg_id UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  name1 TEXT NOT NULL DEFAULT '',
  name2 TEXT NOT NULL DEFAULT '',
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON onboarding_email_verifications(token);
