// Package sepa generates SEPA XML payment files (pain.001, pain.008).
package sepa

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// Pain001Entry is one outgoing credit transfer (Gutschrift / Einspeisevergütung).
type Pain001Entry struct {
	Invoice *domain.Invoice
	Member  *domain.Member
}

// GeneratePain001 builds a pain.001.001.09 XML credit-transfer initiation (PSA Austrian standard).
// Only invoices with total_amount < 0 and a member IBAN are included.
// The EEG must have its own IBAN configured.
func GeneratePain001(eeg *domain.EEG, entries []Pain001Entry) ([]byte, error) {
	if eeg.IBAN == "" {
		return nil, fmt.Errorf("EEG hat keine IBAN konfiguriert")
	}

	// Filter: negative amount (EEG owes producer) and member has IBAN
	var txs []Pain001Entry
	for _, e := range entries {
		if e.Invoice.TotalAmount < 0 && e.Member.IBAN != "" {
			txs = append(txs, e)
		}
	}
	if len(txs) == 0 {
		return nil, fmt.Errorf("keine Gutschriften mit IBAN vorhanden")
	}

	now := time.Now()
	msgID := fmt.Sprintf("P001-%s", now.Format("20060102150405"))
	execDate := now.AddDate(0, 0, 2).Format("2006-01-02")

	var ctrlSum float64
	for _, tx := range txs {
		ctrlSum += -tx.Invoice.TotalAmount // amounts are negative, sum absolute values
	}

	type instdAmt struct {
		Ccy   string `xml:"Ccy,attr"`
		Value string `xml:",chardata"`
	}
	type pmtID struct {
		EndToEndId string `xml:"EndToEndId"`
	}
	type cdtrAcct struct {
		Id struct {
			IBAN string `xml:"IBAN"`
		} `xml:"Id"`
	}
	type party struct {
		Nm string `xml:"Nm"`
	}
	type rmtInf struct {
		Ustrd string `xml:"Ustrd"`
	}
	type cdtTrfTxInf struct {
		PmtId    pmtID    `xml:"PmtId"`
		Amt      instdAmt `xml:"Amt>InstdAmt"`
		Cdtr     party    `xml:"Cdtr"`
		CdtrAcct cdtrAcct `xml:"CdtrAcct"`
		RmtInf   rmtInf   `xml:"RmtInf"`
	}

	type dbtrAcct struct {
		Id struct {
			IBAN string `xml:"IBAN"`
		} `xml:"Id"`
	}
	type dbtrAgt struct {
		FinInstnId struct {
			BICFI string `xml:"BICFI,omitempty"`
		} `xml:"FinInstnId"`
	}
	// DateAndDateTime2Choice: pain.001.001.09 uses <ReqdExctnDt><Dt>...</Dt></ReqdExctnDt>
	type dateChoice struct {
		Dt string `xml:"Dt"`
	}
	type pmtTpInf struct {
		SvcLvl struct {
			Cd string `xml:"Cd"`
		} `xml:"SvcLvl"`
	}
	type pmtInf struct {
		PmtInfId    string        `xml:"PmtInfId"`
		PmtMtd      string        `xml:"PmtMtd"`
		BtchBookg   bool          `xml:"BtchBookg"`
		NbOfTxs     int           `xml:"NbOfTxs"`
		CtrlSum     string        `xml:"CtrlSum"`
		PmtTpInf    pmtTpInf      `xml:"PmtTpInf"`
		ReqdExctnDt dateChoice    `xml:"ReqdExctnDt"`
		Dbtr        party         `xml:"Dbtr"`
		DbtrAcct    dbtrAcct      `xml:"DbtrAcct"`
		DbtrAgt     dbtrAgt       `xml:"DbtrAgt"`
		ChrgBr      string        `xml:"ChrgBr"`
		CdtTrfTxInf []cdtTrfTxInf `xml:"CdtTrfTxInf"`
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
		XMLName          xml.Name `xml:"Document"`
		Xmlns            string   `xml:"xmlns,attr"`
		CstmrCdtTrfInitn body     `xml:"CstmrCdtTrfInitn"`
	}

	var txElems []cdtTrfTxInf
	for _, tx := range txs {
		amount := -tx.Invoice.TotalAmount
		memberName := memberFullName(tx.Member)
		ref := invoiceRef(tx.Invoice)
		var ca cdtrAcct
		ca.Id.IBAN = tx.Member.IBAN
		txElems = append(txElems, cdtTrfTxInf{
			PmtId:    pmtID{EndToEndId: strings.ReplaceAll(tx.Invoice.ID.String(), "-", "")},
			Amt:      instdAmt{Ccy: "EUR", Value: fmt.Sprintf("%.2f", amount)},
			Cdtr:     party{Nm: memberName},
			CdtrAcct: ca,
			RmtInf:   rmtInf{Ustrd: ref},
		})
	}

	var da dbtrAcct
	da.Id.IBAN = eeg.IBAN
	var dag dbtrAgt
	dag.FinInstnId.BICFI = eeg.BIC

	var pt pmtTpInf
	pt.SvcLvl.Cd = "SEPA"

	pi := pmtInf{
		PmtInfId:    msgID + "-PMT",
		PmtMtd:      "TRF",
		BtchBookg:   false,
		NbOfTxs:     len(txs),
		CtrlSum:     fmt.Sprintf("%.2f", ctrlSum),
		PmtTpInf:    pt,
		ReqdExctnDt: dateChoice{Dt: execDate},
		Dbtr:        party{Nm: eeg.Name},
		DbtrAcct:    da,
		DbtrAgt:     dag,
		ChrgBr:      "SLEV",
		CdtTrfTxInf: txElems,
	}

	doc := document{
		Xmlns: "urn:iso:std:iso:20022:tech:xsd:pain.001.001.09",
		CstmrCdtTrfInitn: body{
			GrpHdr: grpHdr{
				MsgId:   msgID,
				CreDtTm: now.UTC().Format("2006-01-02T15:04:05Z"),
				NbOfTxs: len(txs),
				CtrlSum: fmt.Sprintf("%.2f", ctrlSum),
				InitgPty: party{Nm: eeg.Name},
			},
			PmtInf: []pmtInf{pi},
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal pain.001: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}

func memberFullName(m *domain.Member) string {
	if m.Name2 != "" {
		return m.Name1 + " " + m.Name2
	}
	return m.Name1
}

func invoiceRef(inv *domain.Invoice) string {
	period := inv.PeriodStart.Format("01/2006")
	if inv.InvoiceNumber != nil {
		return fmt.Sprintf("Rg. %d Periode %s", *inv.InvoiceNumber, period)
	}
	return fmt.Sprintf("Rg. %s Periode %s", inv.ID.String()[:8], period)
}

func mandateID(m *domain.Member) string {
	if m.MitgliedsNr != "" {
		return m.MitgliedsNr
	}
	return m.ID.String()[:8]
}

func mandateDate(m *domain.Member) string {
	if m.SepaMandateSignedAt != nil {
		return m.SepaMandateSignedAt.Format("2006-01-02")
	}
	return m.CreatedAt.Format("2006-01-02")
}

// BuildEntries maps invoices to their members for SEPA generation.
func BuildEntries(invoices []domain.Invoice, membersByID map[uuid.UUID]*domain.Member) []Pain001Entry {
	var entries []Pain001Entry
	for i := range invoices {
		inv := &invoices[i]
		m, ok := membersByID[inv.MemberID]
		if !ok {
			continue
		}
		entries = append(entries, Pain001Entry{Invoice: inv, Member: m})
	}
	return entries
}
