-- Which "Verbrauchsentwicklungsgrafik" the "individuell" invoice design shows:
-- 'absolute' (default, current bar chart with kWh values) or 'percentage' (new
-- 100%-stacked bar chart showing Community- vs. Netz-/Restanteil per month).
ALTER TABLE eegs ADD COLUMN invoice_chart_type text NOT NULL DEFAULT 'absolute';
