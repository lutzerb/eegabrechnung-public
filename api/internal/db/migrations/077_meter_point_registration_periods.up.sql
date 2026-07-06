CREATE TABLE meter_point_registration_periods (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id            UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    zaehlpunkt        TEXT NOT NULL,
    meter_point_id    UUID REFERENCES meter_points(id) ON DELETE SET NULL,
    registriert_seit  DATE NOT NULL,
    abgemeldet_am     DATE,
    consent_id        TEXT NOT NULL DEFAULT '',
    source            TEXT NOT NULL DEFAULT 'eda',
    reason            TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_meter_point_registration_periods_zp
  ON meter_point_registration_periods(eeg_id, zaehlpunkt, registriert_seit);

CREATE UNIQUE INDEX meter_point_registration_periods_one_open_uq
  ON meter_point_registration_periods(eeg_id, zaehlpunkt)
  WHERE abgemeldet_am IS NULL;

-- Backfill one period per existing meter point that has been confirmed at least once,
-- reflecting its current registriert_seit/abgemeldet_am snapshot.
INSERT INTO meter_point_registration_periods
    (eeg_id, zaehlpunkt, meter_point_id, registriert_seit, abgemeldet_am, consent_id, source)
SELECT eeg_id, zaehlpunkt, id, registriert_seit, abgemeldet_am, consent_id, 'backfill'
FROM meter_points
WHERE registriert_seit IS NOT NULL;
