-- uid_nummer is the member's VAT ID (Umsatzsteuer-Identifikationsnummer).
-- Required for reverse-charge determination on producer credit notes.
ALTER TABLE members ADD COLUMN uid_nummer text NOT NULL DEFAULT '';
