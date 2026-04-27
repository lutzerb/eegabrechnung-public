ALTER TABLE members
  ADD COLUMN IF NOT EXISTS beitritt_datum date,
  ADD COLUMN IF NOT EXISTS austritt_datum date;
