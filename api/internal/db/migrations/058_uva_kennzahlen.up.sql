-- Add missing UVA Kennzahlen columns:
-- kz_044: Steuer für 10% Umsätze (KZ029 × 10%) — VERSTEUERT section
-- kz_057: Steuerschuld gem. §19 Abs. 1 UStG (Reverse Charge domestic) — VERSTEUERT section
ALTER TABLE ea_uva_perioden
    ADD COLUMN IF NOT EXISTS kz_044 NUMERIC(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS kz_057 NUMERIC(12,2) NOT NULL DEFAULT 0;
