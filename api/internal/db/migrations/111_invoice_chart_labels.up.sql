-- Legend/axis label overrides for the "individuell" design's Verbrauchsentwicklungsgrafik
-- (see InvoiceTheme.ChartLabel* / drawBarChartThemed / drawPercentBarChartThemed in
-- pdf_theme.go). Empty (default) falls back to the current hardcoded German text —
-- same orDefault() convention as invoice_chart_title. Previously these legend strings
-- were hardcoded even though the visually adjacent chart title (invoice_chart_title,
-- migration 109) and the energy breakdown table headers (invoice_energy_label_*,
-- migration 099-ish) were already customizable, so an EEG that renamed "Community"
-- there (e.g. to "Energieverein") ended up with mismatched terminology in the chart
-- legend.
ALTER TABLE eegs ADD COLUMN invoice_chart_label_community text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_chart_label_community_bezug text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_chart_label_community_einspeisung text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_chart_label_netzbezug text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_chart_label_resteinspeisung text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_chart_label_bezug text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_chart_label_einspeisung text NOT NULL DEFAULT '';
