package handler

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ParseMT940 parses an MT940 bank statement file into EABankTransaktionen.
// MT940 is a SWIFT standard used by most Austrian banks (.sta files).
func ParseMT940(data []byte, eegID uuid.UUID) ([]domain.EABankTransaktion, error) {
	// Normalize line endings
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var result []domain.EABankTransaktion
	var currentIBAN string

	lines := strings.Split(text, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimRight(lines[i], " ")
		// Merge continuation lines (start with space or no tag)
		for i+1 < len(lines) && len(lines[i+1]) > 0 && lines[i+1][0] == ' ' {
			i++
			line = line + strings.TrimLeft(lines[i], " ")
		}

		if strings.HasPrefix(line, ":25:") {
			// Account IBAN/number
			raw := strings.TrimPrefix(line, ":25:")
			raw = strings.TrimSpace(raw)
			// Remove currency suffix if present (e.g. "AT611904300234573201/EUR")
			if idx := strings.Index(raw, "/"); idx >= 0 {
				currentIBAN = raw[:idx]
			} else {
				currentIBAN = raw
			}
		} else if strings.HasPrefix(line, ":61:") {
			// Transaction line
			raw := strings.TrimPrefix(line, ":61:")
			t, err := parseMT940Transaction(raw, eegID, currentIBAN)
			if err == nil {
				// Look ahead for :86: narrative
				if i+1 < len(lines) && strings.HasPrefix(lines[i+1], ":86:") {
					i++
					narrative := strings.TrimPrefix(lines[i], ":86:")
					// Merge continuation lines
					for i+1 < len(lines) && len(lines[i+1]) > 0 && lines[i+1][0] == ' ' {
						i++
						narrative += strings.TrimLeft(lines[i], " ")
					}
					t.Verwendungszweck = cleanMT940Text(narrative)
				}
				result = append(result, t)
			}
		}
		i++
	}
	return result, nil
}

func parseMT940Transaction(raw string, eegID uuid.UUID, iban string) (domain.EABankTransaktion, error) {
	// Format: YYMMDD[MMDD]2aN15d[//16x][34x]
	// e.g.: 260410D1234,56NTRFMUSTERMANN
	if len(raw) < 10 {
		return domain.EABankTransaktion{}, fmt.Errorf("too short")
	}
	dateStr := raw[:6] // YYMMDD
	year, _ := strconv.Atoi("20" + dateStr[:2])
	month, _ := strconv.Atoi(dateStr[2:4])
	day, _ := strconv.Atoi(dateStr[4:6])
	buchDatum := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	rest := raw[6:]
	// Optional valuta date (MMDD)
	var valuta *time.Time
	if len(rest) >= 4 && rest[0] >= '0' && rest[0] <= '9' {
		vm, _ := strconv.Atoi(rest[:2])
		vd, _ := strconv.Atoi(rest[2:4])
		if vm >= 1 && vm <= 12 && vd >= 1 && vd <= 31 {
			vt := time.Date(year, time.Month(vm), vd, 0, 0, 0, 0, time.UTC)
			valuta = &vt
			rest = rest[4:]
		}
	}

	if len(rest) < 2 {
		return domain.EABankTransaktion{}, fmt.Errorf("too short after date")
	}
	direction := rest[0]
	rest = rest[1:]

	// Amount: digits with comma as decimal separator
	amtEnd := strings.IndexAny(rest, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if amtEnd < 0 {
		return domain.EABankTransaktion{}, fmt.Errorf("no type code found")
	}
	amtStr := strings.Replace(rest[:amtEnd], ",", ".", 1)
	amt, err := strconv.ParseFloat(amtStr, 64)
	if err != nil {
		return domain.EABankTransaktion{}, fmt.Errorf("parse amount: %w", err)
	}
	if direction == 'D' {
		amt = -amt
	}

	// Skip transaction type code (4 chars) and reference
	rest = rest[amtEnd:]
	// reference after next /
	referenz := ""
	if idx := strings.Index(rest, "//"); idx >= 0 {
		refEnd := strings.Index(rest[idx+2:], "\n")
		if refEnd < 0 {
			referenz = rest[idx+2:]
		} else {
			referenz = rest[idx+2 : idx+2+refEnd]
		}
	}

	t := domain.EABankTransaktion{
		EegID:         eegID,
		ImportFormat:  "MT940",
		KontoIBAN:     iban,
		Buchungsdatum: buchDatum,
		Valutadatum:   valuta,
		Betrag:        amt,
		Waehrung:      "EUR",
		Referenz:      referenz,
		MatchStatus:   "offen",
	}
	return t, nil
}

func cleanMT940Text(s string) string {
	// Remove SWIFT sub-field codes like ?20, ?21, ?22 etc.
	var parts []string
	i := 0
	for i < len(s) {
		if s[i] == '?' && i+2 < len(s) {
			i += 3 // skip ?XX
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		parts = append(parts, string(r))
		i += size
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

// ParseCAMT053 parses a CAMT.053 (ISO 20022) bank statement XML.
func ParseCAMT053(data []byte, eegID uuid.UUID) ([]domain.EABankTransaktion, error) {
	type Amount struct {
		Ccy   string `xml:"Ccy,attr"`
		Value string `xml:",chardata"`
	}
	type Entry struct {
		Amount    Amount `xml:"Amt"`
		CreditDebit string `xml:"CdtDbtInd"`
		BookgDt   struct {
			Dt string `xml:"Dt"`
		} `xml:"BookgDt"`
		ValDt struct {
			Dt string `xml:"Dt"`
		} `xml:"ValDt"`
		AcctSvcrRef string `xml:"AcctSvcrRef"`
		NtryDtls    struct {
			TxDtls []struct {
				Refs struct {
					EndToEndId string `xml:"EndToEndId"`
				} `xml:"Refs"`
				RmtInf struct {
					Ustrd string `xml:"Ustrd"`
				} `xml:"RmtInf"`
				RltdPties struct {
					Dbtr struct {
						Nm string `xml:"Nm"`
					} `xml:"Dbtr"`
					Cdtr struct {
						Nm string `xml:"Cdtr"`
					} `xml:"Cdtr"`
				} `xml:"RltdPties"`
			} `xml:"TxDtls"`
		} `xml:"NtryDtls"`
	}
	type Stmt struct {
		Acct struct {
			Id struct {
				IBAN string `xml:"IBAN"`
			} `xml:"Id"`
		} `xml:"Acct"`
		Ntry []Entry `xml:"Ntry"`
	}
	type Document struct {
		BkToCstmrStmt struct {
			Stmt []Stmt `xml:"Stmt"`
		} `xml:"BkToCstmrStmt"`
	}

	var doc Document
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("xml decode: %w", err)
	}

	var result []domain.EABankTransaktion
	for _, stmt := range doc.BkToCstmrStmt.Stmt {
		iban := stmt.Acct.Id.IBAN
		for _, entry := range stmt.Ntry {
			amt, err := strconv.ParseFloat(strings.TrimSpace(entry.Amount.Value), 64)
			if err != nil {
				continue
			}
			if entry.CreditDebit == "DBIT" {
				amt = -amt
			}
			buchDatum, err := time.Parse("2006-01-02", strings.TrimSpace(entry.BookgDt.Dt))
			if err != nil {
				continue
			}
			t := domain.EABankTransaktion{
				EegID:         eegID,
				ImportFormat:  "CAMT053",
				KontoIBAN:     iban,
				Buchungsdatum: buchDatum,
				Betrag:        amt,
				Waehrung:      entry.Amount.Ccy,
				Referenz:      entry.AcctSvcrRef,
				MatchStatus:   "offen",
			}
			if valStr := strings.TrimSpace(entry.ValDt.Dt); valStr != "" {
				if vt, err := time.Parse("2006-01-02", valStr); err == nil {
					t.Valutadatum = &vt
				}
			}
			if len(entry.NtryDtls.TxDtls) > 0 {
				td := entry.NtryDtls.TxDtls[0]
				t.Verwendungszweck = td.RmtInf.Ustrd
				if t.Referenz == "" {
					t.Referenz = td.Refs.EndToEndId
				}
				if td.RltdPties.Dbtr.Nm != "" {
					t.AuftraggeberEmpfaenger = td.RltdPties.Dbtr.Nm
				} else if td.RltdPties.Cdtr.Nm != "" {
					t.AuftraggeberEmpfaenger = td.RltdPties.Cdtr.Nm
				}
			}
			result = append(result, t)
		}
	}
	return result, nil
}
