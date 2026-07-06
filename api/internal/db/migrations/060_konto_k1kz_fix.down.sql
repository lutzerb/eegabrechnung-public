-- Revert to (incorrect) prior assignments
UPDATE ea_konten SET k1_kz = '9140' WHERE nummer = '6100';
UPDATE ea_konten SET k1_kz = '9170' WHERE nummer = '6200';
UPDATE ea_konten SET k1_kz = '9160' WHERE nummer IN ('6300', '6400', '6500');
UPDATE ea_konten SET k1_kz = '9190' WHERE nummer = '6900';
