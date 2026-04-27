package sepa

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// Pain008Entry is one incoming direct debit (Lastschrift / Bezug).
type Pain008Entry struct {
	Invoice *domain.Invoice
	Member  *domain.Member
}

// GeneratePain008 builds a pain.008.001.08 XML direct-debit initiation (PSA Austrian standard).
// Only invoices with total_amount > 0 and a member IBAN are included.
// The EEG must have its own IBAN and a SEPA Gläubiger-ID configured.
func GeneratePain008(eeg *domain.EEG, entries []Pain008Entry) ([]byte, error) {
	if eeg.IBAN == "" {
		return nil, fmt.Errorf("EEG hat keine IBAN konfiguriert")
	}
	if eeg.SepaCreditorID == "" {
		return nil, fmt.Errorf("EEG hat keine SEPA Gläubiger-ID konfiguriert")
	}

	// Filter: positive amount (member owes EEG) and member has IBAN
	var txs []Pain008Entry
	for _, e := range entries {
		if e.Invoice.TotalAmount > 0 && e.Member.IBAN != "" {
			txs = append(txs, e)
		}
	}
	if len(txs) == 0 {
		return nil, fmt.Errorf("keine Lastschriften mit IBAN vorhanden")
	}

	now := time.Now()
	msgID := fmt.Sprintf("P008-%s", now.Format("20060102150405"))

	// Determine collection date per SEPA Rulebook:
	// invoice date (= pre-notification date) + sepa_pre_notification_days.
	// Use the latest invoice creation time among all included transactions, then add
	// the configured notice period. Clamp to at least tomorrow so the file is always valid.
	noticeDays := eeg.SepaPreNotificationDays
	if noticeDays <= 0 {
		noticeDays = 14
	}
	var latestInvoiceAt time.Time
	for _, tx := range txs {
		if tx.Invoice.CreatedAt.After(latestInvoiceAt) {
			latestInvoiceAt = tx.Invoice.CreatedAt
		}
	}
	earliest := latestInvoiceAt.AddDate(0, 0, noticeDays)
	tomorrow := now.AddDate(0, 0, 1)
	if earliest.Before(tomorrow) {
		earliest = tomorrow
	}
	collectionDate := earliest.Format("2006-01-02")

	var ctrlSum float64
	for _, tx := range txs {
		ctrlSum += tx.Invoice.TotalAmount
	}

	type instdAmt struct {
		Ccy   string `xml:"Ccy,attr"`
		Value string `xml:",chardata"`
	}
	type pmtID struct {
		EndToEndId string `xml:"EndToEndId"`
	}
	type mndtRltdInf struct {
		MndtId    string `xml:"MndtId"`
		DtOfSgntr string `xml:"DtOfSgntr"`
		AmdmntInd bool   `xml:"AmdmntInd"`
	}
	type drctDbtTx struct {
		MndtRltdInf mndtRltdInf `xml:"MndtRltdInf"`
	}
	type dbtrAgt struct {
		FinInstnId struct {
			Othr struct {
				Id string `xml:"Id"`
			} `xml:"Othr"`
		} `xml:"FinInstnId"`
	}
	type party struct {
		Nm string `xml:"Nm"`
	}
	type dbtrAcct struct {
		Id struct {
			IBAN string `xml:"IBAN"`
		} `xml:"Id"`
	}
	type rmtInf struct {
		Ustrd string `xml:"Ustrd"`
	}
	type drctDbtTxInf struct {
		PmtId     pmtID     `xml:"PmtId"`
		InstdAmt  instdAmt  `xml:"InstdAmt"`
		DrctDbtTx drctDbtTx `xml:"DrctDbtTx"`
		DbtrAgt   dbtrAgt   `xml:"DbtrAgt"`
		Dbtr      party     `xml:"Dbtr"`
		DbtrAcct  dbtrAcct  `xml:"DbtrAcct"`
		RmtInf    rmtInf    `xml:"RmtInf"`
	}
	type cdtrAcct struct {
		Id struct {
			IBAN string `xml:"IBAN"`
		} `xml:"Id"`
	}
	type cdtrAgt struct {
		FinInstnId struct {
			BICFI string `xml:"BICFI,omitempty"`
		} `xml:"FinInstnId"`
	}
	type cdtrSchmeId struct {
		Id struct {
			PrvtId struct {
				Othr struct {
					Id     string `xml:"Id"`
					SchmeNm struct {
						Prtry string `xml:"Prtry"`
					} `xml:"SchmeNm"`
				} `xml:"Othr"`
			} `xml:"PrvtId"`
		} `xml:"Id"`
	}
	type pmtTpInf struct {
		SvcLvl  struct{ Cd string `xml:"Cd"` }  `xml:"SvcLvl"`
		LclInstrm struct{ Cd string `xml:"Cd"` } `xml:"LclInstrm"`
		SeqTp   string                            `xml:"SeqTp"`
	}
	type pmtInf struct {
		PmtInfId       string         `xml:"PmtInfId"`
		PmtMtd         string         `xml:"PmtMtd"`
		NbOfTxs        int            `xml:"NbOfTxs"`
		CtrlSum        string         `xml:"CtrlSum"`
		PmtTpInf       pmtTpInf       `xml:"PmtTpInf"`
		ReqdColltnDt   string         `xml:"ReqdColltnDt"`
		Cdtr           party          `xml:"Cdtr"`
		CdtrAcct       cdtrAcct       `xml:"CdtrAcct"`
		CdtrAgt        cdtrAgt        `xml:"CdtrAgt"`
		CdtrSchmeId    cdtrSchmeId    `xml:"CdtrSchmeId"`
		DrctDbtTxInf   []drctDbtTxInf `xml:"DrctDbtTxInf"`
	}
	type grpHdr struct {
		MsgId   string `xml:"MsgId"`
		CreDtTm string `xml:"CreDtTm"`
		NbOfTxs int    `xml:"NbOfTxs"`
		CtrlSum string `xml:"CtrlSum"`
		InitgPty party `xml:"InitgPty"`
	}
	type body struct {
		GrpHdr grpHdr   `xml:"GrpHdr"`
		PmtInf []pmtInf `xml:"PmtInf"`
	}
	type document struct {
		XMLName           xml.Name `xml:"Document"`
		Xmlns             string   `xml:"xmlns,attr"`
		CstmrDrctDbtInitn body     `xml:"CstmrDrctDbtInitn"`
	}

	var txElems []drctDbtTxInf
	for _, tx := range txs {
		memberName := memberFullName(tx.Member)
		ref := invoiceRef(tx.Invoice)

		var da dbtrAcct
		da.Id.IBAN = tx.Member.IBAN
		var dag dbtrAgt
		dag.FinInstnId.Othr.Id = "NOTPROVIDED"

		dt := drctDbtTx{
			MndtRltdInf: mndtRltdInf{
				MndtId:    mandateID(tx.Member),
				DtOfSgntr: mandateDate(tx.Member),
				AmdmntInd: false,
			},
		}

		txElems = append(txElems, drctDbtTxInf{
			PmtId:     pmtID{EndToEndId: strings.ReplaceAll(tx.Invoice.ID.String(), "-", "")},
			InstdAmt:  instdAmt{Ccy: "EUR", Value: fmt.Sprintf("%.2f", tx.Invoice.TotalAmount)},
			DrctDbtTx: dt,
			DbtrAgt:   dag,
			Dbtr:      party{Nm: memberName},
			DbtrAcct:  da,
			RmtInf:    rmtInf{Ustrd: ref},
		})
	}

	var ca cdtrAcct
	ca.Id.IBAN = eeg.IBAN
	var cag cdtrAgt
	cag.FinInstnId.BICFI = eeg.BIC

	var csi cdtrSchmeId
	csi.Id.PrvtId.Othr.Id = eeg.SepaCreditorID
	csi.Id.PrvtId.Othr.SchmeNm.Prtry = "SEPA"

	var pt pmtTpInf
	pt.SvcLvl.Cd = "SEPA"
	pt.LclInstrm.Cd = "CORE"
	pt.SeqTp = "OOFF"

	pi := pmtInf{
		PmtInfId:     msgID + "-PMT",
		PmtMtd:       "DD",
		NbOfTxs:      len(txs),
		CtrlSum:      fmt.Sprintf("%.2f", ctrlSum),
		PmtTpInf:     pt,
		ReqdColltnDt: collectionDate,
		Cdtr:         party{Nm: eeg.Name},
		CdtrAcct:     ca,
		CdtrAgt:      cag,
		CdtrSchmeId:  csi,
		DrctDbtTxInf: txElems,
	}

	doc := document{
		Xmlns: "urn:iso:std:iso:20022:tech:xsd:pain.008.001.08",
		CstmrDrctDbtInitn: body{
			GrpHdr: grpHdr{
				MsgId:    msgID,
				CreDtTm:  now.UTC().Format("2006-01-02T15:04:05Z"),
				NbOfTxs:  len(txs),
				CtrlSum:  fmt.Sprintf("%.2f", ctrlSum),
				InitgPty: party{Nm: eeg.Name},
			},
			PmtInf: []pmtInf{pi},
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal pain.008: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}
