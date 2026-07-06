package importer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ParseStammdaten parses a Stammdaten XLSX file.
// Automatically detects the format:
//   - Internal format (eegabrechnung export): two sheets "Mitglieder" + "Zählpunkte"
//   - EDA portal format: single sheet "EEG Stammdaten", header rows 0–8 skipped, data from row 9
func ParseStammdaten(path string) ([]domain.StammdatenRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if idx, _ := f.GetSheetIndex("Mitglieder"); idx != -1 {
		return parseInternalTwoSheetFormat(f)
	}

	rows, err := f.GetRows("EEG Stammdaten")
	if err != nil {
		return nil, fmt.Errorf("sheet 'EEG Stammdaten' nicht gefunden — bitte prüfen Sie das Dateiformat: %w", err)
	}
	return parseEDAFormat(rows)
}

// parseInternalTwoSheetFormat reads the eegabrechnung two-sheet format.
//
// Sheet "Mitglieder" columns (row 0 = headers, data from row 1):
//
//	Mitglieds-Nr(0) Name1(1) Name2(2) E-Mail(3) IBAN(4) Geschäftsrolle(5)
//	Straße(6) PLZ(7) Ort(8) UID-Nr(9) Status(10)
//
// Sheet "Zählpunkte" columns (row 0 = headers, data from row 1):
//
//	Mitglieds-Nr(0) Zählpunkt(1) Energierichtung(2) Verteilungsmodell(3)
//	Zuteilung %(4) Status(5) Registriert seit(6)
func parseInternalTwoSheetFormat(f *excelize.File) ([]domain.StammdatenRow, error) {
	memberRows, err := f.GetRows("Mitglieder")
	if err != nil {
		return nil, fmt.Errorf("sheet 'Mitglieder': %w", err)
	}

	get := func(row []string, idx int) string {
		if idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	type memberData struct {
		name1, name2, email, iban, businessRole, strasse, plz, ort string
	}
	members := make(map[string]memberData, len(memberRows))
	for i := 1; i < len(memberRows); i++ {
		row := memberRows[i]
		nr := get(row, 0)
		if nr == "" {
			continue
		}
		members[nr] = memberData{
			name1:        get(row, 1),
			name2:        get(row, 2),
			email:        get(row, 3),
			iban:         get(row, 4),
			businessRole: get(row, 5),
			strasse:      get(row, 6),
			plz:          get(row, 7),
			ort:          get(row, 8),
		}
	}

	zpRows, err := f.GetRows("Zählpunkte")
	if err != nil {
		return nil, fmt.Errorf("sheet 'Zählpunkte': %w", err)
	}

	var result []domain.StammdatenRow
	for i := 1; i < len(zpRows); i++ {
		row := zpRows[i]
		if len(row) == 0 {
			continue
		}
		mitgliedsNr := get(row, 0)
		zp := get(row, 1)
		if zp == "" {
			continue
		}
		zpStatus := get(row, 5)
		if zpStatus == "INACTIVE" || zpStatus == "DEACTIVATED" {
			continue
		}
		m, ok := members[mitgliedsNr]
		if !ok {
			continue
		}
		pct, _ := strconv.ParseFloat(get(row, 4), 64)
		result = append(result, domain.StammdatenRow{
			MitgliedsNr:         mitgliedsNr,
			Name1:               m.name1,
			Name2:               m.name2,
			Email:               m.email,
			IBAN:                m.iban,
			BusinessRole:        m.businessRole,
			Strasse:             m.strasse,
			Plz:                 m.plz,
			Ort:                 m.ort,
			Zaehlpunkt:          zp,
			Energierichtung:     get(row, 2),
			Verteilungsmodell:   get(row, 3),
			ZugeteilteMenugePct: pct,
			Zaehlpunktstatus:    zpStatus,
			RegistriertSeit:     get(row, 6),
		})
	}
	return result, nil
}

// parseEDAFormat reads the EDA portal Stammdaten export format.
// Rows 0–8 are header/marker rows (skipped); data starts at row 9.
func parseEDAFormat(rows [][]string) ([]domain.StammdatenRow, error) {
	var result []domain.StammdatenRow
	for i := 9; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		if strings.HasPrefix(row[0], "[###") {
			continue
		}
		r := parseEDARow(row)
		if r.Zaehlpunktstatus != "ACTIVATED" {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

// EDA portal column indices.
const (
	colNetzbetreiber     = 0
	colGemeinschaftID    = 1
	colZaehlpunkt        = 11
	colEnergierichtung   = 12
	colVerteilungsmodell = 17
	colZugeteilteMenuge  = 18
	colName1             = 20
	colName2             = 21
	colBusinessRole      = 23
	colIBAN              = 24
	colEmail             = 26
	colMitgliedsNr       = 28
	colZaehlpunktstatus  = 29
	colRegistriertSeit   = 31
)

func parseEDARow(row []string) domain.StammdatenRow {
	get := func(idx int) string {
		if idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	pct, _ := strconv.ParseFloat(get(colZugeteilteMenuge), 64)

	return domain.StammdatenRow{
		Netzbetreiber:       get(colNetzbetreiber),
		GemeinschaftID:      get(colGemeinschaftID),
		Zaehlpunkt:          get(colZaehlpunkt),
		Energierichtung:     get(colEnergierichtung),
		Verteilungsmodell:   get(colVerteilungsmodell),
		ZugeteilteMenugePct: pct,
		Name1:               get(colName1),
		Name2:               get(colName2),
		Email:               get(colEmail),
		IBAN:                get(colIBAN),
		BusinessRole:        get(colBusinessRole),
		MitgliedsNr:         get(colMitgliedsNr),
		Zaehlpunktstatus:    get(colZaehlpunktstatus),
		RegistriertSeit:     get(colRegistriertSeit),
	}
}
