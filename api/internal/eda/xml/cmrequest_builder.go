package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// CMRequest XML builder for Austrian EDA CustomerConsent processes.
//
// Namespace references (schema version 01.30):
//
//	CMRequest: http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p30
//	CommonTypes: http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20
//
// Supported message codes:
//
//	ANFORDERUNG_ECON — Online-Anmeldeanforderung (EC_REQ_ONL)
//	ANFORDERUNG_ECOF — Offline-Anmeldeanforderung (EC_REQ_OFF)

const (
	cmRequestNS        = "http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p30"
	cmRequestSchemaVer = "01.30"
	cmDateLayout       = "2006-01-02"
	cmDateTimeLayout   = "2006-01-02T15:04:05"
)

// CMRequestParams holds all parameters for building a CMRequest 01.30 XML message.
type CMRequestParams struct {
	// Routing
	From string // sender ECNumber (EEG's MarktpartnerID)
	To   string // receiver ECNumber (Netzbetreiber)

	// Message identifiers (UUIDs; hyphens are stripped automatically, max 35 chars)
	MessageID      string
	ConversationID string
	CMRequestID    string // separate ID for the consent request element

	// Process data
	MeteringPoint   string    // Zählpunkt (AT...)
	ECID            string    // GemeinschaftID (optional, omit if empty)
	DateFrom        time.Time // start date (required)
	DateTo          time.Time // end date (optional, zero = omit from XML)
	ECPartFact      float64   // participation factor 0-100 (0 = omit)
	ECShare         *float64  // optional EC share, 4 decimal places
	EnergyDirection string    // "CONSUMPTION" or "GENERATION" (empty = omit)
	MessageCode     string    // override MessageCode (default "ANFORDERUNG_ECON"; use "ANFORDERUNG_ECOF" for EC_REQ_OFF)
	Purpose         string    // optional free-text purpose (max 35 chars, EC_REQ_OFF)
}

// BuildCMRequest builds an outbound CMRequest 01.30 XML message for Austrian EEG online/offline registration.
func BuildCMRequest(p CMRequestParams) (string, error) {
	if p.From == "" {
		return "", fmt.Errorf("From (Marktpartner-ID) is required")
	}
	if p.To == "" {
		return "", fmt.Errorf("To (Netzbetreiber-ID) is required")
	}
	if p.MeteringPoint == "" {
		return "", fmt.Errorf("MeteringPoint (Zaehlpunkt) is required")
	}
	if p.MessageID == "" || p.ConversationID == "" {
		return "", fmt.Errorf("MessageID and ConversationID are required")
	}
	if p.CMRequestID == "" {
		return "", fmt.Errorf("CMRequestID is required")
	}
	if p.DateFrom.IsZero() {
		return "", fmt.Errorf("DateFrom is required")
	}

	// Strip hyphens and truncate to 35 chars (ct:GroupingId pattern restriction).
	msgID := strings.ReplaceAll(p.MessageID, "-", "")
	convID := strings.ReplaceAll(p.ConversationID, "-", "")
	cmReqID := strings.ReplaceAll(p.CMRequestID, "-", "")
	if len(msgID) > 35 {
		msgID = msgID[:35]
	}
	if len(convID) > 35 {
		convID = convID[:35]
	}
	if len(cmReqID) > 35 {
		cmReqID = cmReqID[:35]
	}

	now := time.Now()

	// Resolve MessageCode: default to ANFORDERUNG_ECON (online).
	messageCode := p.MessageCode
	if messageCode == "" {
		messageCode = "ANFORDERUNG_ECON"
	}

	// ReqDatType for EC_REQ_ONL and EC_REQ_OFF is always "EnergyCommunityRegistration"
	// per ebutilities.at RequestedDataTypes document (valid since October 2022).
	reqDatType := "EnergyCommunityRegistration"

	// Build the inner CMRequest element.
	inner := cmInnerRequestXML{
		ReqDatType: reqDatType,
		DateFrom:   p.DateFrom.Format(cmDateLayout),
	}

	// DateTo: only include if non-zero.
	if !p.DateTo.IsZero() {
		s := p.DateTo.Format(cmDateLayout)
		inner.DateTo = &s
	}

	// ECID: only include if non-empty.
	if p.ECID != "" {
		inner.ECID = p.ECID
	}

	// ECPartFact: include as integer if non-zero.
	if p.ECPartFact > 0 {
		v := int(p.ECPartFact)
		inner.ECPartFact = &v
	}

	// ECShare: format with 4 decimal places if provided.
	if p.ECShare != nil {
		s := fmt.Sprintf("%.4f", *p.ECShare)
		inner.ECShare = &s
	}

	// EnergyDirection: only include if non-empty.
	if p.EnergyDirection != "" {
		inner.EnergyDirection = p.EnergyDirection
	}

	// Purpose: only include if non-empty (max 100 chars per CMRequest schema 01.30).
	if p.Purpose != "" {
		purpose := p.Purpose
		if len(purpose) > 100 {
			purpose = purpose[:100]
		}
		inner.Purpose = purpose
	}

	doc := cmRequestXML{
		NS:    cmRequestNS,
		NSct:  cpCommonNS,
		NSxsi: xsiNS,
		MarketDir: cmMarketDirXML{
			DocumentMode:  "PROD",
			Duplicate:     "false",
			SchemaVersion: cmRequestSchemaVer,
			RoutingHeader: cpRoutingHeaderXML{
				Sender:                   cpMessageAddressXML{AddressType: "ECNumber", Value: p.From},
				Receiver:                 cpMessageAddressXML{AddressType: "ECNumber", Value: p.To},
				DocumentCreationDateTime: formatDocCreationDateTime(now),
			},
			Sector:      "01",
			MessageCode: messageCode,
		},
		ProcessDir: cmProcessDirXML{
			MessageID:      msgID,
			ConversationID: convID,
			ProcessDate:    now.Format(cmDateLayout),
			MeteringPoint:  p.MeteringPoint,
			CMRequestID:    cmReqID,
			CMRequest:      inner,
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("xml.Marshal: %w", err)
	}
	return xml.Header + string(out), nil
}

// ── XML marshal structs ───────────────────────────────────────────────────────

type cmRequestXML struct {
	XMLName    xml.Name         `xml:"cp:CMRequest"`
	NS         string           `xml:"xmlns:cp,attr"`
	NSct       string           `xml:"xmlns:ct,attr"`
	NSxsi      string           `xml:"xmlns:xsi,attr"`
	MarketDir  cmMarketDirXML   `xml:"cp:MarketParticipantDirectory"`
	ProcessDir cmProcessDirXML  `xml:"cp:ProcessDirectory"`
}

type cmMarketDirXML struct {
	DocumentMode  string             `xml:"DocumentMode,attr"`
	Duplicate     string             `xml:"Duplicate,attr"`
	SchemaVersion string             `xml:"SchemaVersion,attr"`
	RoutingHeader cpRoutingHeaderXML `xml:"ct:RoutingHeader"`
	Sector        string             `xml:"ct:Sector"`
	MessageCode   string             `xml:"cp:MessageCode"`
}

type cmProcessDirXML struct {
	MessageID      string           `xml:"ct:MessageId"`
	ConversationID string           `xml:"ct:ConversationId"`
	ProcessDate    string           `xml:"cp:ProcessDate"`
	MeteringPoint  string           `xml:"cp:MeteringPoint"`
	CMRequestID    string           `xml:"cp:CMRequestId"`
	CMRequest      cmInnerRequestXML `xml:"cp:CMRequest"`
}

type cmInnerRequestXML struct {
	ReqDatType      string  `xml:"cp:ReqDatType"`
	DateFrom        string  `xml:"cp:DateFrom"`
	DateTo          *string `xml:"cp:DateTo,omitempty"`
	ECID            string  `xml:"cp:ECID,omitempty"`
	ECPartFact      *int    `xml:"cp:ECPartFact,omitempty"`
	ECShare         *string `xml:"cp:ECShare,omitempty"`
	EnergyDirection string  `xml:"cp:EnergyDirection,omitempty"`
	Purpose         string  `xml:"cp:Purpose,omitempty"`
}
