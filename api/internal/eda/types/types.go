// Package types defines shared EDA types used across the eda package hierarchy.
package types

import (
	"context"
	"time"
)

// Message represents a MaKo XML message exchanged via the EDA protocol.
type Message struct {
	// ID is a unique identifier for this message (UUID or message-id header).
	ID string
	// Process is the EDA process code, e.g. ANFORDERUNG_ECON or DATEN_CRMSG.
	Process string
	// Direction is either "inbound" or "outbound".
	Direction string
	// From is the sender address or meter point identifier.
	From string
	// To is the recipient address.
	To string
	// GemeinschaftID is the energy community identifier.
	GemeinschaftID string
	// Subject is the email subject line (populated for MAIL transport inbound messages).
	Subject string
	// EmailBody is the plain-text body of the email (MAIL transport only, without the XML attachment).
	EmailBody string
	// XMLPayload is the raw XML body.
	XMLPayload string
	// CreatedAt is when the message was created/received.
	CreatedAt time.Time
}

// Process code for ConsumptionRecord messages (energy data delivery).
const ProcessCRMsg = "DATEN_CRMSG"

// ProcessParseError is a synthetic process type returned by transports when an
// inbound message cannot be parsed. The worker stores these in eda_errors.
const ProcessParseError = "PARSE_ERROR"

// Direction constants.
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// Status constants for eda_messages table.
const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusAck     = "ack"
	StatusError   = "error"
)

// Transport abstracts different EDA transport backends (MAIL, PONTON, FILE).
type Transport interface {
	// Send sends an outbound MaKo XML message.
	Send(ctx context.Context, msg *Message) error
	// Poll checks for new inbound messages and returns them.
	Poll(ctx context.Context) ([]*Message, error)
	// SendAck sends an ebMS 2.0 acknowledgement for the given inbound message.
	// originalMsgID is the Message-ID of the received message.
	// from / to are the sender and recipient EDA addresses.
	SendAck(ctx context.Context, originalMsgID, from, to string) error
}
