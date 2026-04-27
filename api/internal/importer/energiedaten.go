package importer

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// colType identifies which value a column contains.
type colType int

const (
	colTypeUnknown   colType = iota
	colTypeTotal             // "Gesamtverbrauch" / "Gesamte gemeinschaftliche Erzeugung"
	colTypeCommunity         // "Anteil gemeinschaftliche" (consumption only)
	colTypeSelf              // "Eigendeckung gemeinschaftliche" but NOT "aus erneuerbarer"
	colTypeResidual          // "Restüberschuss bei EG und je ZP" = grid export for generation meters
)

func classifyColumn(header string) colType {
	h := strings.ToLower(header)
	switch {
	// ── Consumption columns ───────────────────────────────────────────────────
	case strings.Contains(h, "gesamtverbrauch"):
		return colTypeTotal
	case strings.Contains(h, "anteil gemeinschaftliche"):
		return colTypeCommunity
	case strings.Contains(h, "eigendeckung gemeinschaftliche") && !strings.Contains(h, "aus erneuerbarer"):
		return colTypeSelf
	// ── Generation columns ────────────────────────────────────────────────────
	// "Gesamte gemeinschaftliche Erzeugung" = total physical generation at this meter
	case strings.Contains(h, "gesamte gemeinschaftliche erzeugung"):
		return colTypeTotal
	// "Restüberschuss bei EG und je ZP" = grid export (Netzanteil); community = total - residual
	case strings.Contains(h, "restüberschuss bei eg"):
		return colTypeResidual
	// "Erzeugung lt. Messung entsprechend dem Teilnahmefaktor" = total × factor, NOT community share
	// → ignored: community is derived from total - residual instead
	case strings.Contains(h, "erzeugung lt. messung"):
		return colTypeUnknown
	default:
		return colTypeUnknown
	}
}

// ParseEnergieDaten parses an energy data XLSX file.
// Works for both Format A (TEST_EEG_Report) and Format B (RC105970).
func ParseEnergieDaten(path string) ([]domain.EnergyRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Energiedaten")
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	// Step 1: find meter row and metercode row
	meterRowIdx := -1
	metercodeRowIdx := -1
	directionRowIdx := -1

	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		col0 := strings.ToLower(strings.TrimSpace(row[0]))
		if meterRowIdx < 0 && strings.Contains(col0, "meteringp") {
			meterRowIdx = i
		}
		if metercodeRowIdx < 0 && strings.Contains(col0, "meterc") {
			metercodeRowIdx = i
		}
		if directionRowIdx < 0 && strings.Contains(col0, "direction") {
			directionRowIdx = i
		}
	}

	if meterRowIdx < 0 {
		return nil, fmt.Errorf("meter row not found")
	}
	if metercodeRowIdx < 0 {
		return nil, fmt.Errorf("metercode row not found")
	}

	meterRow := padRow(rows[meterRowIdx])
	metercodeRow := padRow(rows[metercodeRowIdx])

	// Step 2: build col → meter_id mapping (skip "MM", "TOTAL", empty)
	colToMeter := map[int]string{}
	currentMeter := ""
	for i := 1; i < len(meterRow); i++ {
		v := strings.TrimSpace(meterRow[i])
		if v == "" || strings.ToUpper(v) == "MM" || strings.ToUpper(v) == "TOTAL" {
			// Carry currentMeter for adjacent "MM" columns in Format B
			if v == "" || strings.ToUpper(v) == "TOTAL" {
				continue
			}
			// "MM" col: keep same meter
			if currentMeter != "" {
				colToMeter[i] = currentMeter
			}
			continue
		}
		currentMeter = v
		colToMeter[i] = currentMeter
	}

	// Step 3: build col → colType mapping
	colToType := map[int]colType{}
	for i := 1; i < len(metercodeRow); i++ {
		v := strings.TrimSpace(metercodeRow[i])
		if v == "" || strings.ToUpper(v) == "MM" {
			continue
		}
		ct := classifyColumn(v)
		if ct != colTypeUnknown {
			colToType[i] = ct
		}
	}

	// Step 4: build direction map per meter
	directionByMeter := map[string]string{}
	if directionRowIdx >= 0 {
		dirRow := padRow(rows[directionRowIdx])
		for i := 1; i < len(dirRow); i++ {
			meter, ok := colToMeter[i]
			if !ok {
				continue
			}
			v := strings.TrimSpace(dirRow[i])
			if v != "" {
				directionByMeter[meter] = v
			}
		}
	}

	// Step 5: find first data row — first row where col[0] is a parseable datetime
	dataStartIdx := -1
	for i := metercodeRowIdx + 1; i < len(rows); i++ {
		if len(rows[i]) == 0 {
			continue
		}
		if _, err := parseDateTime(rows[i][0]); err == nil {
			dataStartIdx = i
			break
		}
	}
	if dataStartIdx < 0 {
		return nil, fmt.Errorf("no data rows found")
	}

	// Step 6: collect which cols we actually care about per meter
	// For each meter we want: total, community, self, residual columns
	type meterCols struct {
		total    int
		community int
		self     int
		residual int // grid export for generation meters; -1 if absent
	}
	meterColMap := map[string]*meterCols{}
	for col, meter := range colToMeter {
		ct, ok := colToType[col]
		if !ok {
			continue
		}
		mc := meterColMap[meter]
		if mc == nil {
			mc = &meterCols{total: -1, community: -1, self: -1, residual: -1}
			meterColMap[meter] = mc
		}
		switch ct {
		case colTypeTotal:
			if mc.total < 0 {
				mc.total = col
			}
		case colTypeCommunity:
			if mc.community < 0 {
				mc.community = col
			}
		case colTypeSelf:
			if mc.self < 0 {
				mc.self = col
			}
		case colTypeResidual:
			if mc.residual < 0 {
				mc.residual = col
			}
		}
	}

	// Step 7: parse data rows
	var result []domain.EnergyRow
	for i := dataStartIdx; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		ts, err := parseDateTime(strings.TrimSpace(row[0]))
		if err != nil {
			continue // skip summary/header rows mixed in
		}

		paddedRow := padRow(row)
		for meter, mc := range meterColMap {
			total := getFloat(paddedRow, mc.total)
			community := getFloat(paddedRow, mc.community)
			self := getFloat(paddedRow, mc.self)

			// For generation meters: derive community as total − residual (grid export).
			// This gives the actual EEG community share per 15-min interval.
			if mc.residual >= 0 {
				residual := getFloat(paddedRow, mc.residual)
				community = total - residual
				if community < 0 {
					community = 0
				}
			}

			// Only emit row if at least one value is present
			if mc.total < 0 && mc.community < 0 && mc.self < 0 && mc.residual < 0 {
				continue
			}

			result = append(result, domain.EnergyRow{
				MeterID:     meter,
				Ts:          ts,
				WhTotal:     total,
				WhCommunity: community,
				WhSelf:      self,
			})
		}
		_ = directionByMeter // used for future direction-aware logic
	}

	return result, nil
}

// parseDateTime parses timestamps in multiple formats.
func parseDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"02.01.2006 15:04:05",
		"02.01.2006 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2.1.2006 15:04:05",
		"2.1.2006 15:04",
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as datetime", s)
}

func padRow(row []string) []string {
	return row // excelize already returns nil for missing cells, but we return as-is
}

func getFloat(row []string, col int) float64 {
	if col < 0 || col >= len(row) {
		return 0
	}
	v := strings.TrimSpace(row[col])
	// Format B has "L1" quality suffix — ignore it
	if strings.Contains(v, " ") {
		v = strings.Fields(v)[0]
	}
	// Remove thousands separator commas (Format A: "1,234.567")
	v = strings.ReplaceAll(v, ",", "")
	f, _ := strconv.ParseFloat(v, 64)
	return f
}
