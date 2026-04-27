package xml_test

import (
	"strings"
	"testing"
	"time"

	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// realCPRequestAnforderungECP is from testdata/a3fe8f39-2221-4d2b-8b10-1c2c2a7f9436.xml
// (CPRequest/ANFORDERUNG_ECP, outbound from EEG to Netzbetreiber — requests meter point list)
const realCPRequestAnforderungECP = `<cp:CPRequest xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cp="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12" xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20" xsi:schemaLocation="http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12 http://www.ebutilities.at/schemata/customerprocesses/cprequest/01p12/CPRequest_01p12.xsd">
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
</cp:CPRequest>`

func buildCPReqParams(process string) edaxml.CPRequestParams {
	return edaxml.CPRequestParams{
		From:           "RC105970",
		To:             "AT002000",
		GemeinschaftID: "AT00200000000RC105970000000001289",
		Process:        process,
		MessageID:      "RC105970202604010000000000000000001",
		ConversationID: "RC105970202604010000000000000000001",
		Zaehlpunkt:     "AT0020000000000000000000100242261",
		ValidFrom:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestBuildCPRequest_EinzelAnm(t *testing.T) {
	p := buildCPReqParams("EC_EINZEL_ANM")
	p.ShareType = "RC_R"
	p.ParticipationFactor = 100

	xmlStr, err := edaxml.BuildCPRequest(p)
	if err != nil {
		t.Fatalf("BuildCPRequest: %v", err)
	}

	if !strings.Contains(xmlStr, "<?xml") {
		t.Error("missing XML header")
	}
	if !strings.Contains(xmlStr, "EC_EINZEL_ANM") {
		t.Error("missing MessageCode EC_EINZEL_ANM")
	}
	if !strings.Contains(xmlStr, "RC105970") {
		t.Error("missing sender RC105970")
	}
	if !strings.Contains(xmlStr, "AT002000") {
		t.Error("missing receiver AT002000")
	}
	if !strings.Contains(xmlStr, "AT0020000000000000000000100242261") {
		t.Error("missing Zaehlpunkt")
	}
	if !strings.Contains(xmlStr, "AT00200000000RC105970000000001289") {
		t.Error("missing GemeinschaftID")
	}
	if !strings.Contains(xmlStr, "RC_R") {
		t.Error("missing ShareType RC_R")
	}
	if !strings.Contains(xmlStr, "100") {
		t.Error("missing ParticipationFactor 100")
	}
	if !strings.Contains(xmlStr, "2026-04-01") {
		t.Error("missing ValidFrom 2026-04-01")
	}
}

func TestBuildCPRequest_EinzelAbm(t *testing.T) {
	p := buildCPReqParams("EC_EINZEL_ABM")

	xmlStr, err := edaxml.BuildCPRequest(p)
	if err != nil {
		t.Fatalf("BuildCPRequest: %v", err)
	}

	if !strings.Contains(xmlStr, "EC_EINZEL_ABM") {
		t.Error("missing MessageCode EC_EINZEL_ABM")
	}
	if !strings.Contains(xmlStr, "AT0020000000000000000000100242261") {
		t.Error("missing Zaehlpunkt")
	}
	if !strings.Contains(xmlStr, "2026-04-01") {
		t.Error("missing ValidFrom 2026-04-01")
	}
	// ShareType must not appear in Abmeldung
	if strings.Contains(xmlStr, "ShareType") {
		t.Error("ShareType should not appear in EC_EINZEL_ABM")
	}
}

func TestBuildCPRequest_PrtfactChg(t *testing.T) {
	p := buildCPReqParams("EC_PRTFACT_CHG")
	p.ParticipationFactor = 75

	xmlStr, err := edaxml.BuildCPRequest(p)
	if err != nil {
		t.Fatalf("BuildCPRequest: %v", err)
	}

	if !strings.Contains(xmlStr, "EC_PRTFACT_CHG") {
		t.Error("missing MessageCode EC_PRTFACT_CHG")
	}
	if !strings.Contains(xmlStr, "75") {
		t.Error("missing ParticipationFactor 75")
	}
	if !strings.Contains(xmlStr, "2026-04-01") {
		t.Error("missing ValidFrom")
	}
}

func TestBuildCPRequest_ReqPT(t *testing.T) {
	p := buildCPReqParams("EC_REQ_PT")
	p.DateFrom = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.DateTo = time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	xmlStr, err := edaxml.BuildCPRequest(p)
	if err != nil {
		t.Fatalf("BuildCPRequest: %v", err)
	}

	if !strings.Contains(xmlStr, "EC_REQ_PT") {
		t.Error("missing MessageCode EC_REQ_PT")
	}
	if !strings.Contains(xmlStr, "2026-01-01") {
		t.Error("missing DateFrom")
	}
	if !strings.Contains(xmlStr, "2026-03-31") {
		t.Error("missing DateTo")
	}
}

func TestBuildCPRequest_ReqONL(t *testing.T) {
	p := buildCPReqParams("EC_REQ_ONL")
	p.ShareType = "RC_R"
	p.ParticipationFactor = 100

	xmlStr, err := edaxml.BuildCPRequest(p)
	if err != nil {
		t.Fatalf("BuildCPRequest EC_REQ_ONL: %v", err)
	}

	if !strings.Contains(xmlStr, "EC_REQ_ONL") {
		t.Error("missing MessageCode EC_REQ_ONL")
	}
}

func TestBuildCPRequest_MissingFields(t *testing.T) {
	base := buildCPReqParams("EC_EINZEL_ANM")

	tests := []struct {
		name   string
		mutate func(*edaxml.CPRequestParams)
	}{
		{"missing From", func(p *edaxml.CPRequestParams) { p.From = "" }},
		{"missing To", func(p *edaxml.CPRequestParams) { p.To = "" }},
		{"missing GemeinschaftID", func(p *edaxml.CPRequestParams) { p.GemeinschaftID = "" }},
		{"missing Process", func(p *edaxml.CPRequestParams) { p.Process = "" }},
		{"missing Zaehlpunkt", func(p *edaxml.CPRequestParams) { p.Zaehlpunkt = "" }},
		{"missing MessageID", func(p *edaxml.CPRequestParams) { p.MessageID = "" }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			_, err := edaxml.BuildCPRequest(p)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestCPRequest_DetectionFunctions(t *testing.T) {
	// The ANFORDERUNG_ECP from testdata should NOT be detected as inbound EDA messages
	// (it's an outbound CPRequest, not a CPDocument/CMNotification/ECMPList)
	if edaxml.IsCPDocument(realCPRequestAnforderungECP) {
		t.Error("CPRequest should not be detected as CPDocument")
	}
	if edaxml.IsCMNotification(realCPRequestAnforderungECP) {
		t.Error("CPRequest should not be detected as CMNotification")
	}
	if edaxml.IsECMPList(realCPRequestAnforderungECP) {
		t.Error("CPRequest should not be detected as ECMPList")
	}
}
