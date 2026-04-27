-- Configurable contract text shown in the public onboarding form.
-- Supports placeholders: {iban} (member's entered IBAN), {datum} (today's date).
-- If empty, the frontend falls back to a built-in default.
ALTER TABLE eegs ADD COLUMN IF NOT EXISTS onboarding_contract_text TEXT NOT NULL DEFAULT '';
