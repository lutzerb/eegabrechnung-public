-- Per-EEG configurable list of "how did you hear about us" options shown on
-- the public onboarding form. "Sonstiges" (free text) is always additionally
-- offered by the frontend and is not stored here.
ALTER TABLE eegs
    ADD COLUMN IF NOT EXISTS referral_options TEXT[] NOT NULL
    DEFAULT ARRAY['Empfehlung von einem Mitglied','Internet','Zeitung'];
