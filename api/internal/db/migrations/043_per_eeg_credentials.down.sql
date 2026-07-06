ALTER TABLE eegs
  DROP COLUMN IF EXISTS eda_imap_host,
  DROP COLUMN IF EXISTS eda_imap_user,
  DROP COLUMN IF EXISTS eda_imap_password_enc,
  DROP COLUMN IF EXISTS eda_smtp_host,
  DROP COLUMN IF EXISTS eda_smtp_user,
  DROP COLUMN IF EXISTS eda_smtp_password_enc,
  DROP COLUMN IF EXISTS eda_smtp_from,
  DROP COLUMN IF EXISTS smtp_host,
  DROP COLUMN IF EXISTS smtp_user,
  DROP COLUMN IF EXISTS smtp_password_enc,
  DROP COLUMN IF EXISTS smtp_from;
