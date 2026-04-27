package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// CMRevokeDoc holds the parsed fields from an inbound CMRevoke message.
// MessageCode distinguishes the two inbound variants:
//   - "AUFHEBUNG_CCMS"     → CM_REV_CUS: customer revoked their own consent
//   - "AUFHEBUNG_CCMS_IMP" → CM_REV_IMP: Netzbetreiber revokes due to impossibility
type CMRevokeDoc struct {
	MessageCode    string // "AUFHEBUNG_CCMS" or "AUFHEBUNG_CCMS_IMP"
	From           string // Sender Marktpartner-ID (Netzbetreiber)
	To             string // Receiver Marktpartner-ID (EEG)
	ConversationID string // ct:ConversationId
	MeteringPoint  string // cp:MeteringPoint
	ConsentID      string // cp:ConsentId
	ConsentEnd     string // cp:ConsentEnd (date string YYYY-MM-DD)
}

// IsCMRevoke returns true if xmlStr looks like a CMRevoke document.
func IsCMRevoke(xmlStr string) bool {
	return strings.Contains(xmlStr, "cmrevoke") ||
		strings.Contains(xmlStr, "AUFHEBUNG_CCMS") ||
		strings.Contains(xmlStr, "CMRevoke")
}

// ParseCMRevoke parses an inbound CMRevoke XML (CM_REV_CUS or CM_REV_IMP from Netzbetreiber).
func ParseCMRevoke(xmlStr string) (*CMRevokeDoc, error) {
	var doc xmlCMRevokeRoot
	if err := xml.Unmarshal([]byte(xmlStr), &doc); err != nil {
		return nil, fmt.Errorf("xml.Unmarshal CMRevoke: %w", err)
	}
	return &CMRevokeDoc{
		MessageCode:    doc.MarketDir.MessageCode,
		From:           doc.MarketDir.RoutingHeader.Sender.MessageAddress,
		To:             doc.MarketDir.RoutingHeader.Receiver.MessageAddress,
		ConversationID: doc.ProcessDir.ConversationID,
		MeteringPoint:  doc.ProcessDir.MeteringPoint,
		ConsentID:      doc.ProcessDir.ConsentID,
		ConsentEnd:     doc.ProcessDir.ConsentEnd,
	}, nil
}

// ── XML unmarshal structs (local-name matching) ──────────────────────────────

type xmlCMRevokeRoot struct {
	XMLName    xml.Name              `xml:"CMRevoke"`
	MarketDir  xmlCMRevokeMarketDir  `xml:"MarketParticipantDirectory"`
	ProcessDir xmlCMRevokeProcessDir `xml:"ProcessDirectory"`
}

type xmlCMRevokeMarketDir struct {
	DocumentMode  string                `xml:"DocumentMode,attr"`
	MessageCode   string                `xml:"MessageCode"`
	RoutingHeader xmlCMRevokeRouting    `xml:"RoutingHeader"`
}

type xmlCMRevokeRouting struct {
	Sender   struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Sender"`
	Receiver struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Receiver"`
}

type xmlCMRevokeProcessDir struct {
	MessageID      string `xml:"MessageId"`
	ConversationID string `xml:"ConversationId"`
	MeteringPoint  string `xml:"MeteringPoint"`
	ConsentID      string `xml:"ConsentId"`
	ConsentEnd     string `xml:"ConsentEnd"`
	ReasonKey      int    `xml:"ReasonKey"`
	Reason         string `xml:"Reason"`
}
