-- Bank statement import for E/A payment matching
CREATE TABLE IF NOT EXISTS ea_banktransaktionen (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id              UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    import_am           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    import_format       VARCHAR(20) NOT NULL DEFAULT 'MT940', -- MT940 | CAMT053 | CSV
    konto_iban          VARCHAR(34),
    buchungsdatum       DATE        NOT NULL,
    valutadatum         DATE,
    betrag              DECIMAL(12,4) NOT NULL,   -- positive = incoming, negative = outgoing
    waehrung            CHAR(3)     NOT NULL DEFAULT 'EUR',
    verwendungszweck    TEXT,
    auftraggeber_empfaenger VARCHAR(200),
    referenz            TEXT,                     -- EndToEndId or bank reference
    matched_buchung_id  UUID REFERENCES ea_buchungen(id) ON DELETE SET NULL,
    match_konfidenz     DECIMAL(5,2),
    match_status        VARCHAR(20) NOT NULL DEFAULT 'offen'  -- offen | auto | bestaetigt | ignoriert
);

CREATE INDEX IF NOT EXISTS ea_banktransaktionen_eeg_id_idx    ON ea_banktransaktionen(eeg_id);
CREATE INDEX IF NOT EXISTS ea_banktransaktionen_status_idx     ON ea_banktransaktionen(eeg_id, match_status);
CREATE INDEX IF NOT EXISTS ea_banktransaktionen_buchung_id_idx ON ea_banktransaktionen(matched_buchung_id);
