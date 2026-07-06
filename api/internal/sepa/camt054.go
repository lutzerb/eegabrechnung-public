package sepa

import (
	"encoding/xml"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CAMT054Entry holds extracted return information from a CAMT.054 bank notification.
type CAMT054Entry struct {
	EndToEndID     string    // maps to invoice UUID (set as EndToEndId in pain.008)
	ReasonCode     string    // e.g. "AC01", "AM04", "MD01"
	AdditionalInfo string    // optional additional text from bank
	Amount         float64
	Currency       string
	BookingDate    time.Time
}

// reNs strips XML namespace declarations and prefixes before parsing.
var reNs = regexp.MustCompile(`\s+xmlns(?::[a-zA-Z0-9_]+)?="[^"]*"`)
var rePrefix = regexp.MustCompile(`<(/?)([a-zA-Z0-9_]+):`)

func stripNamespaces(data []byte) []byte {
	// Remove namespace declarations
	cleaned := reNs.ReplaceAll(data, nil)
	// Remove namespace prefixes from element tags
	cleaned = rePrefix.ReplaceAll(cleaned, []byte("<$1"))
	return cleaned
}

// camt054Doc is the namespace-agnostic struct for parsing CAMT.054.
type camt054Doc struct {
	XMLName xml.Name `xml:"Document"`
	Ntfctns []camt054Ntfctn `xml:"BkToCstmrDbtCdtNtfctn>Ntfctn"`
}

type camt054Ntfctn struct {
	Entries []camt054Entry `xml:"Ntry"`
}

type camt054Entry struct {
	// Some banks put EndToEndId directly in Ntry
	EndToEndID  string `xml:"NtryDtls>TxDtls>Refs>EndToEndId"`
	EndToEndID2 string `xml:"Refs>EndToEndId"` // alternative location
	Sts         string `xml:"Sts>Cd"`
	StsLegacy   string `xml:"Sts"` // some schemas use plain text in Sts
	CdtDbtInd   string `xml:"CdtDbtInd"`
	RvslInd     string `xml:"RvslInd"`
	AmtStr      string `xml:"Amt"`
	Currency    string `xml:"Amt>Ccy"` // won't work as attribute; handled separately
	BookingDate string `xml:"BookgDt>Dt"`
	ValDate     string `xml:"ValDt>Dt"`
	// Return reason info
	ReasonCode  string `xml:"NtryDtls>TxDtls>RtrInf>Rsn>Cd"`
	AddtlInf    string `xml:"NtryDtls>TxDtls>RtrInf>AddtlInf"`
	// Also handle variant: reason directly in Ntry level
	ReasonCode2 string `xml:"RtrInf>Rsn>Cd"`
	AddtlInf2   string `xml:"RtrInf>AddtlInf"`
}

// We need a custom approach because xml.Unmarshal doesn't easily handle
// attribute values. Use a raw element approach for Amt.
type camt054EntryRaw struct {
	XMLName     xml.Name `xml:"Ntry"`
	Sts         struct {
		Cd   string `xml:"Cd"`
		Text string `xml:",chardata"`
	} `xml:"Sts"`
	CdtDbtInd string `xml:"CdtDbtInd"`
	RvslInd   string `xml:"RvslInd"`
	Amt       struct {
		Ccy  string `xml:"Ccy,attr"`
		Text string `xml:",chardata"`
	} `xml:"Amt"`
	BookingDate string `xml:"BookgDt>Dt"`
	ValDate     string `xml:"ValDt>Dt"`
	NtryDtls    []struct {
		TxDtls []struct {
			Refs struct {
				EndToEndID string `xml:"EndToEndId"`
			} `xml:"Refs"`
			RtrInf struct {
				Rsn struct {
					Cd string `xml:"Cd"`
				} `xml:"Rsn"`
				AddtlInf string `xml:"AddtlInf"`
			} `xml:"RtrInf"`
		} `xml:"TxDtls"`
	} `xml:"NtryDtls"`
	// Alternative: EndToEndId directly under Refs in Ntry
	Refs struct {
		EndToEndID string `xml:"EndToEndId"`
	} `xml:"Refs"`
	// Alternative: RtrInf directly under Ntry
	RtrInf struct {
		Rsn struct {
			Cd string `xml:"Cd"`
		} `xml:"Rsn"`
		AddtlInf string `xml:"AddtlInf"`
	} `xml:"RtrInf"`
}

type camt054DocRaw struct {
	XMLName xml.Name `xml:"Document"`
	Ntfctns []struct {
		Entries []camt054EntryRaw `xml:"Ntry"`
	} `xml:"BkToCstmrDbtCdtNtfctn>Ntfctn"`
}

// ParseCamt054 parses a CAMT.054 bank notification XML and extracts return entries.
// It is namespace-agnostic — the namespace variant (.04, .08, .09 etc.) is handled
// by stripping all namespace declarations and prefixes before parsing.
func ParseCamt054(data []byte) ([]CAMT054Entry, error) {
	cleaned := stripNamespaces(data)

	var doc camt054DocRaw
	if err := xml.Unmarshal(cleaned, &doc); err != nil {
		return nil, err
	}

	var results []CAMT054Entry
	for _, ntfctn := range doc.Ntfctns {
		for _, raw := range ntfctn.Entries {
			// Determine status: RJCT = rejected/returned; also check reversal indicator
			sts := strings.ToUpper(strings.TrimSpace(raw.Sts.Cd))
			if sts == "" {
				sts = strings.ToUpper(strings.TrimSpace(raw.Sts.Text))
			}
			rvsl := strings.ToLower(strings.TrimSpace(raw.RvslInd)) == "true"
			if sts != "RJCT" && !rvsl {
				continue
			}

			// Find EndToEndId
			endToEnd := ""
			for _, dtls := range raw.NtryDtls {
				for _, tx := range dtls.TxDtls {
					if tx.Refs.EndToEndID != "" {
						endToEnd = tx.Refs.EndToEndID
						break
					}
				}
				if endToEnd != "" {
					break
				}
			}
			if endToEnd == "" {
				endToEnd = raw.Refs.EndToEndID
			}
			if endToEnd == "" {
				continue
			}

			// Find return reason code
			reasonCode := ""
			addtlInf := ""
			for _, dtls := range raw.NtryDtls {
				for _, tx := range dtls.TxDtls {
					if tx.RtrInf.Rsn.Cd != "" {
						reasonCode = tx.RtrInf.Rsn.Cd
						addtlInf = tx.RtrInf.AddtlInf
						break
					}
				}
				if reasonCode != "" {
					break
				}
			}
			if reasonCode == "" {
				reasonCode = raw.RtrInf.Rsn.Cd
				addtlInf = raw.RtrInf.AddtlInf
			}

			// Parse amount
			amtStr := strings.TrimSpace(raw.Amt.Text)
			var amount float64
			if amtStr != "" {
				amount, _ = strconv.ParseFloat(strings.ReplaceAll(amtStr, ",", "."), 64)
			}
			currency := raw.Amt.Ccy

			// Parse booking date
			var bookingDate time.Time
			if raw.BookingDate != "" {
				bookingDate, _ = time.Parse("2006-01-02", raw.BookingDate)
			} else if raw.ValDate != "" {
				bookingDate, _ = time.Parse("2006-01-02", raw.ValDate)
			}
			if bookingDate.IsZero() {
				bookingDate = time.Now()
			}

			results = append(results, CAMT054Entry{
				EndToEndID:     endToEnd,
				ReasonCode:     reasonCode,
				AdditionalInfo: addtlInf,
				Amount:         amount,
				Currency:       currency,
				BookingDate:    bookingDate,
			})
		}
	}
	return results, nil
}
