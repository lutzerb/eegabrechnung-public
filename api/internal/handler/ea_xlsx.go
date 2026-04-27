package handler

import (
	"fmt"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/xuri/excelize/v2"
)

func EASaldenlisteXLSX(rows []domain.EASaldenlisteEintrag, von, bis *time.Time) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Saldenliste"
	f.SetSheetName("Sheet1", sheet)

	// Header row
	headers := []string{"Konto", "Name", "Typ", "Einnahmen (€)", "Ausgaben (€)", "Saldo (€)", "Buchungen"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for i, row := range rows {
		r := i + 2
		f.SetCellValue(sheet, cellName(1, r), row.Nummer)
		f.SetCellValue(sheet, cellName(2, r), row.Name)
		f.SetCellValue(sheet, cellName(3, r), row.Typ)
		f.SetCellFloat(sheet, cellName(4, r), row.Einnahmen, 2, 64)
		f.SetCellFloat(sheet, cellName(5, r), row.Ausgaben, 2, 64)
		f.SetCellFloat(sheet, cellName(6, r), row.Saldo, 2, 64)
		f.SetCellInt(sheet, cellName(7, r), row.AnzahlBuchungen)
	}

	if von != nil || bis != nil {
		subtitle := "Zeitraum: "
		if von != nil {
			subtitle += von.Format("02.01.2006")
		} else {
			subtitle += "Beginn"
		}
		subtitle += " – "
		if bis != nil {
			subtitle += bis.Format("02.01.2006")
		} else {
			subtitle += "Ende"
		}
		f.SetCellValue(sheet, "A1", subtitle)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EAJahresabschlussXLSX(ja *domain.EAJahresabschluss) ([]byte, error) {
	f := excelize.NewFile()
	sheet := fmt.Sprintf("Jahresabschluss %d", ja.Jahr)
	f.SetSheetName("Sheet1", sheet)

	row := 1
	f.SetCellValue(sheet, cellName(1, row), fmt.Sprintf("E/A-Jahresabschluss %d", ja.Jahr))
	row += 2

	// Einnahmen section
	f.SetCellValue(sheet, cellName(1, row), "EINNAHMEN")
	row++
	f.SetCellValue(sheet, cellName(1, row), "Konto")
	f.SetCellValue(sheet, cellName(2, row), "Bezeichnung")
	f.SetCellValue(sheet, cellName(3, row), "Betrag (€)")
	row++
	for _, e := range ja.Einnahmen {
		f.SetCellValue(sheet, cellName(1, row), e.Nummer)
		f.SetCellValue(sheet, cellName(2, row), e.Name)
		f.SetCellFloat(sheet, cellName(3, row), e.Einnahmen, 2, 64)
		row++
	}
	f.SetCellValue(sheet, cellName(2, row), "Summe Einnahmen")
	f.SetCellFloat(sheet, cellName(3, row), ja.TotalEinnahmen, 2, 64)
	row += 2

	// Ausgaben section
	f.SetCellValue(sheet, cellName(1, row), "AUSGABEN")
	row++
	f.SetCellValue(sheet, cellName(1, row), "Konto")
	f.SetCellValue(sheet, cellName(2, row), "Bezeichnung")
	f.SetCellValue(sheet, cellName(3, row), "Betrag (€)")
	row++
	for _, e := range ja.Ausgaben {
		f.SetCellValue(sheet, cellName(1, row), e.Nummer)
		f.SetCellValue(sheet, cellName(2, row), e.Name)
		f.SetCellFloat(sheet, cellName(3, row), e.Ausgaben, 2, 64)
		row++
	}
	f.SetCellValue(sheet, cellName(2, row), "Summe Ausgaben")
	f.SetCellFloat(sheet, cellName(3, row), ja.TotalAusgaben, 2, 64)
	row += 2

	// Überschuss
	f.SetCellValue(sheet, cellName(2, row), "ÜBERSCHUSS / VERLUST")
	f.SetCellFloat(sheet, cellName(3, row), ja.Ueberschuss, 2, 64)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EABuchungenXLSX(buchungen []domain.EABuchung) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Journal"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Buchungsnr", "Zahlung", "Beleg-Datum", "Belegnr", "Beschreibung", "Konto", "Richtung", "Brutto (€)", "USt-Code", "USt (€)", "Netto (€)", "Gegenseite"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for i, b := range buchungen {
		r := i + 2
		f.SetCellValue(sheet, cellName(1, r), b.Buchungsnr)
		if b.ZahlungDatum != nil {
			f.SetCellValue(sheet, cellName(2, r), b.ZahlungDatum.Format("02.01.2006"))
		}
		if b.BelegDatum != nil {
			f.SetCellValue(sheet, cellName(3, r), b.BelegDatum.Format("02.01.2006"))
		}
		f.SetCellValue(sheet, cellName(4, r), b.Belegnr)
		f.SetCellValue(sheet, cellName(5, r), b.Beschreibung)
		if b.Konto != nil {
			f.SetCellValue(sheet, cellName(6, r), b.Konto.Nummer+" – "+b.Konto.Name)
		}
		f.SetCellValue(sheet, cellName(7, r), b.Richtung)
		f.SetCellFloat(sheet, cellName(8, r), b.BetragBrutto, 2, 64)
		f.SetCellValue(sheet, cellName(9, r), b.UstCode)
		f.SetCellFloat(sheet, cellName(10, r), b.UstBetrag, 2, 64)
		f.SetCellFloat(sheet, cellName(11, r), b.BetragNetto, 2, 64)
		f.SetCellValue(sheet, cellName(12, r), b.Gegenseite)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
