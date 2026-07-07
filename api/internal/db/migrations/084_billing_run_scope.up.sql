-- Abrechnungslauf-Scope: Billing-Type (all | consumption_only | production_only)
-- und optionale Mitglieds-Einschränkung werden am Run persistiert, damit die
-- Overlap-Prüfung nur echte Konflikte meldet. Bisher sperrte z. B. ein
-- consumption_only-Lauf den Zeitraum auch für den komplementären
-- production_only-Lauf, und ein Einzelmitglieds-Lauf für alle anderen
-- Mitglieder — der Ausweg war force, das ALLE Doppelabrechnungsprüfungen
-- abschaltet. member_ids IS NULL bedeutet: alle Mitglieder.
ALTER TABLE billing_runs ADD COLUMN billing_type TEXT NOT NULL DEFAULT 'all';
ALTER TABLE billing_runs ADD COLUMN member_ids UUID[];
