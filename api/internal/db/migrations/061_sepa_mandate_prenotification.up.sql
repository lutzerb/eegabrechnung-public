-- SEPA mandate data captured during onboarding (stored on members for PDF generation)
ALTER TABLE members
  ADD COLUMN sepa_mandate_signed_at TIMESTAMPTZ,
  ADD COLUMN sepa_mandate_signed_ip TEXT,
  ADD COLUMN sepa_mandate_text      TEXT;

-- Configurable SEPA pre-notification period (SEPA Rulebook minimum: 14 days)
-- Invoice date serves as pre-notification; collection = invoice_date + sepa_pre_notification_days
ALTER TABLE eegs
  ADD COLUMN sepa_pre_notification_days INT NOT NULL DEFAULT 14;
