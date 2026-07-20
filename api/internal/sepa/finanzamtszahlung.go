package sepa

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// GenerateFinanzamtZahlung builds a pain.001.001.09 SEPA credit transfer marked as a
// "Finanzamtszahlung" (FAZ) per the PSA specification "Finanzamtszahlung in EBICS"
// (Payment Services Austria GmbH, v1.0.01, 20.10.2022, see https://zv.psa.at):
//
//   - <Purp><Cd>TAXS</Cd></Purp> on the individual transaction marks it as a tax payment
//   - <PmtId><EndToEndId> carries the 9-digit "Ordnungsbegriff" (FA-Nr + Steuernummer + Prüfziffer)
//   - <RmtInf><Ustrd> carries the structured "Verrechnungsweisung" (see BuildVerrechnungsweisung)
//
// The receiving bank/Finanzamt recognizes the IBAN + TAXS purpose code and auto-allocates the
// payment to the named Abgabenart/Zeitraum instead of treating it as a generic transfer.
func GenerateFinanzamtZahlung(eeg *domain.EEG, ordnungsbegriff, finanzamtName, finanzamtIBAN string, betrag float64, verrechnungsweisung string, execDate time.Time) ([]byte, error) {
	if eeg.IBAN == "" {
		return nil, fmt.Errorf("EEG hat keine IBAN konfiguriert")
	}
	if finanzamtIBAN == "" {
		return nil, fmt.Errorf("Finanzamt-IBAN ist nicht konfiguriert (EA-Einstellungen)")
	}
	if betrag <= 0.005 {
		return nil, fmt.Errorf("keine Zahllast fällig")
	}
	// IBAN2007Identifier erlaubt keine Leerzeichen — Finanzamt-IBANs werden üblicherweise
	// gruppiert kopiert (z.B. "AT08 0100 0000 0550 4295").
	finanzamtIBAN = strings.ReplaceAll(finanzamtIBAN, " ", "")

	now := time.Now()
	msgID := fmt.Sprintf("FAZ-%s", now.Format("20060102150405"))
	if execDate.IsZero() {
		execDate = now.AddDate(0, 0, 2)
	}
	execDateStr := execDate.Format("2006-01-02")
	amountStr := fmt.Sprintf("%.2f", betrag)

	// Element order within CdtTrfTxInf follows the strict xs:sequence in
	// ISO.pain.001.001.09.austrian.005.xsd (CreditTransferTransaction34):
	// PmtId, Amt, Cdtr, CdtrAcct, Purp, RmtInf.
	type instdAmt struct {
		Ccy   string `xml:"Ccy,attr"`
		Value string `xml:",chardata"`
	}
	type pmtID struct {
		EndToEndId string `xml:"EndToEndId"`
	}
	type party struct {
		Nm string `xml:"Nm"`
	}
	type cdtrAcct struct {
		Id struct {
			IBAN string `xml:"IBAN"`
		} `xml:"Id"`
	}
	type purp struct {
		Cd string `xml:"Cd"`
	}
	type rmtInf struct {
		Ustrd string `xml:"Ustrd"`
	}
	type cdtTrfTxInf struct {
		PmtId    pmtID    `xml:"PmtId"`
		Amt      instdAmt `xml:"Amt>InstdAmt"`
		Cdtr     party    `xml:"Cdtr"`
		CdtrAcct cdtrAcct `xml:"CdtrAcct"`
		Purp     purp     `xml:"Purp"`
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
		MsgId    string `xml:"MsgId"`
		CreDtTm  string `xml:"CreDtTm"`
		NbOfTxs  int    `xml:"NbOfTxs"`
		CtrlSum  string `xml:"CtrlSum"`
		InitgPty party  `xml:"InitgPty"`
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

	var ca cdtrAcct
	ca.Id.IBAN = finanzamtIBAN
	tx := cdtTrfTxInf{
		PmtId:    pmtID{EndToEndId: ordnungsbegriff},
		Amt:      instdAmt{Ccy: "EUR", Value: amountStr},
		Cdtr:     party{Nm: finanzamtName},
		CdtrAcct: ca,
		Purp:     purp{Cd: "TAXS"},
		RmtInf:   rmtInf{Ustrd: verrechnungsweisung},
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
		NbOfTxs:     1,
		CtrlSum:     amountStr,
		PmtTpInf:    pt,
		ReqdExctnDt: dateChoice{Dt: execDateStr},
		Dbtr:        party{Nm: eeg.Name},
		DbtrAcct:    da,
		DbtrAgt:     dag,
		ChrgBr:      "SLEV",
		CdtTrfTxInf: []cdtTrfTxInf{tx},
	}

	doc := document{
		Xmlns: "urn:iso:std:iso:20022:tech:xsd:pain.001.001.09",
		CstmrCdtTrfInitn: body{
			GrpHdr: grpHdr{
				MsgId:    msgID,
				CreDtTm:  now.UTC().Format("2006-01-02T15:04:05Z"),
				NbOfTxs:  1,
				CtrlSum:  amountStr,
				InitgPty: party{Nm: eeg.Name},
			},
			PmtInf: []pmtInf{pi},
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal pain.001 (FAZ): %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}
