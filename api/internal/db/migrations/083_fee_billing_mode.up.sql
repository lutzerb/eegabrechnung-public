-- Fixgebühren-Abrechnungsmodus: bestimmt, ob meter_fee_eur, participation_fee_eur
-- und zaehlpunkts_gebuehr_eur pro angefangenem Kalendermonat des Abrechnungszeitraums
-- ('per_month', Standard) oder einmal pro Abrechnungslauf ('per_invoice') berechnet werden.
-- Bisheriges Verhalten war implizit 'per_invoice'; bei Monatsabrechnung sind beide identisch.
ALTER TABLE eegs ADD COLUMN fee_billing_mode TEXT NOT NULL DEFAULT 'per_month'
    CHECK (fee_billing_mode IN ('per_month', 'per_invoice'));
