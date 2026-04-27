ALTER TABLE members
  DROP COLUMN IF EXISTS beitritt_datum,
  DROP COLUMN IF EXISTS austritt_datum;
