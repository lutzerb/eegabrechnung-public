ALTER TABLE eda_processes
    DROP COLUMN IF EXISTS ec_dis_model,
    DROP COLUMN IF EXISTS date_to,
    DROP COLUMN IF EXISTS energy_direction,
    DROP COLUMN IF EXISTS ec_share;
