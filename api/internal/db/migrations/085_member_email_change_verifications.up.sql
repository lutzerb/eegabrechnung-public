CREATE TABLE member_email_change_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  eeg_id UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
  new_email TEXT NOT NULL,
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON member_email_change_verifications(token);
CREATE UNIQUE INDEX idx_member_email_change_pending_unique
  ON member_email_change_verifications(member_id) WHERE verified_at IS NULL;
