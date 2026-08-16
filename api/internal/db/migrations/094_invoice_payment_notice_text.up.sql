-- Free-text override for the Zahlungshinweis paragraph, used when
-- invoice_payment_notice_mode = 'custom' (4th mode alongside the existing
-- sepa_lastschrift/ueberweisung/none). Supports placeholders {betrag}, {iban},
-- {eeg_iban}, {eeg_bic}, {datum} — same {placeholder} convention already used
-- by onboarding_contract_text (migration 031).
ALTER TABLE eegs ADD COLUMN invoice_payment_notice_text text NOT NULL DEFAULT '';
