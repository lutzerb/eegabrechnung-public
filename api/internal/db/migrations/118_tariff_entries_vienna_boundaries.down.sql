-- Reverses 118.up: reinterpret the Vienna-local wall-clock boundary as if it
-- were UTC, restoring the original (buggy) literal-UTC-midnight values.
UPDATE tariff_entries te
SET
  valid_from  = ((te.valid_from  AT TIME ZONE 'Europe/Vienna') AT TIME ZONE 'UTC'),
  valid_until = ((te.valid_until AT TIME ZONE 'Europe/Vienna') AT TIME ZONE 'UTC')
FROM tariff_schedules ts
WHERE ts.id = te.schedule_id
  AND ts.granularity IN ('annual', 'monthly');
