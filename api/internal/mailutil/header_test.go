package mailutil

import "testing"

func TestSanitizeHeaderValueStripsCRLF(t *testing.T) {
	in := "Hans\r\nBcc: victim@example.com"
	got := SanitizeHeaderValue(in)
	want := "HansBcc: victim@example.com"
	if got != want {
		t.Fatalf("SanitizeHeaderValue = %q, want %q", got, want)
	}
}

func TestHeadersNoInjection(t *testing.T) {
	h := Headers("kontakt@eeg.at", "member@example.com\r\nBcc: evil@x.at",
		"IBAN-Änderung: Hans\r\nBcc: evil@x.at")
	// The security property: the attacker's CRLF must not survive, so the block
	// contains exactly the three header lines we wrote (From/To/Subject) — no
	// injected fourth line. "Bcc: …" may remain as inert text inside the To value,
	// but never as its own CRLF-preceded header.
	if n := countCRLF(h); n != 3 {
		t.Fatalf("expected exactly 3 header lines, got %d in %q", n, h)
	}
	// Every CR and LF in the block must belong to one of our 3 line terminators —
	// i.e. exactly 3 of each, none contributed by the attacker's payload.
	if cr, lf := countRune(h, '\r'), countRune(h, '\n'); cr != 3 || lf != 3 {
		t.Fatalf("stray CR/LF survived: cr=%d lf=%d in %q", cr, lf, h)
	}
}

func countRune(s string, r byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == r {
			n++
		}
	}
	return n
}

func TestEncodeSubjectUmlaut(t *testing.T) {
	got := EncodeSubject("Willkommen zur Energiegemeinschaft Grünwald")
	if !contains(got, "=?utf-8?q?") {
		t.Fatalf("expected RFC2047 encoded subject, got %q", got)
	}
	if contains(got, "\r") || contains(got, "\n") {
		t.Fatalf("subject contains raw CR/LF: %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func countCRLF(s string) int {
	n := 0
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '\r' && s[i+1] == '\n' {
			n++
		}
	}
	return n
}
