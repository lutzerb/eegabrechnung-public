CREATE TABLE member_email_campaigns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  eeg_id UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
  subject TEXT NOT NULL,
  html_body TEXT NOT NULL,
  recipient_count INT NOT NULL DEFAULT 0,
  attachments_json JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON member_email_campaigns(eeg_id, created_at DESC);
