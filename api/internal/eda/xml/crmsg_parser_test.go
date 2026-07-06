package xml_test

import (
	"testing"
	"time"

	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// minimalCRMsg is a trimmed ConsumptionRecord with two 15-minute intervals for testing.
// The real files (34c0c79c-*.xml) are ~46k tokens — we use a minimal version here.
// MeterCode "1-1:1.9.0 G.01": total consumption (1.9.0) without community share (no P.01).
// MeterCode "1-1:1.9.0 P.01": community-assigned consumption share.
const minimalCRMsgConsumption = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:ConsumptionRecord xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p41"
                       xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.41">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-03-26T09:05:04Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>DATEN_CRMSG</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202603261005042567751123742</ns1:MessageId>
    <ns1:ConversationId>AT002000202601231003107881033013733</ns1:ConversationId>
    <ns1:ProcessDate>2026-03-26</ns1:ProcessDate>
    <ns1:MeteringPoint>AT0020000000000000000000020089835</ns1:MeteringPoint>
    <ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID>
    <ns0:Energy>
      <ns0:MeteringReason>00</ns0:MeteringReason>
      <ns0:MeteringPeriodStart>2026-03-25T00:00:00+01:00</ns0:MeteringPeriodStart>
      <ns0:MeteringPeriodEnd>2026-03-25T00:30:00+01:00</ns0:MeteringPeriodEnd>
      <ns0:MeteringIntervall>QH</ns0:MeteringIntervall>
      <ns0:NumberOfMeteringIntervall>2</ns0:NumberOfMeteringIntervall>
      <ns0:EnergyData MeterCode="1-1:1.9.0 G.01" UOM="KWH">
        <ns0:EP>
          <ns0:DTF>2026-03-25T00:00:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T00:15:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>0.012</ns0:BQ>
        </ns0:EP>
        <ns0:EP>
          <ns0:DTF>2026-03-25T00:15:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T00:30:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>0.019</ns0:BQ>
        </ns0:EP>
      </ns0:EnergyData>
      <ns0:EnergyData MeterCode="1-1:1.9.0 P.01" UOM="KWH">
        <ns0:EP>
          <ns0:DTF>2026-03-25T00:00:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T00:15:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>0.005</ns0:BQ>
        </ns0:EP>
        <ns0:EP>
          <ns0:DTF>2026-03-25T00:15:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T00:30:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>0.008</ns0:BQ>
        </ns0:EP>
      </ns0:EnergyData>
    </ns0:Energy>
  </ns0:ProcessDirectory>
</ns0:ConsumptionRecord>`

// minimalCRMsgGeneration uses 2.9.0 OBIS codes for a generation meter point.
const minimalCRMsgGeneration = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:ConsumptionRecord xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p41"
                       xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.41">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-03-26T09:05:04Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>DATEN_CRMSG</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202603261005042567751999999</ns1:MessageId>
    <ns1:ConversationId>AT002000202601231003107881033099999</ns1:ConversationId>
    <ns1:ProcessDate>2026-03-26</ns1:ProcessDate>
    <ns1:MeteringPoint>AT0020000000000000000000100266304</ns1:MeteringPoint>
    <ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID>
    <ns0:Energy>
      <ns0:MeteringReason>00</ns0:MeteringReason>
      <ns0:MeteringPeriodStart>2026-03-25T10:00:00+01:00</ns0:MeteringPeriodStart>
      <ns0:MeteringPeriodEnd>2026-03-25T10:30:00+01:00</ns0:MeteringPeriodEnd>
      <ns0:MeteringIntervall>QH</ns0:MeteringIntervall>
      <ns0:NumberOfMeteringIntervall>2</ns0:NumberOfMeteringIntervall>
      <ns0:EnergyData MeterCode="1-1:2.9.0 G.01" UOM="KWH">
        <ns0:EP>
          <ns0:DTF>2026-03-25T10:00:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T10:15:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>1.234</ns0:BQ>
        </ns0:EP>
        <ns0:EP>
          <ns0:DTF>2026-03-25T10:15:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T10:30:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>1.567</ns0:BQ>
        </ns0:EP>
      </ns0:EnergyData>
      <ns0:EnergyData MeterCode="1-1:2.9.0 P.01" UOM="KWH">
        <ns0:EP>
          <ns0:DTF>2026-03-25T10:00:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T10:15:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>1.000</ns0:BQ>
        </ns0:EP>
        <ns0:EP>
          <ns0:DTF>2026-03-25T10:15:00+01:00</ns0:DTF>
          <ns0:DTT>2026-03-25T10:30:00+01:00</ns0:DTT>
          <ns0:MM>L1</ns0:MM>
          <ns0:BQ>1.200</ns0:BQ>
        </ns0:EP>
      </ns0:EnergyData>
    </ns0:Energy>
  </ns0:ProcessDirectory>
</ns0:ConsumptionRecord>`

const minimalCRMsgSIMU = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:ConsumptionRecord xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p41"
                       xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="SIMU" Duplicate="false" SchemaVersion="01.41">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>DATEN_CRMSG</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>SIMU-MSG-001</ns1:MessageId>
    <ns1:MeteringPoint>AT0020000000000000000000020089835</ns1:MeteringPoint>
    <ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID>
    <ns0:Energy>
      <ns0:MeteringPeriodStart>2026-03-25T00:00:00+01:00</ns0:MeteringPeriodStart>
      <ns0:MeteringPeriodEnd>2026-03-25T00:15:00+01:00</ns0:MeteringPeriodEnd>
      <ns0:EnergyData MeterCode="1-1:1.9.0 G.01" UOM="KWH">
        <ns0:EP>
          <ns0:DTF>2026-03-25T00:00:00+01:00</ns0:DTF>
          <ns0:BQ>0.010</ns0:BQ>
        </ns0:EP>
      </ns0:EnergyData>
    </ns0:Energy>
  </ns0:ProcessDirectory>
</ns0:ConsumptionRecord>`

func TestParseCRMsg_Consumption(t *testing.T) {
	record, err := edaxml.ParseCRMsg(minimalCRMsgConsumption)
	if err != nil {
		t.Fatalf("ParseCRMsg: %v", err)
	}

	if record.DocumentMode != "PROD" {
		t.Errorf("DocumentMode = %q, want PROD", record.DocumentMode)
	}
	if record.MessageID != "AT002000202603261005042567751123742" {
		t.Errorf("MessageID = %q, unexpected", record.MessageID)
	}
	if record.Zaehlpunkt != "AT0020000000000000000000020089835" {
		t.Errorf("Zaehlpunkt = %q, want AT0020000000000000000000020089835", record.Zaehlpunkt)
	}
	if record.GemeinschaftID != "AT00200000000RC105970000000001289" {
		t.Errorf("GemeinschaftID = %q, want AT00200000000RC105970000000001289", record.GemeinschaftID)
	}
	if len(record.Energies) != 1 {
		t.Fatalf("Energies count = %d, want 1", len(record.Energies))
	}

	block := record.Energies[0]
	if block.Intervall != "QH" {
		t.Errorf("Intervall = %q, want QH", block.Intervall)
	}
	if len(block.Data) != 2 {
		t.Fatalf("EnergyData count = %d, want 2 (total + community)", len(block.Data))
	}

	// First EnergyData: total consumption (no P.01 suffix)
	totalData := block.Data[0]
	if len(totalData.Positions) != 2 {
		t.Fatalf("total Positions count = %d, want 2", len(totalData.Positions))
	}
	// Values are in kWh (NOT Wh) — do not divide by 1000
	if totalData.Positions[0].Value != 0.012 {
		t.Errorf("total[0].Value = %v, want 0.012", totalData.Positions[0].Value)
	}
	if totalData.Positions[1].Value != 0.019 {
		t.Errorf("total[1].Value = %v, want 0.019", totalData.Positions[1].Value)
	}

	// Timestamps should be parsed correctly
	wantTS0 := time.Date(2026, 3, 24, 23, 0, 0, 0, time.UTC) // 2026-03-25T00:00:00+01:00 → UTC
	if !totalData.Positions[0].From.Equal(wantTS0) {
		t.Errorf("total[0].From = %v, want %v", totalData.Positions[0].From, wantTS0)
	}
}

func TestParseCRMsg_Generation(t *testing.T) {
	record, err := edaxml.ParseCRMsg(minimalCRMsgGeneration)
	if err != nil {
		t.Fatalf("ParseCRMsg: %v", err)
	}

	if record.Zaehlpunkt != "AT0020000000000000000000100266304" {
		t.Errorf("Zaehlpunkt = %q, want AT0020000000000000000000100266304", record.Zaehlpunkt)
	}
	if len(record.Energies) != 1 {
		t.Fatalf("Energies count = %d, want 1", len(record.Energies))
	}

	block := record.Energies[0]
	if len(block.Data) != 2 {
		t.Fatalf("EnergyData count = %d, want 2", len(block.Data))
	}
	// Generation OBIS 2.9.0 total
	total := block.Data[0]
	if total.Positions[0].Value != 1.234 {
		t.Errorf("generation total[0].Value = %v, want 1.234", total.Positions[0].Value)
	}
}

func TestParseCRMsg_SIMU(t *testing.T) {
	record, err := edaxml.ParseCRMsg(minimalCRMsgSIMU)
	if err != nil {
		t.Fatalf("ParseCRMsg: %v", err)
	}
	if record.DocumentMode != "SIMU" {
		t.Errorf("DocumentMode = %q, want SIMU", record.DocumentMode)
	}
}

func TestParseCRMsg_InvalidXML(t *testing.T) {
	_, err := edaxml.ParseCRMsg("not xml <<<")
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}
