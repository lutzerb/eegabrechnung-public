package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// ECMPList XML builder for Austrian EDA CustomerProcesses.
//
// Namespace references (schema version 01.10):
//
//	ECMPList:    http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10
//	CommonTypes: http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20
//
// Supported message codes:
//
//	ANFORDERUNG_CPF — Teilnahmefaktoränderung (EC_PRTFACT_CHG)
//	ANFORDERUNG_ECC — Einzelabmeldung (EC_EINZEL_ABM)

const (
	ecmpListNS        = "http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10"
	ecmpListSchemaVer = "01.10"
	ecmpDateLayout    = "2006-01-02"
	ecmpDateTimeLayout = "2006-01-02T15:04:05"
)

// ECMPListParams holds all parameters for building an ECMPList XML message.
type ECMPListParams struct {
	// Routing
	From string // sender MarktpartnerID (own EC number)
	To   string // receiver Netzbetreiber-ID

	// Message identifiers (UUIDs; hyphens are stripped automatically, max 35 chars)
	MessageID      string
	ConversationID string

	// Community / process
	ECID        string // Gemeinschafts-ID (not the meter point!)
	ECType      string // GC, RC_R, RC_L, CC
	ECDisModel  string // S = statisch, D = dynamisch
	MessageCode string // ANFORDERUNG_CPF or ANFORDERUNG_ECC

	// Meter point data
	MeteringPoint  string    // Zählpunkt (AT...)
	DateFrom       time.Time // valid from
	DateTo         time.Time // valid until (zero → "9999-12-31")
	DateActivate   time.Time // Versorgt seit (zero → DateFrom)
	DateDeactivate *time.Time // optional, only for Abmeldung

	EnergyDirection string   // CONSUMPTION or GENERATION
	ECPartFact      float64  // participation factor 0..100 (formatted as integer)
	ECShare         *float64 // optional share % with 4 decimal places
}

// BuildECMPList builds an outbound ECMPList 01.10 XML message for Austrian EEG processes.
func BuildECMPList(p ECMPListParams) (string, error) {
	if p.From == "" {
		return "", fmt.Errorf("From (Marktpartner-ID) is required")
	}
	if p.To == "" {
		return "", fmt.Errorf("To (Netzbetreiber-ID) is required")
	}
	if p.ECID == "" {
		return "", fmt.Errorf("ECID (Gemeinschafts-ID) is required")
	}
	if p.MessageCode == "" {
		return "", fmt.Errorf("MessageCode is required")
	}
	if p.MeteringPoint == "" {
		return "", fmt.Errorf("MeteringPoint (Zaehlpunkt) is required")
	}
	if p.MessageID == "" || p.ConversationID == "" {
		return "", fmt.Errorf("MessageID and ConversationID are required")
	}

	// Strip hyphens and truncate to 35 chars (ct:GroupingId pattern restriction).
	msgID := strings.ReplaceAll(p.MessageID, "-", "")
	convID := strings.ReplaceAll(p.ConversationID, "-", "")
	if len(msgID) > 35 {
		msgID = msgID[:35]
	}
	if len(convID) > 35 {
		convID = convID[:35]
	}

	now := time.Now()

	// DateTo: zero → 9999-12-31
	dateTo := "9999-12-31"
	if !p.DateTo.IsZero() {
		dateTo = p.DateTo.Format(ecmpDateLayout)
	}

	// DateActivate: zero → DateFrom
	dateActivate := p.DateFrom.Format(ecmpDateLayout)
	if !p.DateActivate.IsZero() {
		dateActivate = p.DateActivate.Format(ecmpDateLayout)
	}

	// ECPartFact as integer string
	ecPartFact := fmt.Sprintf("%d", int(p.ECPartFact))

	// ECShare with 4 decimal places (optional)
	var ecShare *string
	if p.ECShare != nil {
		s := fmt.Sprintf("%.4f", *p.ECShare)
		ecShare = &s
	}

	// DateDeactivate (optional, only for Abmeldung)
	var dateDeactivate *string
	if p.DateDeactivate != nil {
		s := p.DateDeactivate.Format(ecmpDateLayout)
		dateDeactivate = &s
	}

	mpTimeData := ecmpMPTimeDataXML{
		DateFrom:        p.DateFrom.Format(ecmpDateLayout),
		DateTo:          dateTo,
		EnergyDirection: p.EnergyDirection,
		ECPartFact:      ecPartFact,
		DateActivate:    dateActivate,
		DateDeactivate:  dateDeactivate,
		ECShare:         ecShare,
	}

	doc := ecmpListXML{
		NS:    ecmpListNS,
		NSct:  cpCommonNS,
		NSxsi: xsiNS,
		MarketDir: ecmpMarketDirXML{
			DocumentMode:  "PROD",
			Duplicate:     "false",
			SchemaVersion: ecmpListSchemaVer,
			RoutingHeader: cpRoutingHeaderXML{
				Sender:                   cpMessageAddressXML{AddressType: "ECNumber", Value: p.From},
				Receiver:                 cpMessageAddressXML{AddressType: "ECNumber", Value: p.To},
				DocumentCreationDateTime: formatDocCreationDateTime(now),
			},
			Sector:      "01",
			MessageCode: p.MessageCode,
		},
		ProcessDir: ecmpProcessDirXML{
			MessageID:      msgID,
			ConversationID: convID,
			ProcessDate:    now.Format(ecmpDateLayout),
			ECID:           p.ECID,
			ECType:         p.ECType,
			ECDisModel:     p.ECDisModel,
			MPListData: ecmpMPListDataXML{
				MeteringPoint: p.MeteringPoint,
				MPTimeData:    mpTimeData,
			},
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("xml.Marshal: %w", err)
	}
	return xml.Header + string(out), nil
}

// ── XML marshal structs ───────────────────────────────────────────────────────

type ecmpListXML struct {
	XMLName    xml.Name          `xml:"cp:ECMPList"`
	NS         string            `xml:"xmlns:cp,attr"`
	NSct       string            `xml:"xmlns:ct,attr"`
	NSxsi      string            `xml:"xmlns:xsi,attr"`
	MarketDir  ecmpMarketDirXML  `xml:"cp:MarketParticipantDirectory"`
	ProcessDir ecmpProcessDirXML `xml:"cp:ProcessDirectory"`
}

type ecmpMarketDirXML struct {
	DocumentMode  string             `xml:"DocumentMode,attr"`
	Duplicate     string             `xml:"Duplicate,attr"`
	SchemaVersion string             `xml:"SchemaVersion,attr"`
	RoutingHeader cpRoutingHeaderXML `xml:"ct:RoutingHeader"`
	Sector        string             `xml:"ct:Sector"`
	MessageCode   string             `xml:"cp:MessageCode"`
}

type ecmpProcessDirXML struct {
	MessageID      string            `xml:"cp:MessageId"`
	ConversationID string            `xml:"cp:ConversationId"`
	ProcessDate    string            `xml:"cp:ProcessDate"`
	ECID           string            `xml:"cp:ECID"`
	ECType         string            `xml:"cp:ECType"`
	ECDisModel     string            `xml:"cp:ECDisModel"`
	MPListData     ecmpMPListDataXML `xml:"cp:MPListData"`
}

type ecmpMPListDataXML struct {
	MeteringPoint string            `xml:"cp:MeteringPoint"`
	MPTimeData    ecmpMPTimeDataXML `xml:"cp:MPTimeData"`
}

type ecmpMPTimeDataXML struct {
	DateFrom        string  `xml:"cp:DateFrom"`
	DateTo          string  `xml:"cp:DateTo"`
	EnergyDirection string  `xml:"cp:EnergyDirection"`
	ECPartFact      string  `xml:"cp:ECPartFact"`
	DateActivate    string  `xml:"cp:DateActivate"`
	DateDeactivate  *string `xml:"cp:DateDeactivate,omitempty"`
	ECShare         *string `xml:"cp:ECShare,omitempty"`
}
