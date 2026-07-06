-- Add separate net amount columns for consumption (Bezug) and generation (Einspeisung)
-- so that the E/A import can create two bookings with correct accounts.
ALTER TABLE invoices
  ADD COLUMN consumption_net_amount DECIMAL(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN generation_net_amount  DECIMAL(12,4) NOT NULL DEFAULT 0;

-- Backfill: pure consumers (no generation)
UPDATE invoices
SET consumption_net_amount = net_amount,
    generation_net_amount  = 0
WHERE generation_kwh = 0;

-- Backfill: pure producers (no consumption) — net_amount is negative (credit to member)
UPDATE invoices
SET consumption_net_amount = 0,
    generation_net_amount  = ABS(net_amount)
WHERE consumption_kwh = 0 AND generation_kwh > 0;

-- Backfill: prosumers (both consumption and generation) — approximate from flat EEG prices.
-- consumptionNet = consumption_kwh * energy_price/100 + meter_fee + participation_fee
-- generationNet  = generation_kwh * producer_price/100
-- These are approximations since the exact tariff prices at billing time are not stored.
UPDATE invoices i
SET consumption_net_amount = ROUND((i.consumption_kwh * e.energy_price / 100.0 + e.meter_fee_eur + e.participation_fee_eur)::NUMERIC, 4),
    generation_net_amount  = ROUND((i.generation_kwh  * e.producer_price / 100.0)::NUMERIC, 4)
FROM eegs e
WHERE i.eeg_id = e.id
  AND i.consumption_kwh > 0
  AND i.generation_kwh > 0;
