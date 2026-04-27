package xml

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/types"
)

// parsedAnforderung is used for XML unmarshalling.
type parsedAnforderung struct {
	XMLName         xml.Name                `xml:"Anforderung"`
	Marktteilnehmer parsedMarktteilnehmer   `xml:"Marktteilnehmer"`
	Gemeinschaft    parsedGemeinschaft      `xml:"Gemeinschaft"`
}

type parsedMarktteilnehmer struct {
	Absender   string `xml:"Absender"`
	Empfaenger string `xml:"Empfaenger"`
}

type parsedGemeinschaft struct {
	GemeinschaftsID string `xml:"GemeinschaftsID"`
	Prozess         string `xml:"Prozess"`
	Version         string `xml:"Version"`
	ZeitpunktVon    string `xml:"ZeitpunktVon"`
	ZeitpunktBis    string `xml:"ZeitpunktBis"`
}

// ParseAnforderung parses a MaKo XML message body and returns a types.Message.
// The ID field must be set by the caller (e.g. from the message-id header).
func ParseAnforderung(xmlPayload string) (*types.Message, error) {
	var doc parsedAnforderung
	if err := xml.Unmarshal([]byte(xmlPayload), &doc); err != nil {
		return nil, fmt.Errorf("xml.Unmarshal: %w", err)
	}

	if doc.Gemeinschaft.Prozess == "" {
		return nil, fmt.Errorf("missing Prozess element in XML")
	}

	msg := &types.Message{
		Process:        doc.Gemeinschaft.Prozess,
		Direction:      types.DirectionInbound,
		From:           doc.Marktteilnehmer.Absender,
		To:             doc.Marktteilnehmer.Empfaenger,
		GemeinschaftID: doc.Gemeinschaft.GemeinschaftsID,
		XMLPayload:     xmlPayload,
		CreatedAt:      time.Now().UTC(),
	}
	return msg, nil
}
