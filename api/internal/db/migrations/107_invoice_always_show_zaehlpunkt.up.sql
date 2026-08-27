-- Forces the "individuell" invoice design's energy/generation tables to
-- print the "Zählpunkt: xxx" sub-heading band even when the invoice covers
-- only a single Zählpunkt. Default false preserves current behavior (the
-- heading is only shown when an invoice spans more than one Zählpunkt).
ALTER TABLE eegs ADD COLUMN invoice_always_show_zaehlpunkt boolean NOT NULL DEFAULT false;
