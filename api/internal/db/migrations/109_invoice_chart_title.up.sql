-- Optional custom title for the "individuell" design's Verbrauchsentwicklungsgrafik
-- (see InvoiceTheme.ChartTitle / drawBarChartThemed / drawPercentBarChartThemed).
-- Empty (default) falls back to the chart type's built-in default title, which
-- differs between invoice_chart_type='absolute' and 'percentage'.
ALTER TABLE eegs ADD COLUMN invoice_chart_title text NOT NULL DEFAULT '';
