CREATE TABLE member_sepa_mandate_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id   UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    eeg_id      UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    iban        TEXT NOT NULL DEFAULT '',
    signed_at   TIMESTAMPTZ,
    signed_ip   TEXT NOT NULL DEFAULT '',
    signed_text TEXT NOT NULL DEFAULT '',
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_member_sepa_mandate_history_member
  ON member_sepa_mandate_history(member_id, archived_at DESC);
