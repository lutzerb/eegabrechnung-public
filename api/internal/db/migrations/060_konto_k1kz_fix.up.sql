-- Correct K1 KZ assignments based on official BMF K1 form (2025):
--   9160 = Reise-/Fahrtspesen (not Sonstige BA)
--   9170 = tatsächliche Kfz-Kosten (not Bankspesen)
--   9180 = Miet- und Pachtaufwand, Leasing
--   9220 = Zinsen und ähnliche Aufwendungen  ← correct for Bankspesen
--   9230 = Übrige Aufwendungen, Saldo        ← correct catch-all

-- Netzkosten: no dedicated energy/grid-cost KZ in K1 → Übrige Aufwendungen
UPDATE ea_konten SET k1_kz = '9230' WHERE nummer = '6100';

-- Bankspesen → Zinsen und ähnliche Aufwendungen (KZ9220, not 9170)
UPDATE ea_konten SET k1_kz = '9220' WHERE nummer = '6200';

-- IT/Hosting, Vereinskosten, Steuerberatung → Übrige Aufwendungen (not 9160)
UPDATE ea_konten SET k1_kz = '9230' WHERE nummer IN ('6300', '6400', '6500');

-- Sonstige Ausgaben → Übrige Aufwendungen (was 9190 = Provisionen/Lizenzgebühren)
UPDATE ea_konten SET k1_kz = '9230' WHERE nummer = '6900';
