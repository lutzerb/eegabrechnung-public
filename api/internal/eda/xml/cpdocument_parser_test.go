package xml_test

import (
	"testing"
	"time"

	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// CPDocument messages are sent from the Netzbetreiber during EC_EINZEL_ANM / EC_EINZEL_ABM flows.
// We have no real CPDocument testdata files, so we use synthetic but spec-conformant XML.

const syntheticCPDocumentErsteAnm = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CPDocument xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
                xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.40">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-04-01T08:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ERSTE_ANM</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202604010800000000000000001</ns1:MessageId>
    <ns1:ConversationId>RC105970202604010700000000000000001</ns1:ConversationId>
    <ns1:ProcessDate>2026-04-01</ns1:ProcessDate>
    <ns1:MeteringPoint>AT0020000000000000000000100242261</ns1:MeteringPoint>
    <ns0:CommunityID>AT00200000000RC105970000000001289</ns0:CommunityID>
  </ns0:ProcessDirectory>
</ns0:CPDocument>`

const syntheticCPDocumentFinaleAnm = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CPDocument xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
                xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.40">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-04-02T08:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>FINALE_ANM</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202604020800000000000000001</ns1:MessageId>
    <ns1:ConversationId>RC105970202604010700000000000000001</ns1:ConversationId>
    <ns1:ProcessDate>2026-04-02</ns1:ProcessDate>
    <ns1:MeteringPoint>AT0020000000000000000000100242261</ns1:MeteringPoint>
    <ns0:CommunityID>AT00200000000RC105970000000001289</ns0:CommunityID>
    <ns0:ValidFrom>2026-04-01</ns0:ValidFrom>
  </ns0:ProcessDirectory>
</ns0:CPDocument>`

const syntheticCPDocumentAbschlussEcon = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CPDocument xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
                xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.40">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-04-02T10:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ABSCHLUSS_ECON</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202604021000000000000000001</ns1:MessageId>
    <ns1:ConversationId>RC105970202604011200000000000000001</ns1:ConversationId>
    <ns1:ProcessDate>2026-04-02</ns1:ProcessDate>
    <ns1:MeteringPoint>AT0020000000000000000000020089835</ns1:MeteringPoint>
    <ns0:CommunityID>AT00200000000RC105970000000001289</ns0:CommunityID>
  </ns0:ProcessDirectory>
</ns0:CPDocument>`

const syntheticCPDocumentAblehnung = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CPDocument xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
                xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.40">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-04-01T09:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ABLEHNUNG_ANM</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202604010900000000000000001</ns1:MessageId>
    <ns1:ConversationId>RC105970202604010700000000000000001</ns1:ConversationId>
    <ns1:ProcessDate>2026-04-01</ns1:ProcessDate>
    <ns1:MeteringPoint>AT0020000000000000000000100242261</ns1:MeteringPoint>
    <ns0:CommunityID>AT00200000000RC105970000000001289</ns0:CommunityID>
  </ns0:ProcessDirectory>
</ns0:CPDocument>`

func TestIsCPDocument(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{"ERSTE_ANM", syntheticCPDocumentErsteAnm, true},
		{"FINALE_ANM", syntheticCPDocumentFinaleAnm, true},
		{"ABSCHLUSS_ECON", syntheticCPDocumentAbschlussEcon, true},
		{"ABLEHNUNG_ANM", syntheticCPDocumentAblehnung, true},
		{"CMNotification XML", realCMNotificationZustimmung, false},
		{"ECMPList XML", realECMPListSendenECP, false},
		{"empty string", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := edaxml.IsCPDocument(tc.payload)
			if got != tc.want {
				t.Errorf("IsCPDocument() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseCPDocument_ErsteAnm(t *testing.T) {
	res, err := edaxml.ParseCPDocument(syntheticCPDocumentErsteAnm)
	if err != nil {
		t.Fatalf("ParseCPDocument: %v", err)
	}

	if res.DocumentMode != "PROD" {
		t.Errorf("DocumentMode = %q, want PROD", res.DocumentMode)
	}
	if res.MessageCode != "ERSTE_ANM" {
		t.Errorf("MessageCode = %q, want ERSTE_ANM", res.MessageCode)
	}
	if res.ConversationID != "RC105970202604010700000000000000001" {
		t.Errorf("ConversationID = %q, want RC105970202604010700000000000000001", res.ConversationID)
	}
	if res.Zaehlpunkt != "AT0020000000000000000000100242261" {
		t.Errorf("Zaehlpunkt = %q, want AT0020000000000000000000100242261", res.Zaehlpunkt)
	}
	if res.GemeinschaftID != "AT00200000000RC105970000000001289" {
		t.Errorf("GemeinschaftID = %q, want AT00200000000RC105970000000001289", res.GemeinschaftID)
	}
	if res.From != "AT002000" {
		t.Errorf("From = %q, want AT002000", res.From)
	}
	if res.To != "RC105970" {
		t.Errorf("To = %q, want RC105970", res.To)
	}
	wantDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !res.ProcessDate.Equal(wantDate) {
		t.Errorf("ProcessDate = %v, want %v", res.ProcessDate, wantDate)
	}
}

func TestParseCPDocument_FinaleAnm(t *testing.T) {
	res, err := edaxml.ParseCPDocument(syntheticCPDocumentFinaleAnm)
	if err != nil {
		t.Fatalf("ParseCPDocument: %v", err)
	}

	if res.MessageCode != "FINALE_ANM" {
		t.Errorf("MessageCode = %q, want FINALE_ANM", res.MessageCode)
	}
	// Same ConversationID as ERSTE_ANM — links back to the same process
	if res.ConversationID != "RC105970202604010700000000000000001" {
		t.Errorf("ConversationID = %q, want RC105970202604010700000000000000001", res.ConversationID)
	}
	wantValidFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !res.ValidFrom.Equal(wantValidFrom) {
		t.Errorf("ValidFrom = %v, want %v", res.ValidFrom, wantValidFrom)
	}
}

func TestParseCPDocument_AbschlussEcon(t *testing.T) {
	res, err := edaxml.ParseCPDocument(syntheticCPDocumentAbschlussEcon)
	if err != nil {
		t.Fatalf("ParseCPDocument: %v", err)
	}

	if res.MessageCode != "ABSCHLUSS_ECON" {
		t.Errorf("MessageCode = %q, want ABSCHLUSS_ECON", res.MessageCode)
	}
	if res.ConversationID != "RC105970202604011200000000000000001" {
		t.Errorf("ConversationID = %q, want RC105970202604011200000000000000001", res.ConversationID)
	}
	if res.Zaehlpunkt != "AT0020000000000000000000020089835" {
		t.Errorf("Zaehlpunkt = %q, want AT0020000000000000000000020089835", res.Zaehlpunkt)
	}
}

func TestParseCPDocument_Ablehnung(t *testing.T) {
	res, err := edaxml.ParseCPDocument(syntheticCPDocumentAblehnung)
	if err != nil {
		t.Fatalf("ParseCPDocument: %v", err)
	}

	if res.MessageCode != "ABLEHNUNG_ANM" {
		t.Errorf("MessageCode = %q, want ABLEHNUNG_ANM", res.MessageCode)
	}
	if res.ConversationID == "" {
		t.Error("ConversationID should not be empty")
	}
}

func TestParseCPDocument_SIMU(t *testing.T) {
	simu := `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CPDocument xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
                xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="SIMU" Duplicate="false" SchemaVersion="01.40">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ERSTE_ANM</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>SIMU001</ns1:MessageId>
    <ns1:ConversationId>SIMU-CONV-001</ns1:ConversationId>
  </ns0:ProcessDirectory>
</ns0:CPDocument>`

	res, err := edaxml.ParseCPDocument(simu)
	if err != nil {
		t.Fatalf("ParseCPDocument: %v", err)
	}
	if res.DocumentMode != "SIMU" {
		t.Errorf("DocumentMode = %q, want SIMU", res.DocumentMode)
	}
}

func TestParseCPDocument_InvalidXML(t *testing.T) {
	_, err := edaxml.ParseCPDocument("not xml <<<")
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}
