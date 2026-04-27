package handler

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/xuri/excelize/v2"
)

func generateAnnualReportXLSX(eegName string, date, from, to time.Time, members []domain.AnnualReportMember) ([]byte, error) {
	f := excelize.NewFile()

	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
	})
	styleSubHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Italic: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#F8FAFC"}, Pattern: 1},
	})
	styleNum, _ := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0.00
	styleDate, _ := f.NewStyle(&excelize.Style{NumFmt: 14}) // dd/mm/yyyy

	periodLabel := fmt.Sprintf("%s – %s",
		from.Format("02.01.2006"),
		to.AddDate(0, 0, -1).Format("02.01.2006"),
	)

	// ── Sheet 1: Mitglieder (reference date) ──────────────────────────────
	sheet1 := "Mitglieder"
	f.SetSheetName("Sheet1", sheet1)

	s1headers := []string{
		"Nr.", "Name", "E-Mail", "Straße", "PLZ", "Ort",
		"Mitgliedstyp", "Beitritt", "Austritt",
		"Zählpunkte",
	}
	for col, h := range s1headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet1, cell, h)
		f.SetCellStyle(sheet1, cell, cell, styleHeader)
	}

	memberTypeLabel := map[string]string{
		"CONSUMER": "Verbraucher",
		"PRODUCER": "Erzeuger",
		"PROSUMER": "Prosumer",
	}

	for i, m := range members {
		row := i + 2
		typ := memberTypeLabel[m.MemberType]
		if typ == "" {
			typ = m.MemberType
		}
		zps := make([]string, 0, len(m.Zaehlpunkte))
		for _, zp := range m.Zaehlpunkte {
			dir := zp.Energierichtung
			if dir == "CONSUMPTION" {
				dir = "Bezug"
			} else if dir == "GENERATION" {
				dir = "Einspeisung"
			}
			zps = append(zps, fmt.Sprintf("%s (%s)", zp.Zaehlpunkt, dir))
		}

		setCell := func(col int, v interface{}) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet1, cell, v)
		}
		setCell(1, m.MitgliedsNr)
		setCell(2, m.Name)
		setCell(3, m.Email)
		setCell(4, m.Strasse)
		setCell(5, m.Plz)
		setCell(6, m.Ort)
		setCell(7, typ)
		if m.BeitrittsDatum != nil {
			cell, _ := excelize.CoordinatesToCellName(8, row)
			f.SetCellValue(sheet1, cell, m.BeitrittsDatum.Format("02.01.2006"))
			f.SetCellStyle(sheet1, cell, cell, styleDate)
		}
		if m.AustrittsDatum != nil {
			cell, _ := excelize.CoordinatesToCellName(9, row)
			f.SetCellValue(sheet1, cell, m.AustrittsDatum.Format("02.01.2006"))
			f.SetCellStyle(sheet1, cell, cell, styleDate)
		}
		setCell(10, strings.Join(zps, "\n"))
	}

	colWidths1 := []float64{8, 28, 30, 28, 8, 18, 14, 12, 12, 55}
	for i, w := range colWidths1 {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet1, col, col, w)
	}

	// Title row above header
	f.InsertRows(sheet1, 1, 2)
	f.SetCellValue(sheet1, "A1", fmt.Sprintf("Mitgliederliste — Stichtag %s — %s", date.Format("02.01.2006"), eegName))
	f.SetCellStyle(sheet1, "A1", "A1", styleSubHeader)
	f.MergeCell(sheet1, "A1", "J1")

	// ── Sheet 2: Energie & Abrechnung ────────────────────────────────────
	sheet2 := "Energie & Abrechnung"
	f.NewSheet(sheet2)

	s2headers := []string{
		"Nr.", "Name", "Typ",
		"Bezug gesamt kWh", "Einspeisung gesamt kWh", "Gemeinschaftsanteil kWh",
		"Rechnungen EUR", "Gutschriften EUR", "Anzahl Belege",
	}
	for col, h := range s2headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet2, cell, h)
		f.SetCellStyle(sheet2, cell, cell, styleHeader)
	}

	var totConsumption, totGeneration, totCommunity, totInvoiced, totCredited float64
	for i, m := range members {
		row := i + 2
		typ := memberTypeLabel[m.MemberType]
		if typ == "" {
			typ = m.MemberType
		}

		setNum := func(col int, v float64) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet2, cell, v)
			f.SetCellStyle(sheet2, cell, cell, styleNum)
		}
		setStr := func(col int, v interface{}) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet2, cell, v)
		}

		setStr(1, m.MitgliedsNr)
		setStr(2, m.Name)
		setStr(3, typ)
		setNum(4, m.WhConsumption)
		setNum(5, m.WhGeneration)
		setNum(6, m.WhCommunity)
		setNum(7, m.Invoiced)
		setNum(8, m.Credited)
		setStr(9, m.InvoiceCount)

		totConsumption += m.WhConsumption
		totGeneration += m.WhGeneration
		totCommunity += m.WhCommunity
		totInvoiced += m.Invoiced
		totCredited += m.Credited
	}

	// Totals row
	totRow := len(members) + 2
	styleTotal, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		NumFmt: 4,
	})
	setTot := func(col int, v float64) {
		cell, _ := excelize.CoordinatesToCellName(col, totRow)
		f.SetCellValue(sheet2, cell, v)
		f.SetCellStyle(sheet2, cell, cell, styleTotal)
	}
	totLbl, _ := excelize.CoordinatesToCellName(1, totRow)
	f.SetCellValue(sheet2, totLbl, "Gesamt")
	f.SetCellStyle(sheet2, totLbl, totLbl, styleTotal)
	setTot(4, totConsumption)
	setTot(5, totGeneration)
	setTot(6, totCommunity)
	setTot(7, totInvoiced)
	setTot(8, totCredited)

	colWidths2 := []float64{8, 28, 14, 22, 24, 24, 18, 18, 14}
	for i, w := range colWidths2 {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet2, col, col, w)
	}

	// Title row
	f.InsertRows(sheet2, 1, 2)
	f.SetCellValue(sheet2, "A1", fmt.Sprintf("Energie & Abrechnung — Zeitraum %s — %s", periodLabel, eegName))
	f.SetCellStyle(sheet2, "A1", "A1", styleSubHeader)
	f.MergeCell(sheet2, "A1", "I1")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
