CREATE TABLE member_portal_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id        UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    eeg_id           UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    link_token       VARCHAR(64) UNIQUE NOT NULL,
    session_token    VARCHAR(64) UNIQUE,
    link_used_at     TIMESTAMPTZ,
    link_expires_at  TIMESTAMPTZ NOT NULL,
    session_expires_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON member_portal_sessions(link_token);
CREATE INDEX ON member_portal_sessions(session_token);
