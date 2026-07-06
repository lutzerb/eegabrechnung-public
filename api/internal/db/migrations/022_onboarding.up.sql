CREATE TABLE IF NOT EXISTS onboarding_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id uuid NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending',
    name1 text NOT NULL DEFAULT '',
    name2 text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    strasse text NOT NULL DEFAULT '',
    plz text NOT NULL DEFAULT '',
    ort text NOT NULL DEFAULT '',
    iban text NOT NULL DEFAULT '',
    bic text NOT NULL DEFAULT '',
    member_type text NOT NULL DEFAULT 'CONSUMER',
    meter_points jsonb NOT NULL DEFAULT '[]'::jsonb,
    contract_accepted_at timestamptz,
    contract_ip text NOT NULL DEFAULT '',
    magic_token text NOT NULL DEFAULT '',
    magic_token_expires_at timestamptz NOT NULL DEFAULT (now() + interval '30 days'),
    admin_notes text NOT NULL DEFAULT '',
    converted_member_id uuid REFERENCES members(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS onboarding_requests_eeg_id_idx ON onboarding_requests(eeg_id);
CREATE UNIQUE INDEX IF NOT EXISTS onboarding_requests_magic_token_idx ON onboarding_requests(magic_token) WHERE magic_token != '';
