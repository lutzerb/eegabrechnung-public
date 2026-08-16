-- Customizable column headers for the "individuell" invoice design's generation
-- breakdown table (see drawGenerationPeriodTable) — the Einspeisung counterpart
-- of the Netzbezug/Community-Verbrauch table added in migration 097. Defaults
-- match the hardcoded strings so existing invoices render unchanged until an
-- admin customizes them.
ALTER TABLE eegs ADD COLUMN invoice_energy_label_gesamteinspeisung text NOT NULL DEFAULT 'Gesamteinspeisung kWh';
ALTER TABLE eegs ADD COLUMN invoice_energy_label_abnahme_energiegemeinschaft text NOT NULL DEFAULT 'Abnahme durch Energiegemeinschaft kWh';
ALTER TABLE eegs ADD COLUMN invoice_energy_label_resteinspeisung text NOT NULL DEFAULT 'Resteinspeisung kWh';
