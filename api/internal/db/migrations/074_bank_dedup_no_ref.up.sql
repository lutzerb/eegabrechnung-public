-- Secondary dedup index for entries without a bank reference (e.g. Raiffeisen NOTPROVIDED charges).
-- Uses date + amount + description as the unique key when referenz is empty.
CREATE UNIQUE INDEX ea_banktransaktionen_dedup_noref_idx
    ON ea_banktransaktionen (eeg_id, buchungsdatum, betrag, verwendungszweck)
    WHERE (referenz IS NULL OR referenz = '') AND verwendungszweck IS NOT NULL AND verwendungszweck <> '';
