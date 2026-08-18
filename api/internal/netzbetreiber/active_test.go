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
// unrecognized Marktpartner-ID is still returned, using the raw ID as Name,
// and is flagged Unresolved so callers know not to trust it blindly.
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
	if !got[0].Unresolved {
		t.Errorf("expected Unresolved=true for unknown prefix, got %+v", got[0])
	}
}

// TestResolveRoutingID_KnownPrefix verifies that a registered EC-Nummer
// resolves to itself with ok=true.
func TestResolveRoutingID_KnownPrefix(t *testing.T) {
	id, ok := netzbetreiber.ResolveRoutingID("AT001000123456789012345678")
	if !ok || id != "AT001000" {
		t.Errorf("expected (AT001000, true), got (%q, %v)", id, ok)
	}
}

// TestResolveRoutingID_Override verifies that a documented prefix override
// (e.g. AT008230 → AT008000, Energienetze Steiermark Verteilnetzbereich-Code)
// resolves to the actual target EC-Nummer, not the literal Zählpunkt prefix.
func TestResolveRoutingID_Override(t *testing.T) {
	id, ok := netzbetreiber.ResolveRoutingID("AT0082300801000000000000000089806")
	if !ok || id != "AT008000" {
		t.Errorf("expected (AT008000, true), got (%q, %v)", id, ok)
	}
}

// TestResolveRoutingID_UnknownPrefix verifies that a prefix with neither a
// registry entry nor an override is reported as unresolved (ok=false), even
// though the raw prefix is still returned for error messages.
func TestResolveRoutingID_UnknownPrefix(t *testing.T) {
	id, ok := netzbetreiber.ResolveRoutingID("AT999999123456789012345678")
	if ok {
		t.Errorf("expected ok=false for unknown prefix, got id=%q ok=true", id)
	}
	if id != "AT999999" {
		t.Errorf("expected raw prefix AT999999 to still be returned, got %q", id)
	}
}

// TestResolveRoutingID_TooShort verifies that a Zählpunkt shorter than 8
// chars is rejected outright.
func TestResolveRoutingID_TooShort(t *testing.T) {
	if _, ok := netzbetreiber.ResolveRoutingID("AT0010"); ok {
		t.Errorf("expected ok=false for too-short Zählpunkt")
	}
}

// TestActiveFromMeterPoints_OverrideDedup verifies that two different
// Zählpunkt-prefixes resolving to the same real Netzbetreiber via an
// override collapse into a single entry, keyed on the resolved ID.
func TestActiveFromMeterPoints_OverrideDedup(t *testing.T) {
	mps := []domain.MeterPoint{
		mp("AT0082300801000000000000000089806", false), // resolves to AT008000 via override
		mp("AT0080000801000000000000000012345", false), // literal AT008000
	}
	got := netzbetreiber.ActiveFromMeterPoints(mps)
	if len(got) != 1 {
		t.Fatalf("expected 1 deduped entry, got %d: %+v", len(got), got)
	}
	if got[0].ID != "AT008000" || got[0].Name != "Energienetze Steiermark" || got[0].Unresolved {
		t.Errorf("expected resolved AT008000 = Energienetze Steiermark, got %+v", got[0])
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
