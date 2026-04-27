-- BAO §131: Nachvollziehbarkeit von Buchungsänderungen
-- Soft-Delete auf ea_buchungen + Changelog-Tabelle für alle Mutationen

ALTER TABLE ea_buchungen
  ADD COLUMN deleted_at  TIMESTAMPTZ,
  ADD COLUMN deleted_by  TEXT;

CREATE TABLE ea_buchungen_changelog (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  buchung_id  UUID        NOT NULL REFERENCES ea_buchungen(id) ON DELETE CASCADE,
  operation   TEXT        NOT NULL CHECK (operation IN ('create', 'update', 'delete')),
  changed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  changed_by  TEXT        NOT NULL DEFAULT '',
  old_values  JSONB,
  new_values  JSONB,
  reason      TEXT
);

CREATE INDEX ea_buchungen_changelog_buchung_idx   ON ea_buchungen_changelog (buchung_id);
CREATE INDEX ea_buchungen_changelog_changed_at_idx ON ea_buchungen_changelog (changed_at DESC);
CREATE INDEX ea_buchungen_changelog_eeg_idx        ON ea_buchungen_changelog (buchung_id, changed_at DESC);
