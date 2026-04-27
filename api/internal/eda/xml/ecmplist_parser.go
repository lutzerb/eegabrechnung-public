package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// ECMPListEntry represents one meter point entry in an ECMPList message.
type ECMPListEntry struct {
	MeteringPoint   string
	ConsentID       string
	DateFrom        time.Time
	DateTo          time.Time
	EnergyDirection string // GENERATION | CONSUMPTION
	ECPartFact      string
	PlantCategory   string
	DateActivate    time.Time
	ECShare         string
}

// ECMPListResult holds the parsed content of an incoming ECMPList message.
// ECMPList messages (SENDEN_ECP or ABSCHLUSS_ECON) are sent by the Netzbetreiber
// to confirm the current meter point list of an energy community.
type ECMPListResult struct {
	DocumentMode   string
	MessageCode    string // SENDEN_ECP | ABSCHLUSS_ECON
	MessageID      string
	ConversationID string
	ECID           string
	ECType         string
	ECDisModel     string
	ProcessDate    time.Time
	Entries        []ECMPListEntry
	From           string
	To             string
}

// IsECMPList returns true if the XML payload looks like an ECMPList message.
func IsECMPList(xmlPayload string) bool {
	return strings.Contains(xmlPayload, "ECMPList")
}

// ParseECMPList parses an incoming Austrian EDA ECMPList XML message.
// Uses local-name matching so namespace prefixes in the document are irrelevant.
func ParseECMPList(xmlPayload string) (*ECMPListResult, error) {
	var doc xmlECMPListRoot
	if err := xml.Unmarshal([]byte(xmlPayload), &doc); err != nil {
		return nil, fmt.Errorf("xml.Unmarshal: %w", err)
	}

	res := &ECMPListResult{
		DocumentMode:   doc.MarketDir.DocumentMode,
		MessageCode:    doc.MarketDir.MessageCode,
		MessageID:      doc.ProcessDir.MessageID,
		ConversationID: doc.ProcessDir.ConversationID,
		ECID:           doc.ProcessDir.ECID,
		ECType:         doc.ProcessDir.ECType,
		ECDisModel:     doc.ProcessDir.ECDisModel,
		From:           doc.MarketDir.RoutingHeader.Sender.MessageAddress,
		To:             doc.MarketDir.RoutingHeader.Receiver.MessageAddress,
	}

	if doc.ProcessDir.ProcessDate != "" {
		if t, err := parseCRDateTime(doc.ProcessDir.ProcessDate); err == nil {
			res.ProcessDate = t.UTC()
		}
	}

	for _, mpd := range doc.ProcessDir.MPListData {
		entry := ECMPListEntry{
			MeteringPoint:   mpd.MeteringPoint,
			ConsentID:       mpd.ConsentID,
			EnergyDirection: mpd.TimeData.EnergyDirection,
			ECPartFact:      mpd.TimeData.ECPartFact,
			PlantCategory:   mpd.TimeData.PlantCategory,
			ECShare:         mpd.TimeData.ECShare,
		}
		if mpd.TimeData.DateFrom != "" {
			if t, err := parseCRDateTime(mpd.TimeData.DateFrom); err == nil {
				entry.DateFrom = t.UTC()
			}
		}
		if mpd.TimeData.DateTo != "" && mpd.TimeData.DateTo != "9999-12-31" {
			if t, err := parseCRDateTime(mpd.TimeData.DateTo); err == nil {
				entry.DateTo = t.UTC()
			}
		}
		if mpd.TimeData.DateActivate != "" {
			if t, err := parseCRDateTime(mpd.TimeData.DateActivate); err == nil {
				entry.DateActivate = t.UTC()
			}
		}
		res.Entries = append(res.Entries, entry)
	}

	return res, nil
}

// ── XML unmarshal structs (local-name matching) ─────────────────────────

type xmlECMPListRoot struct {
	XMLName    xml.Name              `xml:"ECMPList"`
	MarketDir  xmlECMPListMarketDir  `xml:"MarketParticipantDirectory"`
	ProcessDir xmlECMPListProcessDir `xml:"ProcessDirectory"`
}

type xmlECMPListMarketDir struct {
	DocumentMode  string             `xml:"DocumentMode,attr"`
	MessageCode   string             `xml:"MessageCode"`
	RoutingHeader xmlECMPListRouting `xml:"RoutingHeader"`
}

type xmlECMPListRouting struct {
	Sender struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Sender"`
	Receiver struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Receiver"`
}

type xmlECMPListProcessDir struct {
	MessageID      string             `xml:"MessageId"`
	ConversationID string             `xml:"ConversationId"`
	ProcessDate    string             `xml:"ProcessDate"`
	ECID           string             `xml:"ECID"`
	ECType         string             `xml:"ECType"`
	ECDisModel     string             `xml:"ECDisModel"`
	MPListData     []xmlECMPListMPData `xml:"MPListData"`
}

type xmlECMPListMPData struct {
	MeteringPoint string             `xml:"MeteringPoint"`
	ConsentID     string             `xml:"ConsentId"`
	TimeData      xmlECMPListTimeData `xml:"MPTimeData"`
}

type xmlECMPListTimeData struct {
	DateFrom        string `xml:"DateFrom"`
	DateTo          string `xml:"DateTo"`
	EnergyDirection string `xml:"EnergyDirection"`
	ECPartFact      string `xml:"ECPartFact"`
	PlantCategory   string `xml:"PlantCategory"`
	DateActivate    string `xml:"DateActivate"`
	ECShare         string `xml:"ECShare"`
}
