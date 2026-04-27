ALTER TABLE members DROP COLUMN IF EXISTS sepa_mandate_signed_at;
ALTER TABLE members DROP COLUMN IF EXISTS sepa_mandate_signed_ip;
ALTER TABLE members DROP COLUMN IF EXISTS sepa_mandate_text;
ALTER TABLE eegs DROP COLUMN IF EXISTS sepa_pre_notification_days;
