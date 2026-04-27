-- Exact recovery of consumption_net_amount / generation_net_amount for prosumer invoices
-- where the EEG is VAT-registered (consumption_vat_pct > 0, consumption_vat_amount > 0).
--
-- Formula (exact, no approximation):
--   consumptionNet = consumption_vat_amount * 100 / consumption_vat_pct
--   generationNet  = consumptionNet − net_amount
--
-- Only updates rows still at 0 (not yet set by migrations 063/064).
UPDATE invoices
SET
  consumption_net_amount = ROUND(
    (consumption_vat_amount * 100.0 / consumption_vat_pct)::NUMERIC, 4
  ),
  generation_net_amount = ROUND(
    (consumption_vat_amount * 100.0 / consumption_vat_pct - net_amount)::NUMERIC, 4
  )
WHERE consumption_net_amount = 0
  AND generation_net_amount  = 0
  AND consumption_kwh > 0
  AND generation_kwh  > 0
  AND consumption_vat_pct > 0
  AND consumption_vat_amount > 0;
