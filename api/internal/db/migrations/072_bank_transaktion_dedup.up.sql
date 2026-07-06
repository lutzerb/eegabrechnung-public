-- Partial unique index: prevents duplicate imports when re-importing overlapping periods.
-- Only applies when referenz is non-empty (AcctSvcrRef in CAMT.053, reference in MT940).
CREATE UNIQUE INDEX ea_banktransaktionen_dedup_idx
    ON ea_banktransaktionen (eeg_id, buchungsdatum, betrag, referenz)
    WHERE referenz IS NOT NULL AND referenz <> '';
