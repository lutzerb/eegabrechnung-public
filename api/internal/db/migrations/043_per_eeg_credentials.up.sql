-- Per-EEG credentials: EDA IMAP/SMTP and invoice SMTP.
-- Passwords are stored AES-GCM encrypted (base64) by the application layer.

ALTER TABLE eegs
  ADD COLUMN eda_imap_host       TEXT,
  ADD COLUMN eda_imap_user       TEXT,
  ADD COLUMN eda_imap_password_enc TEXT,

  ADD COLUMN eda_smtp_host       TEXT,
  ADD COLUMN eda_smtp_user       TEXT,
  ADD COLUMN eda_smtp_password_enc TEXT,
  ADD COLUMN eda_smtp_from       TEXT,

  ADD COLUMN smtp_host           TEXT,
  ADD COLUMN smtp_user           TEXT,
  ADD COLUMN smtp_password_enc   TEXT,
  ADD COLUMN smtp_from           TEXT;
