-- Tracking table for open EDA processes (Anmeldung, Abmeldung, Teilnahmefaktor).
CREATE TABLE eda_processes (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id               uuid NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    meter_point_id       uuid REFERENCES meter_points(id) ON DELETE SET NULL,
    process_type         text NOT NULL,            -- EC_REQ_ONL, EC_EINZEL_ANM, EC_EINZEL_ABM, EC_PRTFACT_CHG
    status               text NOT NULL DEFAULT 'pending',
        -- pending → sent → first_confirmed → confirmed → completed
        -- rejected, error
    conversation_id      text NOT NULL DEFAULT '', -- links outbound request to inbound confirmations
    zaehlpunkt           text NOT NULL,
    valid_from           date,
    participation_factor numeric,                  -- for EC_PRTFACT_CHG / ANM
    share_type           text NOT NULL DEFAULT '', -- GC, RC_R, RC_L, CC, NONE, MULTI
    initiated_at         timestamptz NOT NULL DEFAULT now(),
    deadline_at          timestamptz,              -- 2 months for Anmeldung (EAG §16e)
    completed_at         timestamptz,
    error_msg            text NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_eda_processes_eeg ON eda_processes(eeg_id, status);
CREATE INDEX idx_eda_processes_conv ON eda_processes(conversation_id)
    WHERE conversation_id != '';
