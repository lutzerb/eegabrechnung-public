-- Customizable column headers for the "individuell" invoice design's energy
-- breakdown table (see drawEnergyPeriodTable). Defaults match the current
-- hardcoded strings so existing invoices render unchanged until an admin
-- customizes them.
ALTER TABLE eegs ADD COLUMN invoice_energy_label_zeitraum_von text NOT NULL DEFAULT 'Zeitraum von';
ALTER TABLE eegs ADD COLUMN invoice_energy_label_zeitraum_bis text NOT NULL DEFAULT 'Zeitraum bis';
ALTER TABLE eegs ADD COLUMN invoice_energy_label_gesamtverbrauch text NOT NULL DEFAULT 'Gesamtverbrauch kWh';
ALTER TABLE eegs ADD COLUMN invoice_energy_label_netzbezug text NOT NULL DEFAULT 'Netzbezug kWh';
ALTER TABLE eegs ADD COLUMN invoice_energy_label_community_verbrauch text NOT NULL DEFAULT 'Community-Verbrauch kWh';

-- Whether the Fixgebühr/Teilnahmegebühr and Zählpunktsgebühr line items are
-- shown even when their total is 0,00 € (default: hidden, current behavior).
-- Applies to both the standard and individuell invoice designs.
ALTER TABLE eegs ADD COLUMN invoice_show_zero_fees boolean NOT NULL DEFAULT false;
