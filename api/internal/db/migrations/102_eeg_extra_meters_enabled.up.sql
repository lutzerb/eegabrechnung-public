-- Feature toggle for Zusatzzähler (manually-read submeters, migration 101). Default
-- off: most EEGs don't need this, and the UI (member detail card, extra-meters page)
-- is only shown when an admin explicitly enables it in EEG-Einstellungen.
ALTER TABLE eegs ADD COLUMN extra_meters_enabled boolean NOT NULL DEFAULT false;
