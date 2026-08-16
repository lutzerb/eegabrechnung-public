-- Zusatzzähler: manually-read submeters that are NOT Netzbetreiber smart meters
-- (e.g. Wärmepumpe, Werkstatt) and never receive EDA/Zählpunkt data. Deliberately
-- separate from meter_points/energy_readings — those are tightly coupled to the
-- EDA workflow (Zählpunkt uniqueness, gap-checker, BEG NB-routing) which does not
-- apply here. Billed via a Zählerstand (cumulative counter) diff between readings,
-- priced at the member's normal consumption price (see billing.go).
CREATE TABLE extra_meters (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id     UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    member_id  UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    label      TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'ACTIVE',
    notes      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_extra_meters_member ON extra_meters(member_id);
CREATE INDEX idx_extra_meters_eeg ON extra_meters(eeg_id);

CREATE TABLE extra_meter_readings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    extra_meter_id UUID NOT NULL REFERENCES extra_meters(id) ON DELETE CASCADE,
    reading_date   DATE NOT NULL,
    counter_value  NUMERIC(14,3) NOT NULL,
    notes          TEXT NOT NULL DEFAULT '',
    created_by     UUID REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (extra_meter_id, reading_date)
);

CREATE INDEX idx_extra_meter_readings_meter_date
  ON extra_meter_readings(extra_meter_id, reading_date DESC);
