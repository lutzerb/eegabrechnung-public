ALTER TABLE eegs
  ADD COLUMN referral_bonus_eur NUMERIC(10,2) NOT NULL DEFAULT 5.00;

ALTER TABLE members
  ADD COLUMN referral_code VARCHAR(20) UNIQUE,
  ADD COLUMN referred_by_member_id UUID REFERENCES members(id);

ALTER TABLE onboarding_requests
  ADD COLUMN referred_by_member_id UUID REFERENCES members(id);

CREATE TABLE member_referral_bonuses (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  eeg_id            UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
  referrer_member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  referred_member_id UUID NOT NULL UNIQUE REFERENCES members(id) ON DELETE CASCADE,
  amount_eur        NUMERIC(10,2) NOT NULL,
  status            VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | applied | cancelled
  granted_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  granted_by        UUID REFERENCES users(id),
  applied_invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
  applied_at        TIMESTAMPTZ
);

CREATE INDEX idx_member_referral_bonuses_referrer_pending
  ON member_referral_bonuses (eeg_id, referrer_member_id)
  WHERE status = 'pending';
