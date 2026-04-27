-- E/A-Buchhaltung (Einnahmen-Ausgaben-Rechnung) für EEG-Verein
-- IST-Prinzip: Buchungsdatum = Zahlungsdatum

-- EA settings on eegs (separate from existing accounting/DATEV fields)
ALTER TABLE eegs
  ADD COLUMN IF NOT EXISTS ea_uva_periodentyp VARCHAR(10) NOT NULL DEFAULT 'QUARTAL',
  ADD COLUMN IF NOT EXISTS ea_steuernummer VARCHAR(30),
  ADD COLUMN IF NOT EXISTS ea_finanzamt VARCHAR(100);

-- Chart of accounts (Kontenplan), fully editable per EEG
CREATE TABLE IF NOT EXISTS ea_konten (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id      UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    nummer      VARCHAR(10) NOT NULL,
    name        VARCHAR(200) NOT NULL,
    typ         VARCHAR(20)  NOT NULL DEFAULT 'AUSGABE',     -- EINNAHME | AUSGABE | SONSTIG
    ust_relevanz VARCHAR(20) NOT NULL DEFAULT 'KEINE',       -- KEINE | STEUERBAR | VST | RC
    standard_ust_pct DECIMAL(5,2),
    uva_kz      VARCHAR(10),
    sortierung  INT         NOT NULL DEFAULT 0,
    aktiv       BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(eeg_id, nummer)
);

-- Bookings journal (one row = one cash movement, IST-Prinzip)
CREATE TABLE IF NOT EXISTS ea_buchungen (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id          UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    geschaeftsjahr  INT         NOT NULL,
    buchungsnr      VARCHAR(20),
    zahlung_datum   DATE,                   -- NULL = pending payment (not yet recorded)
    beleg_datum     DATE,
    belegnr         VARCHAR(100),
    beschreibung    TEXT        NOT NULL DEFAULT '',
    konto_id        UUID        NOT NULL REFERENCES ea_konten(id),
    richtung        VARCHAR(10) NOT NULL,   -- EINNAHME | AUSGABE
    betrag_brutto   DECIMAL(12,4) NOT NULL DEFAULT 0,
    ust_code        VARCHAR(20) NOT NULL DEFAULT 'KEINE',
    ust_pct         DECIMAL(5,2),
    ust_betrag      DECIMAL(12,4) NOT NULL DEFAULT 0,
    betrag_netto    DECIMAL(12,4) NOT NULL DEFAULT 0,
    gegenseite      VARCHAR(200),
    quelle          VARCHAR(20) NOT NULL DEFAULT 'manual',   -- manual | eeg_rechnung | eeg_gutschrift | bankimport
    quelle_id       UUID,                   -- → invoices.id or ea_banktransaktionen.id
    beleg_id        UUID,                   -- → ea_belege.id (set after beleg upload)
    notizen         TEXT,
    erstellt_von    UUID REFERENCES users(id),
    erstellt_am     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    aktualisiert_am TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Document storage (Belegverwaltung)
CREATE TABLE IF NOT EXISTS ea_belege (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id          UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    buchung_id      UUID        REFERENCES ea_buchungen(id) ON DELETE SET NULL,
    dateiname       TEXT        NOT NULL,
    pfad            TEXT        NOT NULL,
    groesse         INT,
    mime_typ        VARCHAR(100),
    beschreibung    TEXT,
    hochgeladen_am  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    hochgeladen_von UUID REFERENCES users(id)
);

-- UVA periods (Umsatzsteuervoranmeldung tracking)
CREATE TABLE IF NOT EXISTS ea_uva_perioden (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id      UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    jahr        INT         NOT NULL,
    periodentyp VARCHAR(10) NOT NULL DEFAULT 'QUARTAL',  -- MONAT | QUARTAL
    periode_nr  INT         NOT NULL,                    -- 1-12 or 1-4
    datum_von   DATE        NOT NULL,
    datum_bis   DATE        NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'entwurf',  -- entwurf | eingereicht
    -- Kennzahlen (cached after calculation)
    kz_000      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- Gesamtumsatz
    kz_022      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- steuerpflichtige Umsätze (Basis)
    kz_029      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- Summe Bemessungsgrundlagen
    kz_056      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- Ausgangs-USt
    kz_060      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- Vorsteuer gesamt
    kz_065      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- Vorsteuer RC (§19)
    kz_066      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- Steuerschuld RC
    kz_083      DECIMAL(12,2) NOT NULL DEFAULT 0,  -- RC Bemessungsgrundlage
    zahllast    DECIMAL(12,2) NOT NULL DEFAULT 0,
    eingereicht_am TIMESTAMPTZ,
    erstellt_am TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(eeg_id, jahr, periodentyp, periode_nr)
);

-- Indices
CREATE INDEX IF NOT EXISTS ea_konten_eeg_id_idx           ON ea_konten(eeg_id);
CREATE INDEX IF NOT EXISTS ea_buchungen_eeg_id_idx        ON ea_buchungen(eeg_id);
CREATE INDEX IF NOT EXISTS ea_buchungen_geschaeftsjahr_idx ON ea_buchungen(eeg_id, geschaeftsjahr);
CREATE INDEX IF NOT EXISTS ea_buchungen_zahlung_datum_idx  ON ea_buchungen(eeg_id, zahlung_datum);
CREATE INDEX IF NOT EXISTS ea_belege_buchung_id_idx       ON ea_belege(buchung_id);
CREATE INDEX IF NOT EXISTS ea_uva_perioden_eeg_id_idx     ON ea_uva_perioden(eeg_id);
