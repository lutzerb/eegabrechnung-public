-- Reset VAT-recovered split amounts back to 0.
-- Only resets prosumer invoices (both kwh > 0) that have vat_pct > 0 (non-KU EEGs).
UPDATE invoices
SET consumption_net_amount = 0,
    generation_net_amount  = 0
WHERE consumption_kwh > 0
  AND generation_kwh  > 0
  AND consumption_vat_pct > 0
  AND consumption_vat_amount > 0
  AND consumption_net_amount > 0
  AND generation_net_amount  > 0;
