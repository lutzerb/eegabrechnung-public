ALTER TABLE eegs
  ADD COLUMN producer_price        numeric NOT NULL DEFAULT 0,
  ADD COLUMN use_vat               boolean NOT NULL DEFAULT false,
  ADD COLUMN vat_pct               numeric NOT NULL DEFAULT 20,
  ADD COLUMN meter_fee_eur         numeric NOT NULL DEFAULT 0,
  ADD COLUMN free_kwh              numeric NOT NULL DEFAULT 0,
  ADD COLUMN discount_pct          numeric NOT NULL DEFAULT 0,
  ADD COLUMN participation_fee_eur numeric NOT NULL DEFAULT 0,
  ADD COLUMN billing_period        text    NOT NULL DEFAULT 'monthly';
