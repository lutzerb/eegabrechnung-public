package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// ConsumptionRecord XML schema version 01.30
// Namespace: http://www.ebutilities.at/schemata/customerprocesses/consumptionrecord/01p30
//
// Structure (namespace prefixes stripped for clarity):
//
//	ConsumptionRecord
//	  MarketParticipantDirectory @DocumentMode=PROD|SIMU
//	    RoutingHeader
//	      Sender/MessageAddress
//	      Receiver/MessageAddress
//	  ProcessDirectory
//	    MessageId
//	    MeteringPoint (AT... Zählpunkt)
//	    DeliveryPoint (EC number = GemeinschaftID)
//	    Energy (1..n)
//	      MeteringPeriodStart / MeteringPeriodEnd (DateTimeU: ISO 8601 with +HH:MM)
//	      MeteringIntervall (QH|H|D|V)
//	      EnergyData @MeterCode @UOM (1..n)
//	        EP (unbounded)
//	          DTF  – from datetime
//	          DTT  – to datetime (optional)
//	          BQ   – billing quantity (kWh when UOM=KWH)

// CRMsgRecord is the parsed result of a ConsumptionRecord (CR_MSG) message.
type CRMsgRecord struct {
	DocumentMode   string       // "PROD" or "SIMU"
	MessageID      string       // unique message identifier
	ConversationID string       // links back to the EC_REQ_PT process
	Zaehlpunkt     string       // MeteringPoint (AT...)
	GemeinschaftID string       // DeliveryPoint EC number
	From           string       // Sender MessageAddress
	To             string       // Receiver MessageAddress
	Energies       []CRMsgBlock // one block per Energy element
}

// CRMsgBlock holds all EnergyData for one Energy period.
type CRMsgBlock struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	Intervall   string // QH|H|D|V
	Data        []CRMsgEnergyData
}

// CRMsgEnergyData holds positions for one OBIS code / MeterCode.
type CRMsgEnergyData struct {
	MeterCode string // OBIS code attribute, e.g. "1-1:1.9.0" or "1-1:1.9.0 G.01"
	UOM       string // unit of measurement, e.g. "KWH"
	Positions []CRMsgPosition
}

// CRMsgPosition is a single time-series value.
type CRMsgPosition struct {
	From  time.Time
	To    time.Time  // zero if not present
	Value float64    // kWh (when UOM=KWH)
}

// IsCRMsg returns true if the XML payload looks like a ConsumptionRecord message.
func IsCRMsg(xmlPayload string) bool {
	return strings.Contains(xmlPayload, "ConsumptionRecord")
}

// ParseCRMsg parses an Austrian EDA ConsumptionRecord (CR_MSG) XML body.
// It uses local-name matching so namespace prefixes in the document do not matter.
func ParseCRMsg(xmlPayload string) (*CRMsgRecord, error) {
	var doc xmlCRDoc
	if err := xml.Unmarshal([]byte(xmlPayload), &doc); err != nil {
		return nil, fmt.Errorf("xml.Unmarshal: %w", err)
	}

	gemeinschaftID := doc.ProcessDir.DeliveryPoint
	if gemeinschaftID == "" {
		gemeinschaftID = doc.ProcessDir.ECID // schema ≥01.41 renamed DeliveryPoint → ECID
	}
	record := &CRMsgRecord{
		DocumentMode:   doc.MarketParticipant.DocumentMode,
		MessageID:      doc.ProcessDir.MessageID,
		ConversationID: doc.ProcessDir.ConversationID,
		Zaehlpunkt:     doc.ProcessDir.MeteringPoint,
		GemeinschaftID: gemeinschaftID,
		From:           doc.MarketParticipant.RoutingHeader.Sender.MessageAddress,
		To:             doc.MarketParticipant.RoutingHeader.Receiver.MessageAddress,
	}

	for _, e := range doc.ProcessDir.Energies {
		start, err := parseCRDateTime(e.PeriodStart)
		if err != nil {
			return nil, fmt.Errorf("invalid MeteringPeriodStart %q: %w", e.PeriodStart, err)
		}
		end, err := parseCRDateTime(e.PeriodEnd)
		if err != nil {
			return nil, fmt.Errorf("invalid MeteringPeriodEnd %q: %w", e.PeriodEnd, err)
		}

		block := CRMsgBlock{
			PeriodStart: start.UTC(),
			PeriodEnd:   end.UTC(),
			Intervall:   e.MeteringIntervall,
		}

		for _, ed := range e.EnergyData {
			dataBlock := CRMsgEnergyData{
				MeterCode: ed.MeterCode,
				UOM:       ed.UOM,
			}
			for _, ep := range ed.EP {
				dtf, err := parseCRDateTime(ep.DTF)
				if err != nil {
					continue
				}
				pos := CRMsgPosition{
					From:  dtf.UTC(),
					Value: ep.BQ,
				}
				if ep.DTT != "" {
					if dtt, err := parseCRDateTime(ep.DTT); err == nil {
						pos.To = dtt.UTC()
					}
				}
				dataBlock.Positions = append(dataBlock.Positions, pos)
			}
			if len(dataBlock.Positions) > 0 {
				block.Data = append(block.Data, dataBlock)
			}
		}
		record.Energies = append(record.Energies, block)
	}

	return record, nil
}

// parseCRDateTime parses ISO 8601 datetimes as used in Austrian EDA (DateTimeU type).
// Format: 2006-01-02T15:04:05+01:00 or 2006-01-02T15:04:05Z
func parseCRDateTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty datetime string")
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse datetime %q", s)
}

// ── XML unmarshal structs ──────────────────────────────────────────────────
// Uses local-name matching (xml:"localname") so namespace prefixes are ignored.

type xmlCRDoc struct {
	XMLName          xml.Name              `xml:"ConsumptionRecord"`
	MarketParticipant xmlCRMarketDir       `xml:"MarketParticipantDirectory"`
	ProcessDir        xmlCRProcessDir      `xml:"ProcessDirectory"`
}

type xmlCRMarketDir struct {
	DocumentMode  string           `xml:"DocumentMode,attr"`
	RoutingHeader xmlCRRoutingHdr  `xml:"RoutingHeader"`
}

type xmlCRRoutingHdr struct {
	Sender struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Sender"`
	Receiver struct {
		MessageAddress string `xml:"MessageAddress"`
	} `xml:"Receiver"`
}

type xmlCRProcessDir struct {
	MessageID      string        `xml:"MessageId"`
	ConversationID string        `xml:"ConversationId"`
	MeteringPoint  string        `xml:"MeteringPoint"`
	DeliveryPoint  string        `xml:"DeliveryPoint"` // schema ≤01.30
	ECID           string        `xml:"ECID"`          // schema ≥01.41 (replaces DeliveryPoint)
	Energies       []xmlCREnergy `xml:"Energy"`
}

type xmlCREnergy struct {
	PeriodStart       string              `xml:"MeteringPeriodStart"`
	PeriodEnd         string              `xml:"MeteringPeriodEnd"`
	MeteringIntervall string              `xml:"MeteringIntervall"`
	EnergyData        []xmlCREnergyData   `xml:"EnergyData"`
}

type xmlCREnergyData struct {
	MeterCode string               `xml:"MeterCode,attr"`
	UOM       string               `xml:"UOM,attr"`
	EP        []xmlCREnergyPosition `xml:"EP"`
}

type xmlCREnergyPosition struct {
	DTF string  `xml:"DTF"`
	DTT string  `xml:"DTT"`
	BQ  float64 `xml:"BQ"`
}
