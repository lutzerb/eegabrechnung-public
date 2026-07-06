CREATE TABLE email_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id      UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    email_type  TEXT        NOT NULL,
    to_address  TEXT        NOT NULL,
    subject     TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL CHECK (status IN ('sent', 'failed')),
    error_msg   TEXT        NOT NULL DEFAULT '',
    member_id   UUID        REFERENCES members(id) ON DELETE SET NULL,
    invoice_id  UUID        REFERENCES invoices(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX email_log_eeg_created_idx ON email_log (eeg_id, created_at DESC);
