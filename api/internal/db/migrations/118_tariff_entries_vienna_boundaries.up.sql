-- tariff_entries.valid_from/valid_until for annual/monthly schedules were built by
-- the frontend as literal UTC calendar boundaries (e.g. "2026-08-01T00:00:00Z")
-- instead of Vienna-local month/year boundaries. Billing periods are computed
-- Vienna-local (viennaLoc in billing.go), so this created a 1-2 hour (DST) window
-- at the start of every month/year where readings were priced with the PREVIOUS
-- tariff entry instead of the current one (SumTOUForMember matches readings by
-- literal timestamp against valid_from/valid_until).
--
-- Every annual/monthly entry was confirmed to be at literal UTC midnight before
-- writing this migration, so reinterpreting the stored calendar date as Vienna-
-- local wall-clock time (instead of UTC) is safe and exact. daily/quarter_hour
-- schedules are import-based with real per-interval timestamps and are excluded.
UPDATE tariff_entries te
SET
  valid_from  = (date_trunc('day', te.valid_from  AT TIME ZONE 'UTC') AT TIME ZONE 'Europe/Vienna'),
  valid_until = (date_trunc('day', te.valid_until AT TIME ZONE 'UTC') AT TIME ZONE 'Europe/Vienna')
FROM tariff_schedules ts
WHERE ts.id = te.schedule_id
  AND ts.granularity IN ('annual', 'monthly');
