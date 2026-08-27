-- Per-segment colors for the "individuell" design's Verbrauchsentwicklungsgrafik
-- (see InvoiceTheme.ChartColor* / drawBarChartThemed / drawPercentBarChartThemed).
-- community_bezug/community_einspeisung default to the same green (the two
-- community-covered segments look identical out of the box, matching current
-- behavior) but can be set independently; netzbezug/resteinspeisung only
-- appear on the percentage chart variant.
ALTER TABLE eegs ADD COLUMN invoice_chart_color_community_bezug text NOT NULL DEFAULT '#22c55e';
ALTER TABLE eegs ADD COLUMN invoice_chart_color_netzbezug text NOT NULL DEFAULT '#f59e0b';
ALTER TABLE eegs ADD COLUMN invoice_chart_color_community_einspeisung text NOT NULL DEFAULT '#22c55e';
ALTER TABLE eegs ADD COLUMN invoice_chart_color_resteinspeisung text NOT NULL DEFAULT '#3b82f6';
