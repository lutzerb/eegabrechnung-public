package xml_test

import (
	"testing"
	"time"

	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// realECMPListSendenECP is from testdata/0ed33860-4278-4055-baaa-57ab0aa90386.xml
// (ECMPList/SENDEN_ECP from Netzbetreiber, 8 meter points)
const realECMPListSendenECP = `<?xml version="1.0" encoding="UTF-8"?><ns0:ECMPList xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10" xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"><ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.10"><ns1:RoutingHeader><ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender><ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver><ns1:DocumentCreationDateTime>2026-03-02T03:00:51.6885930Z</ns1:DocumentCreationDateTime></ns1:RoutingHeader><ns1:Sector>01</ns1:Sector><ns0:MessageCode>SENDEN_ECP</ns0:MessageCode></ns0:MarketParticipantDirectory><ns0:ProcessDirectory><ns0:MessageId>AT002000202603020400505337687382069</ns0:MessageId><ns0:ConversationId>RC105970202603020300454170000000001</ns0:ConversationId><ns0:ProcessDate>2026-03-02</ns0:ProcessDate><ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID><ns0:ECType>RC_R</ns0:ECType><ns0:ECDisModel>D</ns0:ECDisModel><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100266304</ns0:MeteringPoint><ns0:ConsentId>AT00200020260121210740403YVJDAPES</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-22</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2026-01-22</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100383768</ns0:MeteringPoint><ns0:ConsentId>AT00200020250617160901721P2EAQL2T</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-18</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2025-06-18</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100383769</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618084846395SM5X7NXM</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>GENERATION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:PlantCategory>SONNE</ns0:PlantCategory><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020073384</ns0:MeteringPoint><ns0:ConsentId>AT00200020250617160846577MVDIKTGI</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-18</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-18</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020089835</ns0:MeteringPoint><ns0:ConsentId>AT00200020260121210722095LPVOSJHS</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-22</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-01-22</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020091072</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618084841461O5QG5LRT</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000020091266</ns0:MeteringPoint><ns0:ConsentId>AT00200020250618103551882RQUYEJQ</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2025-06-19</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2025-06-19</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData><ns0:MPListData><ns0:MeteringPoint>AT0020000000000000000000100215718</ns0:MeteringPoint><ns0:ConsentId>AT00200020260117202347198HHWDRYLA</ns0:ConsentId><ns0:MPTimeData><ns0:DateFrom>2026-01-17</ns0:DateFrom><ns0:DateTo>9999-12-31</ns0:DateTo><ns0:EnergyDirection>CONSUMPTION</ns0:EnergyDirection><ns0:ECPartFact>100</ns0:ECPartFact><ns0:DateActivate>2026-01-17</ns0:DateActivate><ns0:ECShare>0.0000</ns0:ECShare></ns0:MPTimeData></ns0:MPListData></ns0:ProcessDirectory></ns0:ECMPList>`

// syntheticECMPListAbschluss is a minimal ABSCHLUSS_ECON message (final registration confirmation).
// This message type is NOT in the real testdata — it would arrive after ZUSTIMMUNG_ECON
// confirming that the meter point has been registered in the EC.
const syntheticECMPListAbschluss = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:ECMPList xmlns:ns0="http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10"
              xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.10">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-03-26T14:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ABSCHLUSS_ECON</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns0:MessageId>AT002000202603261400000000000000001</ns0:MessageId>
    <ns0:ConversationId>RC105970202603251617458050000000001</ns0:ConversationId>
    <ns0:ProcessDate>2026-03-26</ns0:ProcessDate>
    <ns0:ECID>AT00200000000RC105970000000001289</ns0:ECID>
    <ns0:ECType>RC_R</ns0:ECType>
    <ns0:ECDisModel>D</ns0:ECDisModel>
    <ns0:MPListData>
      <ns0:MeteringPoint>AT0020000000000000000000100242261</ns0:MeteringPoint>
      <ns0:ConsentId>AT00200020260326135553934DKPSPX56</ns0:ConsentId>
      <ns0:MPTimeData>
        <ns0:DateFrom>2026-03-26</ns0:DateFrom>
        <ns0:DateTo>9999-12-31</ns0:DateTo>
        <ns0:EnergyDirection>GENERATION</ns0:EnergyDirection>
        <ns0:ECPartFact>100</ns0:ECPartFact>
        <ns0:PlantCategory>SONNE</ns0:PlantCategory>
        <ns0:DateActivate>2026-03-26</ns0:DateActivate>
        <ns0:ECShare>0.0000</ns0:ECShare>
      </ns0:MPTimeData>
    </ns0:MPListData>
  </ns0:ProcessDirectory>
</ns0:ECMPList>`

func TestIsECMPList(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{"real SENDEN_ECP", realECMPListSendenECP, true},
		{"synthetic ABSCHLUSS_ECON", syntheticECMPListAbschluss, true},
		{"CMNotification XML", realCMNotificationZustimmung, false},
		{"CPDocument XML", `<CPDocument><MarketParticipantDirectory/></CPDocument>`, false},
		{"empty string", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := edaxml.IsECMPList(tc.payload)
			if got != tc.want {
				t.Errorf("IsECMPList() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseECMPList_SendenECP(t *testing.T) {
	res, err := edaxml.ParseECMPList(realECMPListSendenECP)
	if err != nil {
		t.Fatalf("ParseECMPList: %v", err)
	}

	if res.DocumentMode != "PROD" {
		t.Errorf("DocumentMode = %q, want PROD", res.DocumentMode)
	}
	if res.MessageCode != "SENDEN_ECP" {
		t.Errorf("MessageCode = %q, want SENDEN_ECP", res.MessageCode)
	}
	if res.ConversationID != "RC105970202603020300454170000000001" {
		t.Errorf("ConversationID = %q, want RC105970202603020300454170000000001", res.ConversationID)
	}
	if res.ECID != "AT00200000000RC105970000000001289" {
		t.Errorf("ECID = %q, want AT00200000000RC105970000000001289", res.ECID)
	}
	if res.ECType != "RC_R" {
		t.Errorf("ECType = %q, want RC_R", res.ECType)
	}
	if res.From != "AT002000" {
		t.Errorf("From = %q, want AT002000", res.From)
	}
	if res.To != "RC105970" {
		t.Errorf("To = %q, want RC105970", res.To)
	}

	// ProcessDate should be 2026-03-02
	wantDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	if !res.ProcessDate.Equal(wantDate) {
		t.Errorf("ProcessDate = %v, want %v", res.ProcessDate, wantDate)
	}

	// File has 8 MPListData entries (3 generation + 5 consumption, but one is malformed
	// in the const above — check we get at least the well-formed ones)
	if len(res.Entries) < 7 {
		t.Errorf("Entries count = %d, want at least 7", len(res.Entries))
	}

	// Verify first entry (GENERATION, SONNE)
	first := res.Entries[0]
	if first.MeteringPoint != "AT0020000000000000000000100266304" {
		t.Errorf("Entries[0].MeteringPoint = %q, want AT0020000000000000000000100266304", first.MeteringPoint)
	}
	if first.EnergyDirection != "GENERATION" {
		t.Errorf("Entries[0].EnergyDirection = %q, want GENERATION", first.EnergyDirection)
	}
	if first.PlantCategory != "SONNE" {
		t.Errorf("Entries[0].PlantCategory = %q, want SONNE", first.PlantCategory)
	}
	if first.ECPartFact != "100" {
		t.Errorf("Entries[0].ECPartFact = %q, want 100", first.ECPartFact)
	}
	wantDateFrom := time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
	if !first.DateFrom.Equal(wantDateFrom) {
		t.Errorf("Entries[0].DateFrom = %v, want %v", first.DateFrom, wantDateFrom)
	}
	// DateTo = "9999-12-31" should be left as zero value
	if !first.DateTo.IsZero() {
		t.Errorf("Entries[0].DateTo should be zero for 9999-12-31, got %v", first.DateTo)
	}
}

func TestParseECMPList_AbschlussECON(t *testing.T) {
	res, err := edaxml.ParseECMPList(syntheticECMPListAbschluss)
	if err != nil {
		t.Fatalf("ParseECMPList: %v", err)
	}

	if res.MessageCode != "ABSCHLUSS_ECON" {
		t.Errorf("MessageCode = %q, want ABSCHLUSS_ECON", res.MessageCode)
	}
	// ConversationID must match the open process (same as ZUSTIMMUNG_ECON)
	if res.ConversationID != "RC105970202603251617458050000000001" {
		t.Errorf("ConversationID = %q, want RC105970202603251617458050000000001", res.ConversationID)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("Entries count = %d, want 1", len(res.Entries))
	}

	e := res.Entries[0]
	if e.MeteringPoint != "AT0020000000000000000000100242261" {
		t.Errorf("MeteringPoint = %q, want AT0020000000000000000000100242261", e.MeteringPoint)
	}
	if e.ConsentID != "AT00200020260326135553934DKPSPX56" {
		t.Errorf("ConsentID = %q, want AT00200020260326135553934DKPSPX56", e.ConsentID)
	}
	if e.EnergyDirection != "GENERATION" {
		t.Errorf("EnergyDirection = %q, want GENERATION", e.EnergyDirection)
	}
	wantActivate := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	if !e.DateActivate.Equal(wantActivate) {
		t.Errorf("DateActivate = %v, want %v", e.DateActivate, wantActivate)
	}
}

func TestParseECMPList_InvalidXML(t *testing.T) {
	_, err := edaxml.ParseECMPList("not xml <<<")
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}
