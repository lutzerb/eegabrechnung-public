package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// CPDocumentResult holds the parsed content of an incoming CPDocument message.
// CPDocument messages are confirmation/status messages from the Netzbetreiber,
// such as ERSTE_ANM (first registration confirmation) or FINALE_ANM (final confirmation).
type CPDocumentResult struct {
	DocumentMode   string    // PROD | SIMU
	MessageCode    string    // ERSTE_ANM, FINALE_ANM, ABSCHLUSS_ECON, etc.
	MessageID      string
	ConversationID string    // matches the ConversationID of the outbound request
	Zaehlpunkt     string    // MeteringPoint (AT...)
	GemeinschaftID string    // CommunityID / DeliveryPoint
	From           string
	To             string
	ProcessDate    time.Time
	ValidFrom      time.Time // when the process takes effect (if present)
}

// IsCPDocument returns true if the XML payload looks like a CPDocument message.
func IsCPDocument(xmlPayload string) bool {
	return strings.Contains(xmlPayload, "CPDocument")
}

// ParseCPDocument parses an incoming Austrian EDA CPDocument XML message.
// Uses local-name matching so namespace prefixes in the document are irrelevant.
func ParseCPDocument(xmlPayload string) (*CPDocumentResult, error) {
	var doc xmlCPDocRoot
	if err := xml.Unmarshal([]byte(xmlPayload), &doc); err != nil {
		return nil, fmt.Errorf("xml.Unmarshal: %w", err)
	}

	res := &CPDocumentResult{
		DocumentMode:   doc.MarketDir.DocumentMode,
		MessageCode:    doc.MarketDir.MessageCode,
		MessageID:      doc.ProcessDir.MessageID,
		ConversationID: doc.ProcessDir.ConversationID,
		Zaehlpunkt:     doc.ProcessDir.MeteringPoint,
		GemeinschaftID: doc.ProcessDir.CommunityID,
		From:           doc.MarketDir.RoutingHeader.Sender.MessageAddress,
		To:             doc.MarketDir.RoutingHeader.Receiver.MessageAddress,
	}

	if doc.ProcessDir.ProcessDate != "" {
		if t, err := parseCRDateTime(doc.ProcessDir.ProcessDate); err == nil {
			res.ProcessDate = t.UTC()
		}
	}
	if doc.ProcessDir.ValidFrom != "" {
		if t, err := parseCRDateTime(doc.ProcessDir.ValidFrom); err == nil {
			res.ValidFrom = t.UTC()
		}
	}

	return res, nil
}

// ── XML unmarshal structs (local-name matching) ─────────────────────────

type xmlCPDocRoot struct {
	XMLName   xml.Name          `xml:"CPDocument"`
	MarketDir xmlCPDocMarketDir `xml:"MarketParticipantDirectory"`
	ProcessDir xmlCPDocProcessDir `xml:"ProcessDirectory"`
}

type xmlCPDocMarketDir struct {
	DocumentMode  string             `xml:"DocumentMode,attr"`
	MessageCode   string             `xml:"MessageCode"`
	RoutingHeader xmlCPDocRouting    `xml:"RoutingHeader"`
}

type xmlCPDocRouting struct {
	Sender struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Sender"`
	Receiver struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Receiver"`
}

type xmlCPDocProcessDir struct {
	MessageID      string `xml:"MessageId"`
	ConversationID string `xml:"ConversationId"`
	ProcessDate    string `xml:"ProcessDate"`
	MeteringPoint  string `xml:"MeteringPoint"`
	CommunityID    string `xml:"CommunityID"`
	ValidFrom      string `xml:"ValidFrom"`
}
