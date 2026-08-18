-- Whether the "individuell" invoice design's measurement table (top) and
-- pricing table (bottom) list every calendar month individually on
-- multi-month invoices, or only the period total per Zählpunkt/direction.
-- Default true: preserves current behavior (always monthly) for every EEG
-- already using the individuell design. Rendering logic (pdf_theme.go)
-- still forces monthly rows regardless of this flag whenever the tariff
-- price actually varied across months in the period, so the price column
-- never silently blends different rates into one misleading total.
ALTER TABLE eegs ADD COLUMN invoice_show_monthly_breakdown boolean NOT NULL DEFAULT true;
