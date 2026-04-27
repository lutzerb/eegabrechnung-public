-- Billing run: groups all invoices created in one billing operation.
CREATE TABLE billing_runs (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    eeg_id       UUID        NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'completed',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_runs_eeg_id ON billing_runs(eeg_id);

-- Link existing and future invoices to a billing run.
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS billing_run_id UUID REFERENCES billing_runs(id) ON DELETE SET NULL;

CREATE INDEX idx_invoices_billing_run_id ON invoices(billing_run_id);
