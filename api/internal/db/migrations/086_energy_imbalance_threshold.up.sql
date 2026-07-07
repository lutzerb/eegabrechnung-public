-- Migration 086: configurable tolerance for the community-wide generation/consumption
-- balance warning shown on billing runs. wh_self (CONSUMPTION) and wh_community
-- (GENERATION) represent the same physically shared energy pool reported twice by
-- the Netzbetreiber; a relative difference beyond this threshold usually points to
-- missing readings or an allocation problem. Unit: promille (‰) of the larger total.

ALTER TABLE eegs
  ADD COLUMN energy_imbalance_threshold_promille numeric NOT NULL DEFAULT 1;
