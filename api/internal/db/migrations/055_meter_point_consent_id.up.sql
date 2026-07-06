-- Add consent_id to meter_points.
-- Stores the NB-assigned ConsentId from the ZUSTIMMUNG_ECON CMNotification,
-- required to send CM_REV_SP (Widerruf) when a member leaves the EEG.
ALTER TABLE meter_points ADD COLUMN consent_id TEXT NOT NULL DEFAULT '';

-- Backfill consent_id from stored ZUSTIMMUNG_ECON messages.
-- ConversationId links the inbound CMNotification to an eda_process, which links to the meter_point.
WITH ns AS (
  SELECT ARRAY[
    ARRAY['ns0','http://www.ebutilities.at/schemata/customerconsent/cmnotification/01p20'],
    ARRAY['ns1','http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20']
  ] AS arr
),
zustimmung AS (
  SELECT
    (xpath('/ns0:CMNotification/ns0:ProcessDirectory/ns0:ResponseData/ns0:ConsentId/text()',
           m.xml_payload::xml, ns.arr))[1]::text AS consent_id,
    (xpath('/ns0:CMNotification/ns0:ProcessDirectory/ns1:ConversationId/text()',
           m.xml_payload::xml, ns.arr))[1]::text AS raw_conv_id
  FROM eda_messages m, ns
  WHERE m.message_type = 'ZUSTIMMUNG_ECON'
    AND m.xml_payload LIKE '<%'
),
with_conv AS (
  SELECT
    consent_id,
    CASE WHEN length(raw_conv_id) = 32 THEN
      substring(raw_conv_id,1,8)  || '-' ||
      substring(raw_conv_id,9,4)  || '-' ||
      substring(raw_conv_id,13,4) || '-' ||
      substring(raw_conv_id,17,4) || '-' ||
      substring(raw_conv_id,21)
    ELSE raw_conv_id
    END AS conversation_id
  FROM zustimmung
  WHERE consent_id IS NOT NULL AND consent_id <> ''
),
matched AS (
  SELECT p.meter_point_id, w.consent_id
  FROM with_conv w
  JOIN eda_processes p ON p.conversation_id = w.conversation_id
  WHERE p.meter_point_id IS NOT NULL
    AND p.process_type = 'EC_REQ_ONL'
)
UPDATE meter_points mp
SET consent_id = m.consent_id
FROM matched m
WHERE mp.id = m.meter_point_id
  AND mp.consent_id = '';
