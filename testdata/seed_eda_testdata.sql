-- Seed EDA test data for "Sonnenstrom Mustertal" (EEG 5d0151e8-...)
-- Run with: docker compose exec eegabrechnung-postgres psql -U eegabrechnung -d eegabrechnung -f /tmp/seed_eda_testdata.sql
-- (copy to container first or pipe via stdin)

BEGIN;

-- ── 1. Configure EDA on the test EEG ──────────────────────────────────────

UPDATE eegs
SET eda_marktpartner_id = 'RC105970',
    eda_netzbetreiber_id = 'AT002000',
    gemeinschaft_id      = 'AT00200000000RC105970000000001289'
WHERE id = '5d0151e8-8714-4605-9f20-70ec5d5d5b46';

-- ── 2. EDA Processes ───────────────────────────────────────────────────────
-- Meter points (for reference):
--   Biobauernhof  AT0010000000000000001000034567890123  id: 1615fa97-da50-469b-b783-0b9e0654f8e9
--   Hans          AT0010000000000000001000011234567890  id: 76ed3810-b8b4-44e0-a73c-35571165f1de
--   Maria (con)   AT0010000000000000001000022345678901  id: f4c1b54a-144c-43f8-8932-eff4cec79a92
--   Maria (gen)   AT0010000000000000001000023456789012  id: 5180a6bd-d1ed-4726-bb71-9d21fe353cd0

-- Process 1: EC_REQ_ONL for Biobauernhof — completed (online consent flow)
INSERT INTO eda_processes
  (id, eeg_id, meter_point_id, process_type, status, conversation_id, zaehlpunkt,
   valid_from, participation_factor, share_type, initiated_at, deadline_at, completed_at)
VALUES (
  '11111111-eda0-4000-a000-000000000001',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  '1615fa97-da50-469b-b783-0b9e0654f8e9',
  'EC_REQ_ONL', 'confirmed',
  'RC105970202601221000000000000000001',
  'AT0010000000000000001000034567890123',
  '2026-01-22', 100.0, 'RC_R',
  '2026-01-22 10:00:00+01', '2026-03-22 10:00:00+01', '2026-01-26 09:15:00+01'
);

-- Process 2: EC_EINZEL_ANM for Hans — confirmed (offline registration)
INSERT INTO eda_processes
  (id, eeg_id, meter_point_id, process_type, status, conversation_id, zaehlpunkt,
   valid_from, participation_factor, share_type, initiated_at, deadline_at, completed_at)
VALUES (
  '11111111-eda0-4000-a000-000000000002',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  '76ed3810-b8b4-44e0-a73c-35571165f1de',
  'EC_EINZEL_ANM', 'confirmed',
  'RC105970202602051300000000000000002',
  'AT0010000000000000001000011234567890',
  '2026-02-10', 100.0, 'GC',
  '2026-02-05 13:00:00+01', '2026-04-05 13:00:00+01', '2026-02-09 14:30:00+01'
);

-- Process 3: EC_EINZEL_ABM for Maria (consumption) — sent, open
INSERT INTO eda_processes
  (id, eeg_id, meter_point_id, process_type, status, conversation_id, zaehlpunkt,
   valid_from, participation_factor, share_type, initiated_at, deadline_at)
VALUES (
  '11111111-eda0-4000-a000-000000000003',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'f4c1b54a-144c-43f8-8932-eff4cec79a92',
  'EC_EINZEL_ABM', 'sent',
  'RC105970202603241430000000000000003',
  'AT0010000000000000001000022345678901',
  '2026-04-01', NULL, '',
  '2026-03-24 14:30:00+01', '2026-05-24 14:30:00+01'
);

-- Process 4: EC_PRTFACT_CHG for Maria (generation) — first_confirmed
INSERT INTO eda_processes
  (id, eeg_id, meter_point_id, process_type, status, conversation_id, zaehlpunkt,
   valid_from, participation_factor, share_type, initiated_at, deadline_at)
VALUES (
  '11111111-eda0-4000-a000-000000000004',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  '5180a6bd-d1ed-4726-bb71-9d21fe353cd0',
  'EC_PRTFACT_CHG', 'first_confirmed',
  'RC105970202603200900000000000000004',
  'AT0010000000000000001000023456789012',
  '2026-04-01', 75.0, 'RC_R',
  '2026-03-20 09:00:00+01', NULL
);

-- Process 5: EC_EINZEL_ANM — rejected
INSERT INTO eda_processes
  (id, eeg_id, process_type, status, conversation_id, zaehlpunkt,
   valid_from, participation_factor, share_type, initiated_at, deadline_at, completed_at, error_msg)
VALUES (
  '11111111-eda0-4000-a000-000000000005',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'EC_EINZEL_ANM', 'rejected',
  'RC105970202602150800000000000000005',
  'AT0010000000000000001000099999999999',
  '2026-02-20', 100.0, 'GC',
  '2026-02-15 08:00:00+01', '2026-04-15 08:00:00+01', '2026-02-16 11:00:00+01',
  'ABLEHNUNG_ANM: Zählpunkt nicht in diesem Netzgebiet'
);

-- Process 6: EC_REQ_PT (Zählerstandsgang) — completed
INSERT INTO eda_processes
  (id, eeg_id, meter_point_id, process_type, status, conversation_id, zaehlpunkt,
   valid_from, share_type, initiated_at, completed_at)
VALUES (
  '11111111-eda0-4000-a000-000000000006',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  '76ed3810-b8b4-44e0-a73c-35571165f1de',
  'EC_REQ_PT', 'confirmed',
  'RC105970202603010700000000000000006',
  'AT0010000000000000001000011234567890',
  '2026-03-01', '',
  '2026-03-01 07:00:00+01', '2026-03-01 07:45:00+01'
);

-- ── 3. EDA Messages ────────────────────────────────────────────────────────

-- === Process 1 (EC_REQ_ONL, Biobauernhof) ===

-- Outbound: ANFORDERUNG_ECON (EEG initiates consent request)
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000001',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'RC105970202601221000000000000000001',
  'EC_REQ_ONL', 'ANFORDERUNG_ECON',
  'EDA EC_REQ_ONL AT00200000000RC105970000000001289',
  '(CPRequest XML)',
  'outbound', 'sent',
  $xml$<cp:CMRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p21" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p21 http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p21/CMRequest_01p21.xsd">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.21">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>RC105970</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT002000</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-25T16:17:45</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECON</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>RC105970202603251617458050000000001</ct:MessageId>
    <ct:ConversationId>RC105970202603251617458050000000001</ct:ConversationId>
    <cp:ProcessDate>2026-03-25</cp:ProcessDate>
    <cp:MeteringPoint>AT0020000000000000000000100242261</cp:MeteringPoint>
    <cp:CMRequestId>DKPSPX56</cp:CMRequestId>
    <cp:CMRequest>
      <cp:ReqDatType>EnergyCommunityRegistration</cp:ReqDatType>
      <cp:DateFrom>2026-03-26</cp:DateFrom>
      <cp:ECID>AT00200000000RC105970000000001289</cp:ECID>
      <cp:ECPartFact>100</cp:ECPartFact>
      <cp:EnergyDirection>GENERATION</cp:EnergyDirection>
    </cp:CMRequest>
  </cp:ProcessDirectory>
</cp:CMRequest>$xml$, '2026-01-22 10:00:00+01'
);

-- Inbound: ANTWORT_ECON (intermediate ack from NB)
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000002',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202601221005000000000000001',
  'ANTWORT_ECON', 'CMNotification',
  'CMNotification ANTWORT_ECON AT0010000000000000001000034567890123',
  '(CMNotification XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?><ns0:CMNotification xmlns:ns0="http://www.ebutilities.at/schemata/customerconsent/cmnotification/01p12" xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"><ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12"><ns1:RoutingHeader><ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender><ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver><ns1:DocumentCreationDateTime>2026-03-26T12:55:57.4017420Z</ns1:DocumentCreationDateTime></ns1:RoutingHeader><ns1:Sector>01</ns1:Sector><ns0:MessageCode>ZUSTIMMUNG_ECON</ns0:MessageCode></ns0:MarketParticipantDirectory><ns0:ProcessDirectory><ns1:MessageId>AT002000202603251717485477751697266</ns1:MessageId><ns1:ConversationId>RC105970202603251617458050000000001</ns1:ConversationId><ns0:CMRequestId>DKPSPX56</ns0:CMRequestId><ns0:ResponseData><ns0:ConsentId>AT00200020260326135553934DKPSPX56</ns0:ConsentId><ns0:MeteringPoint>AT0020000000000000000000100242261</ns0:MeteringPoint><ns0:ResponseCode>175</ns0:ResponseCode></ns0:ResponseData></ns0:ProcessDirectory></ns0:CMNotification>$xml$, '2026-01-22 10:05:00+01'
);

-- Inbound: ZUSTIMMUNG_ECON (consent granted)
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000003',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202601241400000000000000001',
  'ZUSTIMMUNG_ECON', 'CMNotification',
  'CMNotification ZUSTIMMUNG_ECON AT0010000000000000001000034567890123',
  '(CMNotification XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?><ns0:CMNotification xmlns:ns0="http://www.ebutilities.at/schemata/customerconsent/cmnotification/01p12" xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"><ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12"><ns1:RoutingHeader><ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender><ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver><ns1:DocumentCreationDateTime>2026-03-26T12:55:57.4017420Z</ns1:DocumentCreationDateTime></ns1:RoutingHeader><ns1:Sector>01</ns1:Sector><ns0:MessageCode>ZUSTIMMUNG_ECON</ns0:MessageCode></ns0:MarketParticipantDirectory><ns0:ProcessDirectory><ns1:MessageId>AT002000202603251717485477751697266</ns1:MessageId><ns1:ConversationId>RC105970202603251617458050000000001</ns1:ConversationId><ns0:CMRequestId>DKPSPX56</ns0:CMRequestId><ns0:ResponseData><ns0:ConsentId>AT00200020260326135553934DKPSPX56</ns0:ConsentId><ns0:MeteringPoint>AT0020000000000000000000100242261</ns0:MeteringPoint><ns0:ResponseCode>175</ns0:ResponseCode></ns0:ResponseData></ns0:ProcessDirectory></ns0:CMNotification>$xml$, '2026-01-24 14:05:00+01'
);

-- Inbound: ABSCHLUSS_ECON (final registration confirmed via ECMPList)
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000004',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202601261000000000000000001',
  'ABSCHLUSS_ECON', 'ECMPList',
  'ECMPList ABSCHLUSS_ECON AT00200000000RC105970000000001289',
  '(ECMPList XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?><ns0:ECMPList xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10" xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"><ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.10"><ns1:RoutingHeader><ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender><ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver><ns1:DocumentCreationDateTime>2026-03-27T03:00:50.3604740Z</ns1:DocumentCreationDateTime></ns1:RoutingHeader><ns1:Sector>01</ns1:Sector><ns0:MessageCode>SENDEN_ECP</ns0:MessageCode></ns0:MarketParticipantDirectory><ns0:ProcessDirectory><ns0:MessageId>AT002000202603270400492547752824639</ns0:MessageId><ns0:ConversationId>RC105970202603270300456420000000001</ns0:ConversationId><ns0:ProcessDate>2026-03-27</ns0:ProcessDate><ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID><ns0:ECType>RC_R</ns0:ECType><ns0:ECDisModel>D</ns0:ECDisModel><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100242261</ns0:MeteringPoint><ns0:ConsentId>AT00200020260326135553934DKPSPX56</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-03-26</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2026-03-26</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100266304</ns0:MeteringPoint><ns0:ConsentId>AT00200020260121210740403YVJDAPES</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-22</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2026-01-22</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100383768</ns0:MeteringPoint><ns0:ConsentId>AT00200020250617160901721P2EAQL2T</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-18</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2025-06-18</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100383769</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618084846395SM5X7NXM</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020073384</ns0:MeteringPoint><ns0:ConsentId>AT00200020250617160846577MVDIKTGI</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-18</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-18</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020089835</ns0:MeteringPoint><ns0:ConsentId>AT00200020260121210722095LPVOSJHS</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-22</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-01-22</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020091072</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618084841461O5QG5LRT</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020091266</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618103551882RQUYEJQ</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020111710</ns0:MeteringPoint><ns0:ConsentId>AT00200020260326135611039KGL63O42</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-03-26</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-03-26</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100215718</ns0:MeteringPoint><ns0:ConsentId>AT00200020260117202347198HHWDRYLA</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-17</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-01-17</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData></ns0:ProcessDirectory></ns0:ECMPList>$xml$, '2026-01-26 09:15:00+01'
);

-- === Process 2 (EC_EINZEL_ANM, Hans) ===

-- Outbound: EC_EINZEL_ANM
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000005',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'RC105970202602051300000000000000002',
  'EC_EINZEL_ANM', 'CPRequest',
  'EDA EC_EINZEL_ANM AT00200000000RC105970000000001289',
  '(CPRequest XML)',
  'outbound', 'sent',
  $xml$<cp:CPRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12 http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12/CPRequest_01p12.xsd">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>RC105970</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT002000</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-25T03:00:47</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECP</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>RC105970202603250300479210000000001</ct:MessageId>
    <ct:ConversationId>RC105970202603250300479210000000001</ct:ConversationId>
    <ct:ProcessDate>2026-03-25</ct:ProcessDate>
    <ct:MeteringPoint>AT00200000000RC105970000000001289</ct:MeteringPoint>
    <cp:Extension>
      <cp:DateTimeFrom>2022-01-01T00:00:00+01:00</cp:DateTimeFrom>
      <cp:DateTimeTo>2026-03-24T00:00:00+01:00</cp:DateTimeTo>
      <cp:AssumptionOfCosts>false</cp:AssumptionOfCosts>
    </cp:Extension>
  </cp:ProcessDirectory>
</cp:CPRequest>$xml$, '2026-02-05 13:00:00+01'
);

-- Inbound: ERSTE_ANM
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000006',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202602061100000000000000001',
  'ERSTE_ANM', 'CPDocument',
  'CPDocument ERSTE_ANM AT0010000000000000001000011234567890',
  '(CPDocument XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<CPDocument
    DocumentMode="PROD"
    SchemaVersion="01.40"
    xmlns="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.40">
    <MessageCode>ERSTE_ANM</MessageCode>
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000001</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000002</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-20T05:08:03</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
  </MarketParticipantDirectory>
  <ProcessDirectory>
    <ct:MessageId>e131703b-e690-4ed2-8366-88c0471d38b8</ct:MessageId>
    <ct:ConversationId>02e56af1-bfcb-4a09-9a1c-c4b2225fe840</ct:ConversationId>
    <ct:ProcessDate>2026-03-20</ct:ProcessDate>
    <ct:MeteringPoint>AT0010000000000000000000000TESTc54eC</ct:MeteringPoint>
    <CommunityID>AT0010000000000000000000000TESTc54eE</CommunityID>
    <ValidFrom>2026-03-20</ValidFrom>
  </ProcessDirectory>
</CPDocument>$xml$, '2026-02-06 11:05:00+01'
);

-- Inbound: FINALE_ANM
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000007',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202602091600000000000000001',
  'FINALE_ANM', 'CPDocument',
  'CPDocument FINALE_ANM AT0010000000000000001000011234567890',
  '(CPDocument XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<CPDocument
    DocumentMode="PROD"
    SchemaVersion="01.40"
    xmlns="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.40">
    <MessageCode>ERSTE_ANM</MessageCode>
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000001</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000002</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-20T05:08:03</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
  </MarketParticipantDirectory>
  <ProcessDirectory>
    <ct:MessageId>e131703b-e690-4ed2-8366-88c0471d38b8</ct:MessageId>
    <ct:ConversationId>02e56af1-bfcb-4a09-9a1c-c4b2225fe840</ct:ConversationId>
    <ct:ProcessDate>2026-03-20</ct:ProcessDate>
    <ct:MeteringPoint>AT0010000000000000000000000TESTc54eC</ct:MeteringPoint>
    <CommunityID>AT0010000000000000000000000TESTc54eE</CommunityID>
    <ValidFrom>2026-03-20</ValidFrom>
  </ProcessDirectory>
</CPDocument>$xml$, '2026-02-09 14:30:00+01'
);

-- === Process 3 (EC_EINZEL_ABM, Maria consumption) ===

-- Outbound: EC_EINZEL_ABM — sent, awaiting response
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000008',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'RC105970202603241430000000000000003',
  'EC_EINZEL_ABM', 'CPRequest',
  'EDA EC_EINZEL_ABM AT00200000000RC105970000000001289',
  '(CPRequest XML)',
  'outbound', 'sent',
  $xml$<cp:CPRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12 http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12/CPRequest_01p12.xsd">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>RC105970</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT002000</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-25T03:00:47</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECP</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>RC105970202603250300479210000000001</ct:MessageId>
    <ct:ConversationId>RC105970202603250300479210000000001</ct:ConversationId>
    <ct:ProcessDate>2026-03-25</ct:ProcessDate>
    <ct:MeteringPoint>AT00200000000RC105970000000001289</ct:MeteringPoint>
    <cp:Extension>
      <cp:DateTimeFrom>2022-01-01T00:00:00+01:00</cp:DateTimeFrom>
      <cp:DateTimeTo>2026-03-24T00:00:00+01:00</cp:DateTimeTo>
      <cp:AssumptionOfCosts>false</cp:AssumptionOfCosts>
    </cp:Extension>
  </cp:ProcessDirectory>
</cp:CPRequest>$xml$, '2026-03-24 14:30:00+01'
);

-- === Process 4 (EC_PRTFACT_CHG, Maria generation) ===

-- Outbound: EC_PRTFACT_CHG
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000009',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'RC105970202603200900000000000000004',
  'EC_PRTFACT_CHG', 'CPRequest',
  'EDA EC_PRTFACT_CHG AT00200000000RC105970000000001289',
  '(CPRequest XML)',
  'outbound', 'sent',
  $xml$<cp:CPRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12 http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12/CPRequest_01p12.xsd">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>RC105970</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT002000</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-25T03:00:47</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECP</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>RC105970202603250300479210000000001</ct:MessageId>
    <ct:ConversationId>RC105970202603250300479210000000001</ct:ConversationId>
    <ct:ProcessDate>2026-03-25</ct:ProcessDate>
    <ct:MeteringPoint>AT00200000000RC105970000000001289</ct:MeteringPoint>
    <cp:Extension>
      <cp:DateTimeFrom>2022-01-01T00:00:00+01:00</cp:DateTimeFrom>
      <cp:DateTimeTo>2026-03-24T00:00:00+01:00</cp:DateTimeTo>
      <cp:AssumptionOfCosts>false</cp:AssumptionOfCosts>
    </cp:Extension>
  </cp:ProcessDirectory>
</cp:CPRequest>$xml$, '2026-03-20 09:00:00+01'
);

-- Inbound: ERSTE_ANM for factor change
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000010',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202603211100000000000000001',
  'ERSTE_ANM', 'CPDocument',
  'CPDocument ERSTE_ANM AT0010000000000000001000023456789012',
  '(CPDocument XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<CPDocument
    DocumentMode="PROD"
    SchemaVersion="01.40"
    xmlns="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.40">
    <MessageCode>ERSTE_ANM</MessageCode>
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000001</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000002</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-20T05:08:03</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
  </MarketParticipantDirectory>
  <ProcessDirectory>
    <ct:MessageId>e131703b-e690-4ed2-8366-88c0471d38b8</ct:MessageId>
    <ct:ConversationId>02e56af1-bfcb-4a09-9a1c-c4b2225fe840</ct:ConversationId>
    <ct:ProcessDate>2026-03-20</ct:ProcessDate>
    <ct:MeteringPoint>AT0010000000000000000000000TESTc54eC</ct:MeteringPoint>
    <CommunityID>AT0010000000000000000000000TESTc54eE</CommunityID>
    <ValidFrom>2026-03-20</ValidFrom>
  </ProcessDirectory>
</CPDocument>$xml$, '2026-03-21 11:30:00+01'
);

-- === Process 5 (EC_EINZEL_ANM, rejected) ===

-- Outbound: EC_EINZEL_ANM for unknown meter
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000011',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'RC105970202602150800000000000000005',
  'EC_EINZEL_ANM', 'CPRequest',
  'EDA EC_EINZEL_ANM AT00200000000RC105970000000001289',
  '(CPRequest XML)',
  'outbound', 'sent',
  $xml$<cp:CPRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12 http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12/CPRequest_01p12.xsd">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>RC105970</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT002000</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-25T03:00:47</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECP</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>RC105970202603250300479210000000001</ct:MessageId>
    <ct:ConversationId>RC105970202603250300479210000000001</ct:ConversationId>
    <ct:ProcessDate>2026-03-25</ct:ProcessDate>
    <ct:MeteringPoint>AT00200000000RC105970000000001289</ct:MeteringPoint>
    <cp:Extension>
      <cp:DateTimeFrom>2022-01-01T00:00:00+01:00</cp:DateTimeFrom>
      <cp:DateTimeTo>2026-03-24T00:00:00+01:00</cp:DateTimeTo>
      <cp:AssumptionOfCosts>false</cp:AssumptionOfCosts>
    </cp:Extension>
  </cp:ProcessDirectory>
</cp:CPRequest>$xml$, '2026-02-15 08:00:00+01'
);

-- Inbound: ABLEHNUNG_ANM
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at,
   error_msg)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000012',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202602161100000000000000001',
  'ABLEHNUNG_ANM', 'CPDocument',
  'CPDocument ABLEHNUNG_ANM AT0010000000000000001000099999999999',
  '(CPDocument XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<CPDocument
    DocumentMode="PROD"
    SchemaVersion="01.40"
    xmlns="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.40">
    <MessageCode>ERSTE_ANM</MessageCode>
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000001</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT001000000000000000000000000000002</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-20T05:08:03</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
  </MarketParticipantDirectory>
  <ProcessDirectory>
    <ct:MessageId>e131703b-e690-4ed2-8366-88c0471d38b8</ct:MessageId>
    <ct:ConversationId>02e56af1-bfcb-4a09-9a1c-c4b2225fe840</ct:ConversationId>
    <ct:ProcessDate>2026-03-20</ct:ProcessDate>
    <ct:MeteringPoint>AT0010000000000000000000000TESTc54eC</ct:MeteringPoint>
    <CommunityID>AT0010000000000000000000000TESTc54eE</CommunityID>
    <ValidFrom>2026-03-20</ValidFrom>
  </ProcessDirectory>
</CPDocument>$xml$, '2026-02-16 11:00:00+01',
  'Zählpunkt nicht in diesem Netzgebiet'
);

-- === Process 6 (EC_REQ_PT / Zählerstandsgang) ===

-- Outbound: ANFORDERUNG_ECP
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000013',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'RC105970202603010700000000000000006',
  'EC_REQ_PT', 'CPRequest',
  'EDA EC_REQ_PT AT00200000000RC105970000000001289',
  '(CPRequest XML)',
  'outbound', 'sent',
  $xml$<cp:CPRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12 http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12/CPRequest_01p12.xsd">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>RC105970</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>AT002000</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2026-03-25T03:00:47</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECP</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>RC105970202603250300479210000000001</ct:MessageId>
    <ct:ConversationId>RC105970202603250300479210000000001</ct:ConversationId>
    <ct:ProcessDate>2026-03-25</ct:ProcessDate>
    <ct:MeteringPoint>AT00200000000RC105970000000001289</ct:MeteringPoint>
    <cp:Extension>
      <cp:DateTimeFrom>2022-01-01T00:00:00+01:00</cp:DateTimeFrom>
      <cp:DateTimeTo>2026-03-24T00:00:00+01:00</cp:DateTimeTo>
      <cp:AssumptionOfCosts>false</cp:AssumptionOfCosts>
    </cp:Extension>
  </cp:ProcessDirectory>
</cp:CPRequest>$xml$, '2026-03-01 07:00:00+01'
);

-- Inbound: SENDEN_ECP (meter point list)
INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES (
  'aaaaaaaa-eda0-4000-b000-000000000014',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202603010800000000000000001',
  'SENDEN_ECP', 'ECMPList',
  'ECMPList SENDEN_ECP AT00200000000RC105970000000001289',
  '(ECMPList XML)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?><ns0:ECMPList xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10" xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"><ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.10"><ns1:RoutingHeader><ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender><ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver><ns1:DocumentCreationDateTime>2026-03-27T03:00:50.3604740Z</ns1:DocumentCreationDateTime></ns1:RoutingHeader><ns1:Sector>01</ns1:Sector><ns0:MessageCode>SENDEN_ECP</ns0:MessageCode></ns0:MarketParticipantDirectory><ns0:ProcessDirectory><ns0:MessageId>AT002000202603270400492547752824639</ns0:MessageId><ns0:ConversationId>RC105970202603270300456420000000001</ns0:ConversationId><ns0:ProcessDate>2026-03-27</ns0:ProcessDate><ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID><ns0:ECType>RC_R</ns0:ECType><ns0:ECDisModel>D</ns0:ECDisModel><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100242261</ns0:MeteringPoint><ns0:ConsentId>AT00200020260326135553934DKPSPX56</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-03-26</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2026-03-26</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100266304</ns0:MeteringPoint><ns0:ConsentId>AT00200020260121210740403YVJDAPES</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-22</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2026-01-22</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100383768</ns0:MeteringPoint><ns0:ConsentId>AT00200020250617160901721P2EAQL2T</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-18</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2025-06-18</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100383769</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618084846395SM5X7NXM</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020073384</ns0:MeteringPoint><ns0:ConsentId>AT00200020250617160846577MVDIKTGI</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-18</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-18</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020089835</ns0:MeteringPoint><ns0:ConsentId>AT00200020260121210722095LPVOSJHS</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-22</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-01-22</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020091072</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618084841461O5QG5LRT</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020091266</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618103551882RQUYEJQ</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020111710</ns0:MeteringPoint><ns0:ConsentId>AT00200020260326135611039KGL63O42</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-03-26</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-03-26</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100215718</ns0:MeteringPoint><ns0:ConsentId>AT00200020260117202347198HHWDRYLA</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-17</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-01-17</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData></ns0:ProcessDirectory></ns0:ECMPList>$xml$, '2026-03-01 07:45:00+01'
);

-- === Energiedaten (DATEN_CRMSG) — regular energy data pushes from NB ===

INSERT INTO eda_messages
  (id, eeg_id, message_id, process, message_type, subject, body, direction, status, xml_payload, processed_at)
VALUES
-- Biobauernhof, Feb energy data
(
  'aaaaaaaa-eda0-4000-b000-000000000015',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202602261000000000000000001',
  'DATEN_CRMSG', 'ConsumptionRecord',
  'ConsumptionRecord DATEN_CRMSG AT0010000000000000001000034567890123',
  '(ConsumptionRecord XML — 28 days)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<!-- Sample ConsumptionRecord (CR_MSG) for testing.
     Schema: http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30
     This represents energy data for one consumer meter point for 2024-01-01 (4 quarter-hour slots shown).
-->
<cr:ConsumptionRecord
    xmlns:cr="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">

  <cr:MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.30">
    <ct:RoutingHeader>
      <ct:Sender>
        <ct:MessageAddress AddressType="ECNumber">AT001NB123456789</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver>
        <ct:MessageAddress AddressType="ECNumber">RC000001</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2024-01-02T06:00:00+01:00</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cr:MessageCode>DATEN_CRMSG</cr:MessageCode>
  </cr:MarketParticipantDirectory>

  <cr:ProcessDirectory>
    <ct:MessageId>CR-2024010200001</ct:MessageId>
    <ct:ConversationId>CR-2024010200001</ct:ConversationId>
    <ct:ProcessDate>2024-01-02</ct:ProcessDate>
    <ct:MeteringPoint>AT0030000000000000000000000000001</ct:MeteringPoint>
    <cr:DeliveryPoint AddressType="ECNumber">RC000001</cr:DeliveryPoint>

    <cr:Energy>
      <cr:MeteringReason>02</cr:MeteringReason>
      <cr:MeteringPeriodStart>2024-01-01T00:00:00+01:00</cr:MeteringPeriodStart>
      <cr:MeteringPeriodEnd>2024-01-02T00:00:00+01:00</cr:MeteringPeriodEnd>
      <cr:MeteringIntervall>QH</cr:MeteringIntervall>
      <cr:NumberOfMeteringIntervall>96</cr:NumberOfMeteringIntervall>

      <!-- Gesamtverbrauch (total consumption at meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 G.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.250</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.275</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.300</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.225</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

      <!-- Zugewiesene Erzeugung aus der EG (community share allocated to this meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 P.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.100</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.110</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.120</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.090</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

    </cr:Energy>
  </cr:ProcessDirectory>
</cr:ConsumptionRecord>$xml$, '2026-02-26 10:05:00+01'
),
-- Hans, Feb energy data
(
  'aaaaaaaa-eda0-4000-b000-000000000016',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202602261000000000000000002',
  'DATEN_CRMSG', 'ConsumptionRecord',
  'ConsumptionRecord DATEN_CRMSG AT0010000000000000001000011234567890',
  '(ConsumptionRecord XML — 28 days)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<!-- Sample ConsumptionRecord (CR_MSG) for testing.
     Schema: http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30
     This represents energy data for one consumer meter point for 2024-01-01 (4 quarter-hour slots shown).
-->
<cr:ConsumptionRecord
    xmlns:cr="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">

  <cr:MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.30">
    <ct:RoutingHeader>
      <ct:Sender>
        <ct:MessageAddress AddressType="ECNumber">AT001NB123456789</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver>
        <ct:MessageAddress AddressType="ECNumber">RC000001</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2024-01-02T06:00:00+01:00</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cr:MessageCode>DATEN_CRMSG</cr:MessageCode>
  </cr:MarketParticipantDirectory>

  <cr:ProcessDirectory>
    <ct:MessageId>CR-2024010200001</ct:MessageId>
    <ct:ConversationId>CR-2024010200001</ct:ConversationId>
    <ct:ProcessDate>2024-01-02</ct:ProcessDate>
    <ct:MeteringPoint>AT0030000000000000000000000000001</ct:MeteringPoint>
    <cr:DeliveryPoint AddressType="ECNumber">RC000001</cr:DeliveryPoint>

    <cr:Energy>
      <cr:MeteringReason>02</cr:MeteringReason>
      <cr:MeteringPeriodStart>2024-01-01T00:00:00+01:00</cr:MeteringPeriodStart>
      <cr:MeteringPeriodEnd>2024-01-02T00:00:00+01:00</cr:MeteringPeriodEnd>
      <cr:MeteringIntervall>QH</cr:MeteringIntervall>
      <cr:NumberOfMeteringIntervall>96</cr:NumberOfMeteringIntervall>

      <!-- Gesamtverbrauch (total consumption at meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 G.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.250</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.275</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.300</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.225</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

      <!-- Zugewiesene Erzeugung aus der EG (community share allocated to this meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 P.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.100</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.110</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.120</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.090</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

    </cr:Energy>
  </cr:ProcessDirectory>
</cr:ConsumptionRecord>$xml$, '2026-02-26 10:05:00+01'
),
-- Biobauernhof, März energy data (latest)
(
  'aaaaaaaa-eda0-4000-b000-000000000017',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202603261005042567751123742',
  'DATEN_CRMSG', 'ConsumptionRecord',
  'ConsumptionRecord DATEN_CRMSG AT0010000000000000001000034567890123',
  '(ConsumptionRecord XML — 25 days)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<!-- Sample ConsumptionRecord (CR_MSG) for testing.
     Schema: http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30
     This represents energy data for one consumer meter point for 2024-01-01 (4 quarter-hour slots shown).
-->
<cr:ConsumptionRecord
    xmlns:cr="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">

  <cr:MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.30">
    <ct:RoutingHeader>
      <ct:Sender>
        <ct:MessageAddress AddressType="ECNumber">AT001NB123456789</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver>
        <ct:MessageAddress AddressType="ECNumber">RC000001</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2024-01-02T06:00:00+01:00</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cr:MessageCode>DATEN_CRMSG</cr:MessageCode>
  </cr:MarketParticipantDirectory>

  <cr:ProcessDirectory>
    <ct:MessageId>CR-2024010200001</ct:MessageId>
    <ct:ConversationId>CR-2024010200001</ct:ConversationId>
    <ct:ProcessDate>2024-01-02</ct:ProcessDate>
    <ct:MeteringPoint>AT0030000000000000000000000000001</ct:MeteringPoint>
    <cr:DeliveryPoint AddressType="ECNumber">RC000001</cr:DeliveryPoint>

    <cr:Energy>
      <cr:MeteringReason>02</cr:MeteringReason>
      <cr:MeteringPeriodStart>2024-01-01T00:00:00+01:00</cr:MeteringPeriodStart>
      <cr:MeteringPeriodEnd>2024-01-02T00:00:00+01:00</cr:MeteringPeriodEnd>
      <cr:MeteringIntervall>QH</cr:MeteringIntervall>
      <cr:NumberOfMeteringIntervall>96</cr:NumberOfMeteringIntervall>

      <!-- Gesamtverbrauch (total consumption at meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 G.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.250</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.275</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.300</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.225</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

      <!-- Zugewiesene Erzeugung aus der EG (community share allocated to this meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 P.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.100</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.110</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.120</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.090</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

    </cr:Energy>
  </cr:ProcessDirectory>
</cr:ConsumptionRecord>$xml$, '2026-03-26 09:05:00+01'
),
-- Hans, März energy data
(
  'aaaaaaaa-eda0-4000-b000-000000000018',
  '5d0151e8-8714-4605-9f20-70ec5d5d5b46',
  'AT002000202603261005042567751123743',
  'DATEN_CRMSG', 'ConsumptionRecord',
  'ConsumptionRecord DATEN_CRMSG AT0010000000000000001000011234567890',
  '(ConsumptionRecord XML — 25 days)',
  'inbound', 'ack',
  $xml$<?xml version="1.0" encoding="UTF-8"?>
<!-- Sample ConsumptionRecord (CR_MSG) for testing.
     Schema: http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30
     This represents energy data for one consumer meter point for 2024-01-01 (4 quarter-hour slots shown).
-->
<cr:ConsumptionRecord
    xmlns:cr="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">

  <cr:MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.30">
    <ct:RoutingHeader>
      <ct:Sender>
        <ct:MessageAddress AddressType="ECNumber">AT001NB123456789</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver>
        <ct:MessageAddress AddressType="ECNumber">RC000001</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>2024-01-02T06:00:00+01:00</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cr:MessageCode>DATEN_CRMSG</cr:MessageCode>
  </cr:MarketParticipantDirectory>

  <cr:ProcessDirectory>
    <ct:MessageId>CR-2024010200001</ct:MessageId>
    <ct:ConversationId>CR-2024010200001</ct:ConversationId>
    <ct:ProcessDate>2024-01-02</ct:ProcessDate>
    <ct:MeteringPoint>AT0030000000000000000000000000001</ct:MeteringPoint>
    <cr:DeliveryPoint AddressType="ECNumber">RC000001</cr:DeliveryPoint>

    <cr:Energy>
      <cr:MeteringReason>02</cr:MeteringReason>
      <cr:MeteringPeriodStart>2024-01-01T00:00:00+01:00</cr:MeteringPeriodStart>
      <cr:MeteringPeriodEnd>2024-01-02T00:00:00+01:00</cr:MeteringPeriodEnd>
      <cr:MeteringIntervall>QH</cr:MeteringIntervall>
      <cr:NumberOfMeteringIntervall>96</cr:NumberOfMeteringIntervall>

      <!-- Gesamtverbrauch (total consumption at meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 G.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.250</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.275</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.300</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.225</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

      <!-- Zugewiesene Erzeugung aus der EG (community share allocated to this meter) -->
      <cr:EnergyData MeterCode="1-1:1.9.0 P.01" UOM="KWH">
        <cr:EP>
          <cr:DTF>2024-01-01T00:00:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:15:00+01:00</cr:DTT>
          <cr:BQ>0.100</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:15:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:30:00+01:00</cr:DTT>
          <cr:BQ>0.110</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:30:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T00:45:00+01:00</cr:DTT>
          <cr:BQ>0.120</cr:BQ>
        </cr:EP>
        <cr:EP>
          <cr:DTF>2024-01-01T00:45:00+01:00</cr:DTF>
          <cr:DTT>2024-01-01T01:00:00+01:00</cr:DTT>
          <cr:BQ>0.090</cr:BQ>
        </cr:EP>
      </cr:EnergyData>

    </cr:Energy>
  </cr:ProcessDirectory>
</cr:ConsumptionRecord>$xml$, '2026-03-26 09:05:00+01'
);

COMMIT;

SELECT 'Processes:' AS info, count(*) FROM eda_processes WHERE eeg_id = '5d0151e8-8714-4605-9f20-70ec5d5d5b46'
UNION ALL
SELECT 'Messages:', count(*) FROM eda_messages WHERE eeg_id = '5d0151e8-8714-4605-9f20-70ec5d5d5b46';
