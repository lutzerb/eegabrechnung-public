-- Neue Felder für ECMPList Prozesse (EC_PRTFACT_CHG, EC_EINZEL_ABM)
ALTER TABLE eda_processes
    ADD COLUMN IF NOT EXISTS ec_dis_model     text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS date_to          date,
    ADD COLUMN IF NOT EXISTS energy_direction text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ec_share         numeric;
