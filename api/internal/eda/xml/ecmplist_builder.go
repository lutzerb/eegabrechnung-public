package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/edaversion"
)

// ECMPList XML builder for Austrian EDA CustomerProcesses.
//
// Namespace references:
//
//	until 2026-10-04: http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10 (schema 01.10)
//	from  2026-10-05: http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p20 (schema 01.20)
//	CommonTypes:       http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20
//
// See ecmpListSchemaHistory / api/internal/eda/edaversion for the cutover mechanism.
//
// Supported message codes:
//
//	ANFORDERUNG_CPF — Teilnahmefaktoränderung (EC_PRTFACT_CHG)
//
// EC_PRT_CHANGE (ANFORDERUNG_ECC, "Änderung der Aufteilung") is a distinct
// process introduced in the 2026-10 Marktprozesse release and is deliberately
// NOT implemented here — all communities use dynamic allocation (ECDisModel=D),
// for which this process is not needed.

// ecmpListSchema bundles the namespace and SchemaVersion attribute that change
// together at a schema cutover.
type ecmpListSchema struct {
	NS        string
	SchemaVer string
}

// ecmpListSchemaHistory tracks ECMPList schema versions over time — see
// api/internal/eda/edaversion. Append a new Entry for future schema bumps.
var ecmpListSchemaHistory = []edaversion.Entry[ecmpListSchema]{
	{Value: ecmpListSchema{
		NS:        "http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p10",
		SchemaVer: "01.10",
	}},
	{From: edaversion.Cutover2026_10_05, Value: ecmpListSchema{
		NS:        "http://www.ebutilities.at/schemata/customerprocesses/ecmplist/01p20",
		SchemaVer: "01.20",
	}},
}

const (
	ecmpDateLayout     = "2006-01-02"
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

	// DataType (Datentypen der Anfrage, xsd:string max 30) is a required field
	// on ECMPList schema 01.20 (from 2026-10-05). Callers should always set it;
	// it is silently ignored while schema 01.10 is still active.
	DataType string
}

// BuildECMPList builds an outbound ECMPList XML message for Austrian EEG
// processes. The schema version (01.10 or 01.20) is chosen at call time based
// on ecmpListSchemaHistory — see api/internal/eda/edaversion.
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

	now := time.Now()
	schema := edaversion.Resolve(ecmpListSchemaHistory, now)
	post2026Oct := !now.Before(edaversion.Cutover2026_10_05)
	if post2026Oct && p.DataType == "" {
		return "", fmt.Errorf("DataType is required (ECMPList schema 01.20, from 2026-10-05)")
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
	if post2026Oct {
		mpTimeData.DataType = p.DataType
	}

	doc := ecmpListXML{
		NS:    schema.NS,
		NSct:  cpCommonNS,
		NSxsi: xsiNS,
		MarketDir: ecmpMarketDirXML{
			DocumentMode:  "PROD",
			Duplicate:     "false",
			SchemaVersion: schema.SchemaVer,
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
	DateFrom        string `xml:"cp:DateFrom"`
	DateTo          string `xml:"cp:DateTo"`
	EnergyDirection string `xml:"cp:EnergyDirection"`
	ECPartFact      string `xml:"cp:ECPartFact"`
	// DataType is required from ECMPList schema 01.20 (2026-10-05) onward; the
	// element must not appear at all under schema 01.10, hence omitempty.
	DataType       string  `xml:"cp:DataType,omitempty"`
	DateActivate   string  `xml:"cp:DateActivate"`
	DateDeactivate *string `xml:"cp:DateDeactivate,omitempty"`
	ECShare        *string `xml:"cp:ECShare,omitempty"`
}
