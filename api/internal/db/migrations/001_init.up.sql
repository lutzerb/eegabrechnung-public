CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE eegs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    gemeinschaft_id text UNIQUE NOT NULL,
    netzbetreiber   text NOT NULL,
    name            text NOT NULL,
    energy_price    numeric NOT NULL DEFAULT 0.12,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE members (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id        uuid NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    mitglieds_nr  text NOT NULL,
    name1         text NOT NULL,
    name2         text NOT NULL DEFAULT '',
    email         text NOT NULL,
    iban          text NOT NULL DEFAULT '',
    business_role text NOT NULL DEFAULT 'privat',
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE(eeg_id, mitglieds_nr)
);

CREATE TABLE meter_points (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id             uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    eeg_id                uuid NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    zaehlpunkt            text UNIQUE NOT NULL,
    energierichtung       text NOT NULL,
    verteilungsmodell     text NOT NULL DEFAULT 'DYNAMIC',
    zugeteilte_menge_pct  numeric NOT NULL DEFAULT 0,
    status                text NOT NULL DEFAULT 'ACTIVATED',
    registriert_seit      date,
    created_at            timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE energy_readings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    meter_point_id  uuid NOT NULL REFERENCES meter_points(id) ON DELETE CASCADE,
    ts              timestamptz NOT NULL,
    wh_total        numeric NOT NULL,
    wh_community    numeric NOT NULL,
    wh_self         numeric NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE(meter_point_id, ts)
);

CREATE TABLE invoices (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id    uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    eeg_id       uuid NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    period_start date NOT NULL,
    period_end   date NOT NULL,
    total_kwh    numeric NOT NULL,
    total_amount numeric NOT NULL,
    pdf_path     text NOT NULL DEFAULT '',
    sent_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE jobs (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type       text NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    status     text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
