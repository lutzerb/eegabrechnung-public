-- Per-EEG configurable colors for the web reports page charts
-- (/eegs/{eegID}/reports, web/components/energy-charts.tsx). Independent of the
-- PDF invoice chart colors (migration 110). Each column defaults to the chart's
-- original hardcoded hex so existing EEGs keep their current appearance.
ALTER TABLE eegs ADD COLUMN report_chart_color_ausgetauscht text NOT NULL DEFAULT '#10b981';
ALTER TABLE eegs ADD COLUMN report_chart_color_restbedarf text NOT NULL DEFAULT '#fbbf24';
ALTER TABLE eegs ADD COLUMN report_chart_color_einspeisung_eeg text NOT NULL DEFAULT '#6366f1';
ALTER TABLE eegs ADD COLUMN report_chart_color_resteinspeisung text NOT NULL DEFAULT '#c4b5fd';
ALTER TABLE eegs ADD COLUMN report_chart_color_billed_einspeisung text NOT NULL DEFAULT '#10b981';
ALTER TABLE eegs ADD COLUMN report_chart_color_billed_bezug text NOT NULL DEFAULT '#3b82f6';
ALTER TABLE eegs ADD COLUMN report_chart_color_einnahmen text NOT NULL DEFAULT '#3b82f6';
ALTER TABLE eegs ADD COLUMN report_chart_color_gutschriften text NOT NULL DEFAULT '#f59e0b';
ALTER TABLE eegs ADD COLUMN report_chart_color_member_einspeisung text NOT NULL DEFAULT '#6366f1';
ALTER TABLE eegs ADD COLUMN report_chart_color_member_bezug text NOT NULL DEFAULT '#10b981';
