-- Reset the tariff-backfilled values back to 0 for prosumer invoices
-- (those where flat EEG prices are 0 but tariff schedules exist).
-- Uses the same join logic as the up migration to identify affected rows.
WITH weighted_prices AS (
  SELECT i.id AS invoice_id
  FROM invoices i
  JOIN tariff_schedules ts ON ts.eeg_id = i.eeg_id
  JOIN tariff_entries te ON te.schedule_id = ts.id
    AND te.valid_from  < i.period_end   + INTERVAL '1 second'
    AND te.valid_until > i.period_start
  WHERE i.consumption_kwh > 0
    AND i.generation_kwh  > 0
  GROUP BY i.id
)
UPDATE invoices i
SET consumption_net_amount = 0,
    generation_net_amount  = 0
FROM weighted_prices wp
WHERE i.id = wp.invoice_id;
