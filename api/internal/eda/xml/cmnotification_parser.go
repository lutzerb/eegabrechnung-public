package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// CMNotificationResult holds the parsed content of an incoming CMNotification message.
// CMNotification messages are sent by the Netzbetreiber to report consent decisions,
// e.g. ZUSTIMMUNG_ECON (consent granted, code 175) or ABLEHNUNG_ECON (rejected).
type CMNotificationResult struct {
	DocumentMode   string
	MessageCode    string   // ZUSTIMMUNG_ECON | ABLEHNUNG_ECON | ANTWORT_ECON
	MessageID      string
	ConversationID string
	CMRequestID    string
	ConsentID      string
	MeteringPoint  string
	ResponseCodes  []string // all ResponseCode elements, e.g. ["175"] or ["182","99"]
	// CustomerData fields (only in ZUSTIMMUNG_ECON):
	// Name1 = Nachname or company name; Name2 = Vorname (absent for legal entities)
	CustomerName1     string // e.g. "Puncochar" or "Business Data Solutions G.m.b.H."
	CustomerName2     string // e.g. "Michael" (empty for companies)
	PortalApprovalURL string // NB portal URL for manual consent (present when code 182)
	From              string
	To                string
}

// IsCMNotification returns true if the XML payload looks like a CMNotification message.
func IsCMNotification(xmlPayload string) bool {
	return strings.Contains(xmlPayload, "CMNotification")
}

// ParseCMNotification parses an incoming Austrian EDA CMNotification XML message.
// Uses local-name matching so namespace prefixes in the document are irrelevant.
func ParseCMNotification(xmlPayload string) (*CMNotificationResult, error) {
	var doc xmlCMNotifRoot
	if err := xml.Unmarshal([]byte(xmlPayload), &doc); err != nil {
		return nil, fmt.Errorf("xml.Unmarshal: %w", err)
	}

	return &CMNotificationResult{
		DocumentMode:      doc.MarketDir.DocumentMode,
		MessageCode:       doc.MarketDir.MessageCode,
		MessageID:         doc.ProcessDir.MessageID,
		ConversationID:    doc.ProcessDir.ConversationID,
		CMRequestID:       doc.ProcessDir.CMRequestID,
		ConsentID:         doc.ProcessDir.ResponseData.ConsentID,
		MeteringPoint:     doc.ProcessDir.ResponseData.MeteringPoint,
		ResponseCodes:     doc.ProcessDir.ResponseData.ResponseCodes,
		CustomerName1:     doc.ProcessDir.CustomerData.Name1,
		CustomerName2:     doc.ProcessDir.CustomerData.Name2,
		PortalApprovalURL: doc.ProcessDir.PortalApprovalURL,
		From:              doc.MarketDir.RoutingHeader.Sender.MessageAddress,
		To:                doc.MarketDir.RoutingHeader.Receiver.MessageAddress,
	}, nil
}

// ── XML unmarshal structs (local-name matching) ─────────────────────────

type xmlCMNotifRoot struct {
	XMLName    xml.Name             `xml:"CMNotification"`
	MarketDir  xmlCMNotifMarketDir  `xml:"MarketParticipantDirectory"`
	ProcessDir xmlCMNotifProcessDir `xml:"ProcessDirectory"`
}

type xmlCMNotifMarketDir struct {
	DocumentMode  string            `xml:"DocumentMode,attr"`
	MessageCode   string            `xml:"MessageCode"`
	RoutingHeader xmlCMNotifRouting `xml:"RoutingHeader"`
}

type xmlCMNotifRouting struct {
	Sender struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Sender"`
	Receiver struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Receiver"`
}

type xmlCMNotifProcessDir struct {
	MessageID         string             `xml:"MessageId"`
	ConversationID    string             `xml:"ConversationId"`
	CMRequestID       string             `xml:"CMRequestId"`
	ResponseData      xmlCMNotifResponse `xml:"ResponseData"`
	CustomerData      xmlCMNotifCustomer `xml:"CustomerData"`
	PortalApprovalURL string             `xml:"PortalApprovalURL"`
}

type xmlCMNotifResponse struct {
	ConsentID     string   `xml:"ConsentId"`
	MeteringPoint string   `xml:"MeteringPoint"`
	ResponseCodes []string `xml:"ResponseCode"`
}

type xmlCMNotifCustomer struct {
	Name1 string `xml:"Name1"`
	Name2 string `xml:"Name2"`
}
