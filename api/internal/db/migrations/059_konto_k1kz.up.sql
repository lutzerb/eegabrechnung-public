-- Add K1 Kennzahl code to accounts (maps each account to a FinanzOnline K1 field).
ALTER TABLE ea_konten
    ADD COLUMN IF NOT EXISTS k1_kz VARCHAR(10) NOT NULL DEFAULT '';

-- Pre-populate standard EEG-Verein accounts (matched by Kontonummer).
-- Einnahmen → GEWINN_VERLUSTRECHNUNG KZ9040 (Betriebseinnahmen)
UPDATE ea_konten SET k1_kz = '9040' WHERE nummer IN ('4000', '4010', '4900');
-- Ausgaben Einspeisevergütungen → KZ9100 (Wareneinsatz / Materialeinsatz)
UPDATE ea_konten SET k1_kz = '9100' WHERE nummer IN ('6000', '6010', '6015');
-- Netzkosten → KZ9140 (Energie- und Wasseraufwand)
UPDATE ea_konten SET k1_kz = '9140' WHERE nummer = '6100';
-- Bankspesen → KZ9170 (Zinsen und ähnliche Aufwendungen)
UPDATE ea_konten SET k1_kz = '9170' WHERE nummer = '6200';
-- IT / Steuerberatung / Vereinskosten → KZ9160 (Sonstige Betriebsausgaben)
UPDATE ea_konten SET k1_kz = '9160' WHERE nummer IN ('6300', '6400', '6500');
-- Sonstige Ausgaben → KZ9190 (Übrige Aufwendungen)
UPDATE ea_konten SET k1_kz = '9190' WHERE nummer = '6900';
