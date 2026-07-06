-- Diagnose Zählpunkt-recycling data corruption (Bug B class).
--
-- Read-only. No schema change, no migration. Run manually against a Postgres connection,
-- e.g.:
--   docker exec -i eegabrechnung-eegabrechnung-postgres-1 \
--     psql -U eegabrechnung -d eegabrechnung -f api/scripts/diagnose_meterpoint_recycling.sql
--
-- Background: before the ingestion-time-awareness fix (GetByZaehlpunktAtTime), any energy
-- reading whose own timestamp fell before/after its meter_point_id row's own
-- registriert_seit/abgemeldet_am window was a strong signal that the reading had been
-- misattributed during a Zählpunkt handover (Mieterwechsel) — e.g. a late/corrected EDA
-- message or an XLSX import spanning the handover date, resolved via "whichever row is
-- currently active" instead of "whichever row was active at this reading's own timestamp".
--
-- This script finds exactly those readings. It does NOT fix anything — for each hit,
-- cross-check against the sibling meter_points row(s) sharing (eeg_id, zaehlpunkt) to find
-- which one's window actually covers the offending timestamps, then decide with the EEG
-- operator whether a manual UPDATE energy_readings SET meter_point_id = ... is warranted
-- (safe against the UNIQUE(meter_point_id, ts) constraint as long as the sibling's window
-- doesn't already have data at those exact timestamps, which by construction of the
-- partial-unique index on meter_points it should not).
--
-- Known limitation: this does NOT detect Bug A (Stammdaten Upsert silently overwriting
-- member_id on the same row before the fix). meter_points has no updated_at column, and
-- affected rows' meter_point_registration_periods entries were not correctly rotated
-- either (they kept the OLD tenant's registriert_seit) — there is no reliable signal left
-- to reconstruct which member a given historical reading originally belonged to. That
-- requires manual reconciliation against the EEG operator's own records.

-- 1) Readings whose ts falls outside their own meter_point_id row's registered window.
SELECT
    er.meter_point_id,
    mp.zaehlpunkt,
    mp.eeg_id,
    mp.member_id,
    mp.registriert_seit,
    mp.abgemeldet_am,
    MIN(er.ts)  AS earliest_offending_ts,
    MAX(er.ts)  AS latest_offending_ts,
    COUNT(*)    AS offending_reading_count
FROM energy_readings er
JOIN meter_points mp ON mp.id = er.meter_point_id
WHERE mp.registriert_seit IS NOT NULL
  AND (
    -- registriert_seit/abgemeldet_am are Vienna-local calendar dates — must convert via
    -- timezone('Europe/Vienna', ...), not a naive UTC cast, or every reading in the first
    -- ~1-2 hours of a Vienna day (previous UTC calendar day) is a false positive here.
    er.ts < timezone('Europe/Vienna', mp.registriert_seit::timestamp)
    OR (mp.abgemeldet_am IS NOT NULL AND er.ts >= timezone('Europe/Vienna', mp.abgemeldet_am::timestamp))
  )
GROUP BY er.meter_point_id, mp.zaehlpunkt, mp.eeg_id, mp.member_id, mp.registriert_seit, mp.abgemeldet_am
ORDER BY offending_reading_count DESC;

-- 2) For a specific (eeg_id, zaehlpunkt) flagged above, list all sibling rows (old + new)
--    to find which one's window actually covers the offending timestamps:
--
-- SELECT id, member_id, registriert_seit, abgemeldet_am, created_at
-- FROM meter_points
-- WHERE eeg_id = :eeg_id AND zaehlpunkt = :zaehlpunkt
-- ORDER BY registriert_seit;
--
-- Once the correct sibling meter_point_id is identified, reattach just the offending
-- window (do not touch readings outside [earliest_offending_ts, latest_offending_ts]):
--
-- UPDATE energy_readings
-- SET meter_point_id = :correct_meter_point_id
-- WHERE meter_point_id = :wrong_meter_point_id
--   AND ts >= :earliest_offending_ts
--   AND ts <  :latest_offending_ts + interval '15 minutes';
