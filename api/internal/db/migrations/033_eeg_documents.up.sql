CREATE TABLE eeg_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  eeg_id UUID NOT NULL REFERENCES eegs(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  filename TEXT NOT NULL,
  file_path TEXT NOT NULL,
  mime_type TEXT NOT NULL DEFAULT 'application/pdf',
  file_size_bytes BIGINT NOT NULL DEFAULT 0,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON eeg_documents(eeg_id, sort_order);
