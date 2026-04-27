package importer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ParseStammdaten parses a Stammdaten XLSX file.
// Returns a slice of StammdatenRow for ACTIVATED meter points.
//
// File layout (0-indexed rows):
//   0-5: header / marker rows — skip
//   6:   column headers
//   7-8: marker rows — skip
//   9+:  data rows (skip if col[0] starts with "[###")
func ParseStammdaten(path string) ([]domain.StammdatenRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows("EEG Stammdaten")
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	// Row 6 (index 6) contains headers — used to verify structure.
	// Data starts at row index 9 (row 10 in 1-based).
	var result []domain.StammdatenRow
	for i := 9; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		if strings.HasPrefix(row[0], "[###") {
			continue
		}

		r := parseStammdatenRow(row)

		// Only import ACTIVATED meter points
		if r.Zaehlpunktstatus != "ACTIVATED" {
			continue
		}

		result = append(result, r)
	}
	return result, nil
}

// Column indices for Stammdaten sheet.
const (
	colNetzbetreiber      = 0
	colGemeinschaftID     = 1
	colZaehlpunkt         = 11
	colEnergierichtung    = 12
	colVerteilungsmodell  = 17
	colZugeteilteMenuge   = 18
	colName1              = 20
	colName2              = 21
	colBusinessRole       = 23
	colIBAN               = 24
	colEmail              = 26
	colMitgliedsNr        = 28
	colZaehlpunktstatus   = 29
	colRegistriertSeit    = 31
)

func parseStammdatenRow(row []string) domain.StammdatenRow {
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
