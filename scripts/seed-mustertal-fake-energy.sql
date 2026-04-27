-- Seed fake monthly energy readings for the "Sonnenstrom Mustertal" demo/test EEG.
-- Purpose: populate the reports page with plausible-looking data for screenshots
-- and local demos, aligned with the current report semantics:
--   CONSUMPTION wh_self      = EEG share consumed ("Ausgetauscht")
--   GENERATION  wh_community = EEG share fed into the EEG pool
--
-- The script is intentionally narrow: it only touches four known meter points of
-- the Mustertal EEG and only for March 2026.

BEGIN;

DELETE FROM energy_readings
WHERE meter_point_id IN (
  '76ed3810-b8b4-44e0-a73c-35571165f1de', -- Hans Mustermann (consumption)
  'f4c1b54a-144c-43f8-8932-eff4cec79a92', -- Maria Sonnleitner (consumption)
  '5180a6bd-d1ed-4726-bb71-9d21fe353cd0', -- Maria Sonnleitner (generation)
  '1615fa97-da50-469b-b783-0b9e0654f8e9'  -- Biobauernhof Grünwald GmbH (generation)
)
AND ts >= TIMESTAMPTZ '2026-03-01 00:00:00+00'
AND ts <  TIMESTAMPTZ '2026-04-01 00:00:00+00';

WITH days AS (
  SELECT generate_series(DATE '2026-03-01', DATE '2026-03-31', INTERVAL '1 day')::date AS d
),
weather AS (
  SELECT
    d,
    EXTRACT(DAY FROM d)::numeric AS day_no,
    CASE WHEN EXTRACT(ISODOW FROM d) IN (6, 7) THEN 1::numeric ELSE 0::numeric END AS weekend,
    GREATEST(
      0.18::numeric,
      LEAST(
        1.00::numeric,
        0.55
        + 0.28 * sin((EXTRACT(DAY FROM d)::numeric / 31) * 2 * pi() - 0.9)
        + 0.09 * cos(EXTRACT(DAY FROM d)::numeric * 0.61)
      )
    ) AS solar_idx
  FROM days
),
readings AS (
  -- Hans Mustermann — consumption meter
  SELECT
    '76ed3810-b8b4-44e0-a73c-35571165f1de'::uuid AS meter_point_id,
    (d::timestamptz + INTERVAL '12 hours') AS ts,
    ROUND((10.8 + 1.4 * sin(day_no * 0.42) + 0.8 * cos(day_no * 0.17) + weekend * 1.0)::numeric, 3) AS wh_total,
    ROUND((LEAST(
      (10.8 + 1.4 * sin(day_no * 0.42) + 0.8 * cos(day_no * 0.17) + weekend * 1.0)::numeric,
      GREATEST(
        2.4::numeric,
        (10.8 + 1.4 * sin(day_no * 0.42) + 0.8 * cos(day_no * 0.17) + weekend * 1.0)::numeric
        * (0.46 + 0.26 * solar_idx + 0.04 * cos(day_no * 0.35))
      )
    ))::numeric, 3) AS wh_self,
    ROUND((LEAST(
      (10.8 + 1.4 * sin(day_no * 0.42) + 0.8 * cos(day_no * 0.17) + weekend * 1.0)::numeric,
      GREATEST(
        2.4::numeric,
        (10.8 + 1.4 * sin(day_no * 0.42) + 0.8 * cos(day_no * 0.17) + weekend * 1.0)::numeric
        * (0.50 + 0.24 * solar_idx + 0.03 * sin(day_no * 0.22))
      )
    ))::numeric, 3) AS wh_community
  FROM weather

  UNION ALL

  -- Maria Sonnleitner — consumption meter
  SELECT
    'f4c1b54a-144c-43f8-8932-eff4cec79a92'::uuid AS meter_point_id,
    (d::timestamptz + INTERVAL '12 hours') AS ts,
    ROUND((8.1 + 1.1 * sin(day_no * 0.31 + 0.4) + 0.7 * cos(day_no * 0.19) + weekend * 0.6)::numeric, 3) AS wh_total,
    ROUND((LEAST(
      (8.1 + 1.1 * sin(day_no * 0.31 + 0.4) + 0.7 * cos(day_no * 0.19) + weekend * 0.6)::numeric,
      GREATEST(
        1.9::numeric,
        (8.1 + 1.1 * sin(day_no * 0.31 + 0.4) + 0.7 * cos(day_no * 0.19) + weekend * 0.6)::numeric
        * (0.48 + 0.28 * solar_idx + 0.03 * sin(day_no * 0.41))
      )
    ))::numeric, 3) AS wh_self,
    ROUND((LEAST(
      (8.1 + 1.1 * sin(day_no * 0.31 + 0.4) + 0.7 * cos(day_no * 0.19) + weekend * 0.6)::numeric,
      GREATEST(
        1.9::numeric,
        (8.1 + 1.1 * sin(day_no * 0.31 + 0.4) + 0.7 * cos(day_no * 0.19) + weekend * 0.6)::numeric
        * (0.52 + 0.24 * solar_idx + 0.04 * cos(day_no * 0.27))
      )
    ))::numeric, 3) AS wh_community
  FROM weather

  UNION ALL

  -- Maria Sonnleitner — PV generation meter
  SELECT
    '5180a6bd-d1ed-4726-bb71-9d21fe353cd0'::uuid AS meter_point_id,
    (d::timestamptz + INTERVAL '12 hours') AS ts,
    ROUND((11.5 + 10.8 * solar_idx + 1.4 * sin(day_no * 0.36))::numeric, 3) AS wh_total,
    ROUND((LEAST(
      (11.5 + 10.8 * solar_idx + 1.4 * sin(day_no * 0.36))::numeric,
      GREATEST(
        2.8::numeric,
        (11.5 + 10.8 * solar_idx + 1.4 * sin(day_no * 0.36))::numeric
        * (0.30 + 0.15 * cos(day_no * 0.44) + 0.08 * weekend)
      )
    ))::numeric, 3) AS wh_community,
    ROUND((GREATEST(
      0::numeric,
      (11.5 + 10.8 * solar_idx + 1.4 * sin(day_no * 0.36))::numeric
      - LEAST(
          (11.5 + 10.8 * solar_idx + 1.4 * sin(day_no * 0.36))::numeric,
          GREATEST(
            2.8::numeric,
            (11.5 + 10.8 * solar_idx + 1.4 * sin(day_no * 0.36))::numeric
            * (0.30 + 0.15 * cos(day_no * 0.44) + 0.08 * weekend)
          )
        )
    ))::numeric, 3) AS wh_self
  FROM weather

  UNION ALL

  -- Biobauernhof Grünwald GmbH — larger PV generation meter
  SELECT
    '1615fa97-da50-469b-b783-0b9e0654f8e9'::uuid AS meter_point_id,
    (d::timestamptz + INTERVAL '12 hours') AS ts,
    ROUND((29.0 + 26.0 * solar_idx + 2.6 * cos(day_no * 0.33))::numeric, 3) AS wh_total,
    ROUND((LEAST(
      (29.0 + 26.0 * solar_idx + 2.6 * cos(day_no * 0.33))::numeric,
      GREATEST(
        8.0::numeric,
        (29.0 + 26.0 * solar_idx + 2.6 * cos(day_no * 0.33))::numeric
        * (0.36 + 0.14 * sin(day_no * 0.28) + 0.05 * weekend)
      )
    ))::numeric, 3) AS wh_community,
    ROUND((GREATEST(
      0::numeric,
      (29.0 + 26.0 * solar_idx + 2.6 * cos(day_no * 0.33))::numeric
      - LEAST(
          (29.0 + 26.0 * solar_idx + 2.6 * cos(day_no * 0.33))::numeric,
          GREATEST(
            8.0::numeric,
            (29.0 + 26.0 * solar_idx + 2.6 * cos(day_no * 0.33))::numeric
            * (0.36 + 0.14 * sin(day_no * 0.28) + 0.05 * weekend)
          )
        )
    ))::numeric, 3) AS wh_self
  FROM weather
)
INSERT INTO energy_readings (
  meter_point_id, ts, wh_total, wh_community, wh_self, source, quality
)
SELECT
  meter_point_id,
  ts,
  wh_total,
  wh_community,
  wh_self,
  'xlsx' AS source,
  'L0' AS quality
FROM readings
ON CONFLICT (meter_point_id, ts) DO UPDATE
SET
  wh_total = EXCLUDED.wh_total,
  wh_community = EXCLUDED.wh_community,
  wh_self = EXCLUDED.wh_self,
  source = EXCLUDED.source,
  quality = EXCLUDED.quality;

COMMIT;
