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
		td := currentTimeData(mpd.TimeDatas)
		entry := ECMPListEntry{
			MeteringPoint:   mpd.MeteringPoint,
			ConsentID:       mpd.ConsentID,
			EnergyDirection: td.EnergyDirection,
			ECPartFact:      td.ECPartFact,
			PlantCategory:   td.PlantCategory,
			ECShare:         td.ECShare,
		}
		if td.DateFrom != "" {
			if t, err := parseCRDateTime(td.DateFrom); err == nil {
				entry.DateFrom = t.UTC()
			}
		}
		if td.DateTo != "" && td.DateTo != "9999-12-31" {
			if t, err := parseCRDateTime(td.DateTo); err == nil {
				entry.DateTo = t.UTC()
			}
		}
		if td.DateActivate != "" {
			if t, err := parseCRDateTime(td.DateActivate); err == nil {
				entry.DateActivate = t.UTC()
			}
		}
		res.Entries = append(res.Entries, entry)
	}

	return res, nil
}

// currentTimeData picks the MPTimeData period that is currently valid for a
// MeteringPoint. A real ECMPList entry can list several MPTimeData periods for
// the same Zählpunkt — its participation factor (or energy direction/plant
// category) can change over time, e.g. a short-term reduction that reverts
// later. The schema lists these chronologically, but we don't rely on that
// implicitly (Go's encoding/xml would otherwise silently take whichever one
// happens to be last in document order for a non-slice field): prefer the
// open-ended period (DateTo == "" or the "9999-12-31" sentinel, i.e. "valid
// until further notice"); if none is open-ended (the Zählpunkt currently has
// no active period at all), fall back to the most recent one by DateFrom —
// ISO date strings ("YYYY-MM-DD") sort correctly as plain strings.
func currentTimeData(periods []xmlECMPListTimeData) xmlECMPListTimeData {
	if len(periods) == 0 {
		return xmlECMPListTimeData{}
	}
	best := periods[0]
	bestOpen := best.DateTo == "" || best.DateTo == "9999-12-31"
	for _, p := range periods[1:] {
		open := p.DateTo == "" || p.DateTo == "9999-12-31"
		switch {
		case open && !bestOpen:
			best, bestOpen = p, true
		case open == bestOpen && p.DateFrom > best.DateFrom:
			best = p
		}
	}
	return best
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
	MeteringPoint string                `xml:"MeteringPoint"`
	ConsentID     string                `xml:"ConsentId"`
	TimeDatas     []xmlECMPListTimeData `xml:"MPTimeData"`
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
