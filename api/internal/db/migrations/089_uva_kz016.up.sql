-- Kleinunternehmer-Umsätze (§ 6 Abs. 1 Z 27 UStG) sind in der UVA/U1 in Kennzahl 016
-- zu erfassen (zusätzlich zu KZ 000), auch wenn die UVA nur wegen einer
-- Reverse-Charge-Steuerschuld (KZ 032) abgegeben wird.
ALTER TABLE ea_uva_perioden ADD COLUMN kz_016 DECIMAL(12,2) NOT NULL DEFAULT 0;
