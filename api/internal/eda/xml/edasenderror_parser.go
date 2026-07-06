package xml

import (
	"encoding/xml"
	"strings"
)

// EDASendError is returned by the edanet gateway when our outbound message fails validation.
type EDASendError struct {
	MailSubject string `xml:"MailSubject"`
	ReasonText  string `xml:"ReasonText"`
	ReceivedAt  string `xml:"ReceivedAt"`
}

// IsEDASendError returns true when the XML string is an EDASendError gateway error.
func IsEDASendError(xmlStr string) bool {
	return strings.Contains(xmlStr, "<EDASendError>") || strings.Contains(xmlStr, "<EDASendError ")
}

// ParseEDASendError parses an EDASendError XML document.
func ParseEDASendError(xmlStr string) (*EDASendError, error) {
	var e EDASendError
	if err := xml.Unmarshal([]byte(xmlStr), &e); err != nil {
		return nil, err
	}
	return &e, nil
}
