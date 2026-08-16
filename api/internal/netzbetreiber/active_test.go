package netzbetreiber_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/netzbetreiber"
)

func mp(zaehlpunkt string, abgemeldet bool) domain.MeterPoint {
	m := domain.MeterPoint{ID: uuid.New(), Zaehlpunkt: zaehlpunkt}
	if abgemeldet {
		t := time.Now()
		m.AbgemeldetAm = &t
	}
	return m
}

// TestActiveFromMeterPoints_DedupAndFilter verifies that inactive meter points
// are excluded and multiple active meter points at the same Netzbetreiber are
// deduplicated to a single entry.
func TestActiveFromMeterPoints_DedupAndFilter(t *testing.T) {
	mps := []domain.MeterPoint{
		mp("AT001000123456789012345678", false), // active, Wiener Netze
		mp("AT001000999999999999999999", false), // active, same NB (dedup)
		mp("AT002000123456789012345678", false), // active, Netz NÖ
		mp("AT003000123456789012345678", true),  // inactive — excluded
	}
	got := netzbetreiber.ActiveFromMeterPoints(mps)
	if len(got) != 2 {
		t.Fatalf("expected 2 distinct active Netzbetreiber, got %d: %+v", len(got), got)
	}
	byID := map[string]netzbetreiber.Info{}
	for _, info := range got {
		byID[info.ID] = info
	}
	if info, ok := byID["AT001000"]; !ok || info.Name != "Wiener Netze" {
		t.Errorf("expected AT001000 = Wiener Netze, got %+v (ok=%v)", info, ok)
	}
	if info, ok := byID["AT002000"]; !ok || info.Name != "Netz NÖ" {
		t.Errorf("expected AT002000 = Netz NÖ, got %+v (ok=%v)", info, ok)
	}
	if _, ok := byID["AT003000"]; ok {
		t.Errorf("expected AT003000 (inactive) to be excluded")
	}
}

// TestActiveFromMeterPoints_UnknownPrefixFallsBackToID verifies that an
// unrecognized Marktpartner-ID is still returned, using the raw ID as Name.
func TestActiveFromMeterPoints_UnknownPrefixFallsBackToID(t *testing.T) {
	mps := []domain.MeterPoint{
		mp("AT999999123456789012345678", false),
	}
	got := netzbetreiber.ActiveFromMeterPoints(mps)
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].ID != "AT999999" || got[0].Name != "AT999999" {
		t.Errorf("expected fallback Info{ID: AT999999, Name: AT999999}, got %+v", got[0])
	}
}

// TestActiveFromMeterPoints_ShortZaehlpunktIgnored verifies that meter points
// with a Zählpunkt shorter than 8 chars are skipped rather than causing a panic.
func TestActiveFromMeterPoints_ShortZaehlpunktIgnored(t *testing.T) {
	mps := []domain.MeterPoint{
		mp("AT0010", false),
		mp("", false),
	}
	got := netzbetreiber.ActiveFromMeterPoints(mps)
	if len(got) != 0 {
		t.Errorf("expected 0 entries for short/empty Zählpunkte, got %d: %+v", len(got), got)
	}
}

// TestActiveFromMeterPoints_Empty verifies that an empty input returns an
// empty (non-nil-panicking) result.
func TestActiveFromMeterPoints_Empty(t *testing.T) {
	got := netzbetreiber.ActiveFromMeterPoints(nil)
	if len(got) != 0 {
		t.Errorf("expected 0 entries, got %d", len(got))
	}
}

// TestActiveFromMeterPoints_SortedByName verifies the result is sorted
// alphabetically by Name, independent of input order.
func TestActiveFromMeterPoints_SortedByName(t *testing.T) {
	mps := []domain.MeterPoint{
		mp("AT009000123456789012345678", false), // Netz Burgenland
		mp("AT001000123456789012345678", false), // Wiener Netze
		mp("AT002000123456789012345678", false), // Netz NÖ
	}
	got := netzbetreiber.ActiveFromMeterPoints(mps)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Name > got[i].Name {
			t.Errorf("result not sorted by Name: %q before %q", got[i-1].Name, got[i].Name)
		}
	}
}
