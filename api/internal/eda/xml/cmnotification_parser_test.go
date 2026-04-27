package xml_test

import (
	"testing"

	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// realCMNotificationZustimmung is the literal XML from testdata/7bbb0769-ffe4-420c-8f9f-b0f7c4668424.xml
// (CMNotification/ZUSTIMMUNG_ECON received from Netzbetreiber AT002000)
const realCMNotificationZustimmung = `<?xml version="1.0" encoding="UTF-8"?><ns0:CMNotification xmlns:ns0="http://www.ebutilities.at/schemata/customerconsent/cmnotification/01p12" xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"><ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12"><ns1:RoutingHeader><ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender><ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver><ns1:DocumentCreationDateTime>2026-03-26T12:55:57.4017420Z</ns1:DocumentCreationDateTime></ns1:RoutingHeader><ns1:Sector>01</ns1:Sector><ns0:MessageCode>ZUSTIMMUNG_ECON</ns0:MessageCode></ns0:MarketParticipantDirectory><ns0:ProcessDirectory><ns1:MessageId>AT002000202603251717485477751697266</ns1:MessageId><ns1:ConversationId>RC105970202603251617458050000000001</ns1:ConversationId><ns0:CMRequestId>DKPSPX56</ns0:CMRequestId><ns0:ResponseData><ns0:ConsentId>AT00200020260326135553934DKPSPX56</ns0:ConsentId><ns0:MeteringPoint>AT0020000000000000000000100242261</ns0:MeteringPoint><ns0:ResponseCode>175</ns0:ResponseCode></ns0:ResponseData></ns0:ProcessDirectory></ns0:CMNotification>`

const syntheticCMNotificationAblehnung = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CMNotification xmlns:ns0="http://www.ebutilities.at/schemata/customerconsent/cmnotification/01p12"
                    xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-04-01T10:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ABLEHNUNG_ECON</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202604011000000000000000001</ns1:MessageId>
    <ns1:ConversationId>RC105970202604011000000000000000001</ns1:ConversationId>
    <ns0:CMRequestId>TESTABCD</ns0:CMRequestId>
    <ns0:ResponseData>
      <ns0:ConsentId>AT00200020260401100000000TESTABCD</ns0:ConsentId>
      <ns0:MeteringPoint>AT0020000000000000000000100999999</ns0:MeteringPoint>
      <ns0:ResponseCode>176</ns0:ResponseCode>
    </ns0:ResponseData>
  </ns0:ProcessDirectory>
</ns0:CMNotification>`

const syntheticCMNotificationAntwort = `<?xml version="1.0" encoding="UTF-8"?>
<ns0:CMNotification xmlns:ns0="http://www.ebutilities.at/schemata/customerconsent/cmnotification/01p12"
                    xmlns:ns1="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20">
  <ns0:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.12">
    <ns1:RoutingHeader>
      <ns1:Sender AddressType="ECNumber"><ns1:MessageAddress>AT002000</ns1:MessageAddress></ns1:Sender>
      <ns1:Receiver AddressType="ECNumber"><ns1:MessageAddress>RC105970</ns1:MessageAddress></ns1:Receiver>
      <ns1:DocumentCreationDateTime>2026-04-01T09:00:00Z</ns1:DocumentCreationDateTime>
    </ns1:RoutingHeader>
    <ns1:Sector>01</ns1:Sector>
    <ns0:MessageCode>ANTWORT_ECON</ns0:MessageCode>
  </ns0:MarketParticipantDirectory>
  <ns0:ProcessDirectory>
    <ns1:MessageId>AT002000202604010900000000000000001</ns1:MessageId>
    <ns1:ConversationId>RC105970202604010900000000000000001</ns1:ConversationId>
    <ns0:CMRequestId>ANTWEF01</ns0:CMRequestId>
    <ns0:ResponseData>
      <ns0:ConsentId></ns0:ConsentId>
      <ns0:MeteringPoint>AT0020000000000000000000100999999</ns0:MeteringPoint>
      <ns0:ResponseCode>100</ns0:ResponseCode>
    </ns0:ResponseData>
  </ns0:ProcessDirectory>
</ns0:CMNotification>`

func TestIsCMNotification(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{"real ZUSTIMMUNG_ECON", realCMNotificationZustimmung, true},
		{"synthetic ABLEHNUNG_ECON", syntheticCMNotificationAblehnung, true},
		{"CPDocument XML", `<CPDocument><MarketParticipantDirectory/></CPDocument>`, false},
		{"ECMPList XML", `<ns0:ECMPList xmlns:ns0="..."><something/></ns0:ECMPList>`, false},
		{"ConsumptionRecord XML", `<ConsumptionRecord><ProcessDirectory/></ConsumptionRecord>`, false},
		{"empty string", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := edaxml.IsCMNotification(tc.payload)
			if got != tc.want {
				t.Errorf("IsCMNotification() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseCMNotification_Zustimmung(t *testing.T) {
	res, err := edaxml.ParseCMNotification(realCMNotificationZustimmung)
	if err != nil {
		t.Fatalf("ParseCMNotification: %v", err)
	}

	if res.DocumentMode != "PROD" {
		t.Errorf("DocumentMode = %q, want PROD", res.DocumentMode)
	}
	if res.MessageCode != "ZUSTIMMUNG_ECON" {
		t.Errorf("MessageCode = %q, want ZUSTIMMUNG_ECON", res.MessageCode)
	}
	if res.MessageID != "AT002000202603251717485477751697266" {
		t.Errorf("MessageID = %q, want AT002000202603251717485477751697266", res.MessageID)
	}
	if res.ConversationID != "RC105970202603251617458050000000001" {
		t.Errorf("ConversationID = %q, want RC105970202603251617458050000000001", res.ConversationID)
	}
	if res.CMRequestID != "DKPSPX56" {
		t.Errorf("CMRequestID = %q, want DKPSPX56", res.CMRequestID)
	}
	if res.ConsentID != "AT00200020260326135553934DKPSPX56" {
		t.Errorf("ConsentID = %q, want AT00200020260326135553934DKPSPX56", res.ConsentID)
	}
	if res.MeteringPoint != "AT0020000000000000000000100242261" {
		t.Errorf("MeteringPoint = %q, want AT0020000000000000000000100242261", res.MeteringPoint)
	}
	if res.ResponseCode != "175" {
		t.Errorf("ResponseCode = %q, want 175", res.ResponseCode)
	}
	if res.From != "AT002000" {
		t.Errorf("From = %q, want AT002000", res.From)
	}
	if res.To != "RC105970" {
		t.Errorf("To = %q, want RC105970", res.To)
	}
}

func TestParseCMNotification_Ablehnung(t *testing.T) {
	res, err := edaxml.ParseCMNotification(syntheticCMNotificationAblehnung)
	if err != nil {
		t.Fatalf("ParseCMNotification: %v", err)
	}

	if res.MessageCode != "ABLEHNUNG_ECON" {
		t.Errorf("MessageCode = %q, want ABLEHNUNG_ECON", res.MessageCode)
	}
	if res.ConversationID != "RC105970202604011000000000000000001" {
		t.Errorf("ConversationID = %q, want RC105970202604011000000000000000001", res.ConversationID)
	}
	if res.CMRequestID != "TESTABCD" {
		t.Errorf("CMRequestID = %q, want TESTABCD", res.CMRequestID)
	}
	if res.ResponseCode != "176" {
		t.Errorf("ResponseCode = %q, want 176", res.ResponseCode)
	}
	if res.MeteringPoint != "AT0020000000000000000000100999999" {
		t.Errorf("MeteringPoint = %q, want AT0020000000000000000000100999999", res.MeteringPoint)
	}
}

func TestParseCMNotification_Antwort(t *testing.T) {
	res, err := edaxml.ParseCMNotification(syntheticCMNotificationAntwort)
	if err != nil {
		t.Fatalf("ParseCMNotification: %v", err)
	}

	if res.MessageCode != "ANTWORT_ECON" {
		t.Errorf("MessageCode = %q, want ANTWORT_ECON", res.MessageCode)
	}
	if res.ConversationID == "" {
		t.Error("ConversationID should not be empty")
	}
	if res.CMRequestID != "ANTWEF01" {
		t.Errorf("CMRequestID = %q, want ANTWEF01", res.CMRequestID)
	}
}

func TestParseCMNotification_InvalidXML(t *testing.T) {
	_, err := edaxml.ParseCMNotification("not xml <<<")
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}
