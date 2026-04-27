-- Re-backfill consumption_net_amount / generation_net_amount for prosumer invoices
-- whose EEG uses tariff schedules (flat eeg.energy_price = 0, causing migration 063
-- to produce zeros). Computes a time-weighted average tariff price over the invoice
-- billing period and applies it to the kWh quantities.
--
-- Only updates rows that are still 0 (i.e. migration 063 left them unset) and that
-- have both consumption and generation kWh > 0.
WITH weighted_prices AS (
  SELECT
    i.id   AS invoice_id,
    i.eeg_id,
    ROUND(
      COALESCE(
        SUM(
          GREATEST(0.0, EXTRACT(EPOCH FROM (
            LEAST(te.valid_until, i.period_end + INTERVAL '1 second') -
            GREATEST(te.valid_from, i.period_start)
          ))) * te.energy_price
        ) /
        NULLIF(EXTRACT(EPOCH FROM (i.period_end + INTERVAL '1 second' - i.period_start)), 0),
        0
      )::NUMERIC, 6
    ) AS weighted_energy_price,
    ROUND(
      COALESCE(
        SUM(
          GREATEST(0.0, EXTRACT(EPOCH FROM (
            LEAST(te.valid_until, i.period_end + INTERVAL '1 second') -
            GREATEST(te.valid_from, i.period_start)
          ))) * te.producer_price
        ) /
        NULLIF(EXTRACT(EPOCH FROM (i.period_end + INTERVAL '1 second' - i.period_start)), 0),
        0
      )::NUMERIC, 6
    ) AS weighted_producer_price
  FROM invoices i
  JOIN tariff_schedules ts ON ts.eeg_id = i.eeg_id
  JOIN tariff_entries te ON te.schedule_id = ts.id
    AND te.valid_from  < i.period_end   + INTERVAL '1 second'
    AND te.valid_until > i.period_start
  WHERE i.consumption_net_amount = 0
    AND i.generation_net_amount  = 0
    AND i.consumption_kwh > 0
    AND i.generation_kwh  > 0
  GROUP BY i.id, i.eeg_id
),
invoice_data AS (
  SELECT
    wp.invoice_id,
    ROUND((inv.consumption_kwh * wp.weighted_energy_price / 100.0 + e.meter_fee_eur + e.participation_fee_eur)::NUMERIC, 4) AS new_consumption_net,
    ROUND((inv.generation_kwh  * wp.weighted_producer_price / 100.0)::NUMERIC, 4) AS new_generation_net
  FROM weighted_prices wp
  JOIN eegs e    ON e.id    = wp.eeg_id
  JOIN invoices inv ON inv.id = wp.invoice_id
  WHERE wp.weighted_energy_price > 0
)
UPDATE invoices
SET
  consumption_net_amount = id.new_consumption_net,
  generation_net_amount  = id.new_generation_net
FROM invoice_data id
WHERE invoices.id = id.invoice_id;
