package xml

import (
	"encoding/xml"
	"strings"
)

// CPNotificationResult holds the parsed content of an inbound CPNotification message.
// CPNotification is the edanet transport-level delivery confirmation.
// ANTWORT_PT (ResponseCode 70) = message received by edanet and forwarded to Netzbetreiber.
type CPNotificationResult struct {
	MessageCode    string // e.g. ANTWORT_PT
	MessageID      string
	ConversationID string
	ResponseCode   string // e.g. "70" = forwarded to Netzbetreiber
	OriginalMsgID  string // MessageId of the message we sent
	From           string // Sender MessageAddress (e.g. "AT002000")
	To             string // Receiver MessageAddress (e.g. "RC105970")
}

// IsCPNotification returns true if the XML payload looks like a CPNotification message.
func IsCPNotification(xmlPayload string) bool {
	return strings.Contains(xmlPayload, "CPNotification")
}

// ParseCPNotification parses an inbound CPNotification XML message.
// Uses local-name matching so namespace prefixes are irrelevant.
func ParseCPNotification(xmlPayload string) CPNotificationResult {
	var doc xmlCPNotifRoot
	if err := xml.Unmarshal([]byte(xmlPayload), &doc); err != nil {
		// Return what we can even on partial parse failure.
		return CPNotificationResult{}
	}
	return CPNotificationResult{
		MessageCode:    doc.MarketDir.MessageCode,
		MessageID:      doc.ProcessDir.MessageID,
		ConversationID: doc.ProcessDir.ConversationID,
		ResponseCode:   doc.ProcessDir.ResponseData.ResponseCode,
		OriginalMsgID:  doc.ProcessDir.ResponseData.OriginalMessageID,
		From:           doc.MarketDir.RoutingHeader.Sender.MessageAddress,
		To:             doc.MarketDir.RoutingHeader.Receiver.MessageAddress,
	}
}

// ── XML unmarshal structs (local-name matching) ───────────────────────────

type xmlCPNotifRoot struct {
	XMLName    xml.Name             `xml:"CPNotification"`
	MarketDir  xmlCPNotifMarketDir  `xml:"MarketParticipantDirectory"`
	ProcessDir xmlCPNotifProcessDir `xml:"ProcessDirectory"`
}

type xmlCPNotifMarketDir struct {
	MessageCode   string              `xml:"MessageCode"`
	RoutingHeader xmlCPNotifRouting   `xml:"RoutingHeader"`
}

type xmlCPNotifRouting struct {
	Sender struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Sender"`
	Receiver struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Receiver"`
}

type xmlCPNotifProcessDir struct {
	MessageID      string              `xml:"MessageId"`
	ConversationID string              `xml:"ConversationId"`
	ResponseData   xmlCPNotifResponse  `xml:"ResponseData"`
}

type xmlCPNotifResponse struct {
	OriginalMessageID string `xml:"OriginalMessageID"`
	ResponseCode      string `xml:"ResponseCode"`
}
