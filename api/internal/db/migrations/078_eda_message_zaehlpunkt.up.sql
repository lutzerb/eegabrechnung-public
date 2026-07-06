ALTER TABLE eda_messages ADD COLUMN zaehlpunkt TEXT NOT NULL DEFAULT '';

-- Backfill: extract every <MeteringPoint> value (any XML namespace prefix) out of the
-- existing raw XML payloads so historical messages become filterable too. Most message
-- types carry exactly one MeteringPoint (the Zählpunkt); ECMPList/EC_PODLIST list
-- responses can carry many, so all distinct values are space-joined.
UPDATE eda_messages
SET zaehlpunkt = COALESCE(
    (SELECT string_agg(DISTINCT m[1], ' ')
     FROM regexp_matches(xml_payload, '<[A-Za-z0-9]*:?MeteringPoint[^>]*>([^<]+)</[A-Za-z0-9]*:?MeteringPoint>', 'g') AS m),
    ''
)
WHERE zaehlpunkt = '';

CREATE INDEX idx_eda_messages_zaehlpunkt ON eda_messages (zaehlpunkt) WHERE zaehlpunkt <> '';
