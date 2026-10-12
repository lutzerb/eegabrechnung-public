-- Freier, platzhalterbasierter Betreff/Body für Rechnungs-/Gutschrift-E-Mails.
-- Leerer String (Default) = Fallback auf den bisherigen hartcodierten Text in invoice/email.go.
ALTER TABLE eegs ADD COLUMN invoice_email_subject text NOT NULL DEFAULT '';
ALTER TABLE eegs ADD COLUMN invoice_email_body text NOT NULL DEFAULT '';
