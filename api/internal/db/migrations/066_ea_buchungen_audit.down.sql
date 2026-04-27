DROP TABLE IF EXISTS ea_buchungen_changelog;

ALTER TABLE ea_buchungen
  DROP COLUMN IF EXISTS deleted_at,
  DROP COLUMN IF EXISTS deleted_by;
