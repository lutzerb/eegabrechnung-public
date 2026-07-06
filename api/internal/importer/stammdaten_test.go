package importer

import (
	"testing"
)

const stammdatenFixture = "../../../tests/fixtures/TE100200-Muster-Stammdatenimport.xlsx"

func TestParseStammdaten_BasicParse(t *testing.T) {
	rows, err := ParseStammdaten(stammdatenFixture)
	if err != nil {
		t.Fatalf("ParseStammdaten failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row, got 0")
	}
}

func TestParseStammdaten_OnlyActivated(t *testing.T) {
	rows, err := ParseStammdaten(stammdatenFixture)
	if err != nil {
		t.Fatalf("ParseStammdaten failed: %v", err)
	}
	for _, row := range rows {
		if row.Zaehlpunktstatus != "ACTIVATED" {
			t.Errorf("expected ACTIVATED, got %q for zaehlpunkt %q", row.Zaehlpunktstatus, row.Zaehlpunkt)
		}
	}
}

func TestParseStammdaten_FieldsPopulated(t *testing.T) {
	rows, err := ParseStammdaten(stammdatenFixture)
	if err != nil {
		t.Fatalf("ParseStammdaten failed: %v", err)
	}

	// Find the first row and verify key fields
	first := rows[0]
	if first.Netzbetreiber == "" {
		t.Error("expected Netzbetreiber to be non-empty")
	}
	if first.GemeinschaftID == "" {
		t.Error("expected GemeinschaftID to be non-empty")
	}
	if first.Zaehlpunkt == "" {
		t.Error("expected Zaehlpunkt to be non-empty")
	}
	if first.MitgliedsNr == "" {
		t.Error("expected MitgliedsNr to be non-empty")
	}
	if first.Name1 == "" {
		t.Error("expected Name1 to be non-empty")
	}
}

func TestParseStammdaten_KnownValues(t *testing.T) {
	rows, err := ParseStammdaten(stammdatenFixture)
	if err != nil {
		t.Fatalf("ParseStammdaten failed: %v", err)
	}

	// Find member "001"
	var found *struct {
		idx int
		row interface{}
	}
	for i, r := range rows {
		if r.MitgliedsNr == "001" {
			found = &struct {
				idx int
				row interface{}
			}{i, nil}
			_ = i

			// Verify known values for member 001
			if r.Netzbetreiber != "AT009999" {
				t.Errorf("expected Netzbetreiber 'AT009999', got %q", r.Netzbetreiber)
			}
			if r.GemeinschaftID != "AT00999900000TC100200000000000002" {
				t.Errorf("unexpected GemeinschaftID: %q", r.GemeinschaftID)
			}
			if r.Zaehlpunkt != "AT0099990000000000000000000020100" {
				t.Errorf("unexpected Zaehlpunkt: %q", r.Zaehlpunkt)
			}
			if r.Energierichtung != "CONSUMPTION" {
				t.Errorf("expected CONSUMPTION, got %q", r.Energierichtung)
			}
			if r.Name1 != "Max" {
				t.Errorf("expected Name1='Max', got %q", r.Name1)
			}
			if r.Name2 != "Mustermann" {
				t.Errorf("expected Name2='Mustermann', got %q", r.Name2)
			}
			if r.Email != "Max.Mustermann@example.at" {
				t.Errorf("expected email 'Max.Mustermann@example.at', got %q", r.Email)
			}
			break
		}
	}
	if found == nil {
		t.Error("member with MitgliedsNr '001' not found")
	}
}

func TestParseStammdaten_MultipleMembers(t *testing.T) {
	rows, err := ParseStammdaten(stammdatenFixture)
	if err != nil {
		t.Fatalf("ParseStammdaten failed: %v", err)
	}

	// Count unique members
	members := map[string]int{}
	for _, r := range rows {
		members[r.MitgliedsNr]++
	}

	// We know from inspection there are at least 5 unique ACTIVATED members
	if len(members) < 5 {
		t.Errorf("expected at least 5 unique members, got %d: %v", len(members), members)
	}
}

func TestParseStammdaten_MultipleZaehlpunkte(t *testing.T) {
	rows, err := ParseStammdaten(stammdatenFixture)
	if err != nil {
		t.Fatalf("ParseStammdaten failed: %v", err)
	}

	// Some members have multiple meter points (e.g., member "003" has 2)
	memberMeterPoints := map[string][]string{}
	for _, r := range rows {
		memberMeterPoints[r.MitgliedsNr] = append(memberMeterPoints[r.MitgliedsNr], r.Zaehlpunkt)
	}
	member003 := memberMeterPoints["003"]
	if len(member003) < 2 {
		t.Errorf("expected member '003' to have >= 2 meter points, got %d", len(member003))
	}
}
