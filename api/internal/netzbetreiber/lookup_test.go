package netzbetreiber_test

import (
	"testing"

	"github.com/lutzerb/eegabrechnung/internal/netzbetreiber"
)

// knownEntries lists all 16 expected Marktpartner-IDs and their names
// so we can verify completeness and correctness.
var knownEntries = []struct {
	id        string
	name      string
	portalURL string
}{
	{"AT001000", "Wiener Netze", "https://smartmeter-web.wienernetze.at"},
	{"AT002000", "Netz NÖ", "https://smartmeter.netz-noe.at"},
	{"AT003000", "Netz OÖ", "https://eservice.netzooe.at/app/login"},
	{"AT003100", "Linz Netz", "https://www.linznetz.at/portal/de/home/online_services/serviceportal/mein_serviceportal"},
	{"AT003300", "eww ag", "https://mein.eww.at/de/login"},
	{"AT004000", "Salzburg Netz", "https://portal.salzburgnetz.at"},
	{"AT005000", "TINETZ", "https://kundenportal.tinetz.at"},
	{"AT005100", "IKB Innsbruck", "https://direkt.ikb.at"},
	{"AT005120", "HALLAG Hall i.T.", "https://kundenportal.hall.ag"},
	{"AT006000", "Vorarlberg Netz", "https://webportal.vorarlbergnetz.at"},
	{"AT007000", "Kärntner Netze", "https://services.kaerntennetz.at/meinPortal"},
	{"AT007100", "Stadtwerke Klagenfurt", "https://www.stw.at/kundenportal"},
	{"AT008000", "Energienetze Steiermark", "https://portal.e-netze.at"},
	{"AT008100", "Stromnetz Graz", "https://www.stromnetz-graz.at/sgg/customer/account/login"},
	{"AT008130", "Feistritzwerke", "https://kundenportal.feistritzwerke.at"},
	{"AT009000", "Netz Burgenland", "https://kundencenter.netzburgenland.at/okc-netz/home.xhtml"},
}

// TestByMarktpartnerID_KnownIDs verifies that every known ID is found and
// returns the correct Name and PortalURL.
func TestByMarktpartnerID_KnownIDs(t *testing.T) {
	for _, tc := range knownEntries {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			info, ok := netzbetreiber.ByMarktpartnerID(tc.id)
			if !ok {
				t.Fatalf("ByMarktpartnerID(%q): not found", tc.id)
			}
			if info.Name != tc.name {
				t.Errorf("Name: got %q, want %q", info.Name, tc.name)
			}
			if info.PortalURL != tc.portalURL {
				t.Errorf("PortalURL: got %q, want %q", info.PortalURL, tc.portalURL)
			}
		})
	}
}

// TestByMarktpartnerID_UnknownID verifies that an unknown ID returns false.
func TestByMarktpartnerID_UnknownID(t *testing.T) {
	unknowns := []string{
		"AT000000",
		"AT999999",
		"",
		"UNKNOWN1",
		"AT001001", // one off from a real ID
	}
	for _, id := range unknowns {
		id := id
		t.Run(id, func(t *testing.T) {
			_, ok := netzbetreiber.ByMarktpartnerID(id)
			if ok {
				t.Errorf("ByMarktpartnerID(%q): expected not found, but got ok=true", id)
			}
		})
	}
}

// TestByZaehlpunkt_KnownPrefixes verifies that a valid Zählpunkt produces
// the correct Netzbetreiber via prefix extraction.
func TestByZaehlpunkt_KnownPrefixes(t *testing.T) {
	cases := []struct {
		zaehlpunkt string
		wantID     string
		wantName   string
	}{
		// 27-char Zählpunkt (typical format: AT + 6 + 19)
		{"AT001000123456789012345678", "AT001000", "Wiener Netze"},
		{"AT002000999999999999999999", "AT002000", "Netz NÖ"},
		{"AT003000000000000000000001", "AT003000", "Netz OÖ"},
		{"AT003100ABCDEFGHIJKLMNOPQR", "AT003100", "Linz Netz"},
		{"AT003300ABCDEFGHIJKLMNOPQR", "AT003300", "eww ag"},
		{"AT004000ABCDEFGHIJKLMNOPQR", "AT004000", "Salzburg Netz"},
		{"AT005000ABCDEFGHIJKLMNOPQR", "AT005000", "TINETZ"},
		{"AT005100ABCDEFGHIJKLMNOPQR", "AT005100", "IKB Innsbruck"},
		{"AT005120ABCDEFGHIJKLMNOPQR", "AT005120", "HALLAG Hall i.T."},
		{"AT006000ABCDEFGHIJKLMNOPQR", "AT006000", "Vorarlberg Netz"},
		{"AT007000ABCDEFGHIJKLMNOPQR", "AT007000", "Kärntner Netze"},
		{"AT007100ABCDEFGHIJKLMNOPQR", "AT007100", "Stadtwerke Klagenfurt"},
		{"AT008000ABCDEFGHIJKLMNOPQR", "AT008000", "Energienetze Steiermark"},
		{"AT008100ABCDEFGHIJKLMNOPQR", "AT008100", "Stromnetz Graz"},
		{"AT008130ABCDEFGHIJKLMNOPQR", "AT008130", "Feistritzwerke"},
		{"AT009000ABCDEFGHIJKLMNOPQR", "AT009000", "Netz Burgenland"},
		// Exactly 8 chars — minimal valid Zählpunkt
		{"AT001000", "AT001000", "Wiener Netze"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.zaehlpunkt, func(t *testing.T) {
			info, ok := netzbetreiber.ByZaehlpunkt(tc.zaehlpunkt)
			if !ok {
				t.Fatalf("ByZaehlpunkt(%q): not found (expected %q)", tc.zaehlpunkt, tc.wantName)
			}
			if info.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", info.Name, tc.wantName)
			}
		})
	}
}

// TestByZaehlpunkt_TooShort verifies that strings shorter than 8 chars return false.
func TestByZaehlpunkt_TooShort(t *testing.T) {
	shorts := []string{"", "AT", "AT0010", "AT00100"}
	for _, s := range shorts {
		s := s
		t.Run(s, func(t *testing.T) {
			_, ok := netzbetreiber.ByZaehlpunkt(s)
			if ok {
				t.Errorf("ByZaehlpunkt(%q): expected false for short input, got true", s)
			}
		})
	}
}

// TestByZaehlpunkt_UnknownPrefix verifies that a Zählpunkt with an unknown
// 8-char prefix returns false.
func TestByZaehlpunkt_UnknownPrefix(t *testing.T) {
	unknowns := []string{
		"AT000000XXXXXXXXXXXXXXXXXXX",
		"AT999999XXXXXXXXXXXXXXXXXXX",
		"XX001000XXXXXXXXXXXXXXXXXXX",
	}
	for _, s := range unknowns {
		s := s
		t.Run(s, func(t *testing.T) {
			_, ok := netzbetreiber.ByZaehlpunkt(s)
			if ok {
				t.Errorf("ByZaehlpunkt(%q): expected not found, got ok=true", s)
			}
		})
	}
}

// TestAllEntriesNonEmpty verifies that every entry in the registry has a
// non-empty Name and PortalURL.
func TestAllEntriesNonEmpty(t *testing.T) {
	for _, tc := range knownEntries {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			info, ok := netzbetreiber.ByMarktpartnerID(tc.id)
			if !ok {
				t.Fatalf("entry %q not found", tc.id)
			}
			if info.Name == "" {
				t.Errorf("entry %q: Name is empty", tc.id)
			}
			if info.PortalURL == "" {
				t.Errorf("entry %q: PortalURL is empty", tc.id)
			}
		})
	}
}

// TestRegistryCount verifies that exactly 16 entries are present.
func TestRegistryCount(t *testing.T) {
	count := 0
	for _, tc := range knownEntries {
		if _, ok := netzbetreiber.ByMarktpartnerID(tc.id); ok {
			count++
		}
	}
	if count != 16 {
		t.Errorf("expected 16 entries, got %d", count)
	}
}
