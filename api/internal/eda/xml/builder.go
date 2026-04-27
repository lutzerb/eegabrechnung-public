// Package xml provides XML builders and parsers for the Austrian MaKo protocol.
package xml

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/processes"
)

const (
	edaNS  = "http://www.ebutilities.at/saldomodell/EDA/1.0"
	xsiNS  = "http://www.w3.org/2001/XMLSchema-instance"
	layout = "2006-01-02T15:04:05"
)

// BuildRequest builds an outbound MaKo XML Anforderung message.
type BuildParams struct {
	From           string
	To             string
	GemeinschaftID string
	Process        string
	FromDate       time.Time
	ToDate         time.Time
}

// anforderungXML mirrors the MaKo XML structure for marshalling.
type anforderungXML struct {
	XMLName        xml.Name           `xml:"req:Anforderung"`
	NS             string             `xml:"xmlns:req,attr"`
	NSxsi          string             `xml:"xmlns:xsi,attr"`
	Marktteilnehmer marktteilnehmerXML `xml:"req:Marktteilnehmer"`
	Gemeinschaft   gemeinschaftXML    `xml:"req:Gemeinschaft"`
}

type marktteilnehmerXML struct {
	Absender  string `xml:"req:Absender"`
	Empfaenger string `xml:"req:Empfaenger"`
}

type gemeinschaftXML struct {
	GemeinschaftsID string `xml:"req:GemeinschaftsID"`
	Prozess         string `xml:"req:Prozess"`
	Version         string `xml:"req:Version"`
	ZeitpunktVon    string `xml:"req:ZeitpunktVon"`
	ZeitpunktBis    string `xml:"req:ZeitpunktBis"`
}

// BuildAnforderung creates a MaKo XML request message body.
func BuildAnforderung(p BuildParams) (string, error) {
	if p.From == "" {
		return "", fmt.Errorf("From address is required")
	}
	if p.To == "" {
		return "", fmt.Errorf("To address is required")
	}
	if p.GemeinschaftID == "" {
		return "", fmt.Errorf("GemeinschaftID is required")
	}
	if !processes.IsKnown(p.Process) {
		return "", fmt.Errorf("unknown process code: %s", p.Process)
	}

	version, err := processes.Version(p.Process)
	if err != nil {
		return "", err
	}

	doc := anforderungXML{
		NS:    edaNS,
		NSxsi: xsiNS,
		Marktteilnehmer: marktteilnehmerXML{
			Absender:   p.From,
			Empfaenger: p.To,
		},
		Gemeinschaft: gemeinschaftXML{
			GemeinschaftsID: p.GemeinschaftID,
			Prozess:         p.Process,
			Version:         version,
			ZeitpunktVon:    p.FromDate.UTC().Format(layout),
			ZeitpunktBis:    p.ToDate.UTC().Format(layout),
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("xml.Marshal: %w", err)
	}
	return xml.Header + string(out), nil
}
