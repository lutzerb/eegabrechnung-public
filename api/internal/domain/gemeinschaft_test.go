package domain

import "testing"

func TestMarktpartnerPrefix(t *testing.T) {
	want := map[string]string{"EEG": "RC", "GEA": "GC", "BEG": "CC", "P2P": "PP", "EVA": "SC"}
	for typ, prefix := range want {
		got, ok := MarktpartnerPrefix(typ)
		if !ok || got != prefix {
			t.Errorf("MarktpartnerPrefix(%q) = %q, %v; want %q", typ, got, ok, prefix)
		}
	}
	if ValidGemeinschaftTyp("XX") || ValidGemeinschaftTyp("") {
		t.Error("unknown type must be invalid")
	}
}

func TestAllowsMultiNetzbetreiber(t *testing.T) {
	for typ, want := range map[string]bool{"EEG": false, "GEA": false, "": false, "BEG": true, "P2P": true, "EVA": true} {
		if got := AllowsMultiNetzbetreiber(typ); got != want {
			t.Errorf("AllowsMultiNetzbetreiber(%q) = %v, want %v", typ, got, want)
		}
	}
}
