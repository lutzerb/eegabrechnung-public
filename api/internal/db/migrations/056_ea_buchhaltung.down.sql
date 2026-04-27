DROP TABLE IF EXISTS ea_uva_perioden;
DROP TABLE IF EXISTS ea_belege;
DROP TABLE IF EXISTS ea_buchungen;
DROP TABLE IF EXISTS ea_konten;
ALTER TABLE eegs
  DROP COLUMN IF EXISTS ea_uva_periodentyp,
  DROP COLUMN IF EXISTS ea_steuernummer,
  DROP COLUMN IF EXISTS ea_finanzamt;
