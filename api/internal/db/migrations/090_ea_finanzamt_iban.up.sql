-- IBAN der zuständigen Finanzamt-Dienststelle, für die SEPA-Finanzamtszahlung (FAZ)
-- der UVA-Zahllast (Purp/Cd=TAXS, siehe PSA "Finanzamtszahlung in EBICS" v1.0.01).
ALTER TABLE eegs ADD COLUMN IF NOT EXISTS ea_finanzamt_iban VARCHAR(34);
