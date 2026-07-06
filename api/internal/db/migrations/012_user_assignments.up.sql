CREATE TABLE user_eeg_assignments (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    eeg_id  UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, eeg_id)
);

-- Normalize legacy 'member' role to 'user'
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'user';
UPDATE users SET role = 'user' WHERE role = 'member';
