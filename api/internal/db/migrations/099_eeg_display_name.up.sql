-- Optional alias/display name for an EEG, shown everywhere except legally
-- binding documents (invoice Rechnungssteller block, SEPA XML/mandate,
-- onboarding declaration), which keep using the legal `name`. Empty string
-- means "no alias set" - callers fall back to `name`.
ALTER TABLE eegs ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
