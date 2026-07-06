package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// CMRevoke XML builder for Austrian EDA CustomerConsent revocation.
//
// Namespace references (schema version 01.10):
//
//	CMRevoke: http://www.ebutilities.at/schemata/customerconsent/cmrevoke/01p10
//	CommonTypes: http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20
//
// Message code: AUFHEBUNG_CCMS
// Used for CM_REV_SP: the EEG revokes a previously granted customer consent.

const (
	cmRevokeNS        = "http://www.ebutilities.at/schemata/customerconsent/cmrevoke/01p10"
	cmRevokeSchemaVer = "01.10"
)

// CMRevokeParams holds all parameters for building a CMRevoke 01.10 XML message.
type CMRevokeParams struct {
	// Routing
	From string // sender ECNumber (EEG's MarktpartnerID)
	To   string // receiver ECNumber (Netzbetreiber)

	// Message identifiers (UUIDs; hyphens stripped, max 35 chars)
	MessageID      string
	ConversationID string

	// Process data
	MeteringPoint string    // Zählpunkt (AT...)
	ConsentID     string    // ConversationId of the original EC_REQ_ONL process
	ConsentEnd    time.Time // End date of the consent

	// Optional
	ReasonKey int    // 0 = omit; 1-9 = reason code
	Reason    string // free text, max 50 chars
}

// BuildCMRevoke builds an outbound CMRevoke 01.10 XML message for revoking customer consent.
func BuildCMRevoke(p CMRevokeParams) (string, error) {
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
	if p.ConsentID == "" {
		return "", fmt.Errorf("ConsentID is required")
	}
	if p.ConsentEnd.IsZero() {
		return "", fmt.Errorf("ConsentEnd is required")
	}

	// Strip hyphens and truncate to 35 chars (ct:GroupingId pattern restriction).
	msgID := strings.ReplaceAll(p.MessageID, "-", "")
	convID := strings.ReplaceAll(p.ConversationID, "-", "")
	consentID := strings.ReplaceAll(p.ConsentID, "-", "")
	if len(msgID) > 35 {
		msgID = msgID[:35]
	}
	if len(convID) > 35 {
		convID = convID[:35]
	}
	if len(consentID) > 35 {
		consentID = consentID[:35]
	}

	// Truncate reason to 50 chars.
	reason := p.Reason
	if len(reason) > 50 {
		reason = reason[:50]
	}

	// Build optional ReasonKey pointer.
	var reasonKey *int
	if p.ReasonKey > 0 {
		rk := p.ReasonKey
		reasonKey = &rk
	}

	now := time.Now()

	doc := cmRevokeXML{
		NS:    cmRevokeNS,
		NSct:  cpCommonNS,
		NSxsi: xsiNS,
		MarketDir: cmRevokeMarketDirXML{
			DocumentMode:  "PROD",
			Duplicate:     "false",
			SchemaVersion: cmRevokeSchemaVer,
			RoutingHeader: cpRoutingHeaderXML{
				Sender:                   cpMessageAddressXML{AddressType: "ECNumber", Value: p.From},
				Receiver:                 cpMessageAddressXML{AddressType: "ECNumber", Value: p.To},
				DocumentCreationDateTime: formatDocCreationDateTime(now),
			},
			Sector:      "01",
			MessageCode: "AUFHEBUNG_CCMS",
		},
		ProcessDir: cmRevokeProcessDirXML{
			MessageID:      msgID,
			ConversationID: convID,
			ProcessDate:    now.Format(cpDateLayout),
			MeteringPoint:  p.MeteringPoint,
			ConsentID:      consentID,
			ConsentEnd:     p.ConsentEnd.Format(cpDateLayout),
			ReasonKey:      reasonKey,
			Reason:         reason,
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("xml.Marshal: %w", err)
	}
	return xml.Header + string(out), nil
}

// ── XML marshal structs ───────────────────────────────────────────────────────

type cmRevokeXML struct {
	XMLName    xml.Name               `xml:"cp:CMRevoke"`
	NS         string                 `xml:"xmlns:cp,attr"`
	NSct       string                 `xml:"xmlns:ct,attr"`
	NSxsi      string                 `xml:"xmlns:xsi,attr"`
	MarketDir  cmRevokeMarketDirXML   `xml:"cp:MarketParticipantDirectory"`
	ProcessDir cmRevokeProcessDirXML  `xml:"cp:ProcessDirectory"`
}

type cmRevokeMarketDirXML struct {
	DocumentMode  string             `xml:"DocumentMode,attr"`
	Duplicate     string             `xml:"Duplicate,attr"`
	SchemaVersion string             `xml:"SchemaVersion,attr"`
	RoutingHeader cpRoutingHeaderXML `xml:"ct:RoutingHeader"`
	Sector        string             `xml:"ct:Sector"`
	MessageCode   string             `xml:"cp:MessageCode"`
}

type cmRevokeProcessDirXML struct {
	MessageID      string `xml:"ct:MessageId"`
	ConversationID string `xml:"ct:ConversationId"`
	ProcessDate    string `xml:"cp:ProcessDate"`
	MeteringPoint  string `xml:"cp:MeteringPoint"`
	ConsentID      string `xml:"cp:ConsentId"`
	ConsentEnd     string `xml:"cp:ConsentEnd"`
	ReasonKey      *int   `xml:"cp:ReasonKey,omitempty"`
	Reason         string `xml:"cp:Reason,omitempty"`
}
