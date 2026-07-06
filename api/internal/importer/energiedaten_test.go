package importer

import (
	"testing"
	"time"
)

const (
	fixtureFormatA = "../../../tests/fixtures/TEST_EEG_Report_AT00999900000TE100100.xlsx"
	fixtureFormatB = "../../../tests/fixtures/RC105970_2026-01-01T00_00-2026-01-31T23_45.xlsx"
)

func TestParseEnergieDaten_FormatA_Basic(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatA)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format A) failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected rows, got 0")
	}
	t.Logf("Format A: %d rows parsed", len(rows))
}

func TestParseEnergieDaten_FormatA_KnownMeters(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatA)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format A) failed: %v", err)
	}

	// Known meters from inspection
	expectedMeters := []string{
		"AT0099990000000000000000000020100",
		"AT0099990000000000000000000020102",
		"AT0099990000000000000000000020104",
	}

	meterIDs := map[string]bool{}
	for _, r := range rows {
		meterIDs[r.MeterID] = true
	}

	for _, m := range expectedMeters {
		if !meterIDs[m] {
			t.Errorf("expected meter %q in results", m)
		}
	}
}

func TestParseEnergieDaten_FormatA_Timestamps(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatA)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format A) failed: %v", err)
	}

	// First data row should be 2023-01-01 00:00:00
	// Find earliest timestamp
	var earliest time.Time
	for _, r := range rows {
		if earliest.IsZero() || r.Ts.Before(earliest) {
			earliest = r.Ts
		}
	}
	expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if !earliest.Equal(expected) {
		t.Errorf("expected earliest timestamp %v, got %v", expected, earliest)
	}
}

func TestParseEnergieDaten_FormatA_Values(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatA)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format A) failed: %v", err)
	}

	// From inspection, first data row (01.01.2023 00:00:00) for AT0099990000000000000000000020100:
	// wh_total=0.004, wh_community=0.000727, wh_self=0.000727
	target := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	targetMeter := "AT0099990000000000000000000020100"

	for _, r := range rows {
		if r.MeterID == targetMeter && r.Ts.Equal(target) {
			// Check approximate values
			if abs(r.WhTotal-0.004) > 0.0001 {
				t.Errorf("expected wh_total=0.004, got %f", r.WhTotal)
			}
			if abs(r.WhCommunity-0.000727) > 0.0001 {
				t.Errorf("expected wh_community=0.000727, got %f", r.WhCommunity)
			}
			if abs(r.WhSelf-0.000727) > 0.0001 {
				t.Errorf("expected wh_self=0.000727, got %f", r.WhSelf)
			}
			return
		}
	}
	t.Errorf("row not found for meter %q at %v", targetMeter, target)
}

func TestParseEnergieDaten_FormatB_Basic(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatB)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format B) failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected rows, got 0")
	}
	t.Logf("Format B: %d rows parsed", len(rows))
}

func TestParseEnergieDaten_FormatB_KnownMeters(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatB)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format B) failed: %v", err)
	}

	expectedMeters := []string{
		"AT0020000000000000000000020091266",
		"AT0020000000000000000000100215718",
		"AT0020000000000000000000020089835",
		"AT0020000000000000000000020073384",
		"AT0020000000000000000000020091072",
	}

	meterIDs := map[string]bool{}
	for _, r := range rows {
		meterIDs[r.MeterID] = true
	}

	for _, m := range expectedMeters {
		if !meterIDs[m] {
			t.Errorf("expected meter %q in results", m)
		}
	}
}

func TestParseEnergieDaten_FormatB_Timestamps(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatB)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format B) failed: %v", err)
	}

	// First data row should be 2026-01-01 00:00
	var earliest time.Time
	for _, r := range rows {
		if earliest.IsZero() || r.Ts.Before(earliest) {
			earliest = r.Ts
		}
	}
	expected := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !earliest.Equal(expected) {
		t.Errorf("expected earliest timestamp %v, got %v", expected, earliest)
	}
}

func TestParseEnergieDaten_FormatB_Values(t *testing.T) {
	rows, err := ParseEnergieDaten(fixtureFormatB)
	if err != nil {
		t.Fatalf("ParseEnergieDaten (Format B) failed: %v", err)
	}

	// From inspection, first data row (01.01.2026 00:00) for AT0020000000000000000000020091266:
	// wh_total=0.05, wh_community=0 (col 5), wh_self=0 (col 7)
	target := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	targetMeter := "AT0020000000000000000000020091266"

	for _, r := range rows {
		if r.MeterID == targetMeter && r.Ts.Equal(target) {
			if abs(r.WhTotal-0.05) > 0.001 {
				t.Errorf("expected wh_total=0.05, got %f", r.WhTotal)
			}
			return
		}
	}
	t.Errorf("row not found for meter %q at %v", targetMeter, target)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
