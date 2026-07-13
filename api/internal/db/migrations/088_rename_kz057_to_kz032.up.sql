-- Reverse-Charge-Steuerschuld (§ 19 Abs. 1d UStG i.V.m. § 2 Z 2 UStBBKV) gehört im
-- FinanzOnline U30 in Kennzahl 032, nicht 057. kz_057 war eine falsche Zuordnung.
ALTER TABLE ea_uva_perioden RENAME COLUMN kz_057 TO kz_032;
