DROP TABLE IF EXISTS member_referral_bonuses;

ALTER TABLE onboarding_requests
  DROP COLUMN IF EXISTS referred_by_member_id;

ALTER TABLE members
  DROP COLUMN IF EXISTS referred_by_member_id,
  DROP COLUMN IF EXISTS referral_code;

ALTER TABLE eegs
  DROP COLUMN IF EXISTS referral_bonus_eur;
