package importer

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func writeInternalStammdaten(t *testing.T, zpRow []any) string {
	t.Helper()
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Mitglieder")
	f.NewSheet("Zählpunkte")
	member := []any{"Mitglieds-Nr", "Name1", "Name2", "E-Mail", "IBAN", "Rolle", "Straße", "PLZ", "Ort"}
	memberData := []any{"1", "Max", "Muster", "m@example.at", "", "privat", "Hauptstr. 1", "2700", "Wiener Neustadt"}
	zpHeader := []any{"Mitglieds-Nr", "Zählpunkt", "Energierichtung", "Verteilungsmodell", "Zuteilung %", "Status", "Registriert seit"}
	for sheet, rows := range map[string][][]any{
		"Mitglieder": {member, memberData},
		"Zählpunkte": {zpHeader, zpRow},
	} {
		for r, row := range rows {
			cell, _ := excelize.CoordinatesToCellName(1, r+1)
			if err := f.SetSheetRow(sheet, cell, &row); err != nil {
				t.Fatal(err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "stammdaten.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseStammdaten_MeterPointAddressColumns(t *testing.T) {
	path := writeInternalStammdaten(t, []any{
		"1", "AT0081000000000000000000000000001", "CONSUMPTION", "DYNAMIC", 100, "ACTIVATED", "2025-01-01",
		"Feldweg 5", "2751", "Hochwolkersdorf",
	})
	rows, err := ParseStammdaten(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.MPStrasse != "Feldweg 5" || r.MPPlz != "2751" || r.MPOrt != "Hochwolkersdorf" {
		t.Errorf("unexpected meter point address: %q %q %q", r.MPStrasse, r.MPPlz, r.MPOrt)
	}
	if r.Strasse != "Hauptstr. 1" {
		t.Errorf("member address must stay separate, got %q", r.Strasse)
	}
}

func TestParseStammdaten_MeterPointAddressColumnsAbsent(t *testing.T) {
	path := writeInternalStammdaten(t, []any{
		"1", "AT0081000000000000000000000000001", "CONSUMPTION", "DYNAMIC", 100, "ACTIVATED", "2025-01-01",
	})
	rows, err := ParseStammdaten(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].MPStrasse != "" || rows[0].MPPlz != "" || rows[0].MPOrt != "" {
		t.Errorf("expected empty meter point address for legacy file, got %+v", rows[0])
	}
}
