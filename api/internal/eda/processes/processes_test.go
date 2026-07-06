package processes_test

import (
	"testing"

	"github.com/lutzerb/eegabrechnung/internal/eda/processes"
)

func TestVersion_KnownCodes(t *testing.T) {
	for _, code := range processes.All {
		t.Run(code, func(t *testing.T) {
			v, err := processes.Version(code)
			if err != nil {
				t.Fatalf("Version(%q): unexpected error: %v", code, err)
			}
			if v == "" {
				t.Errorf("Version(%q): got empty version", code)
			}
		})
	}
}

func TestVersion_UnknownCode(t *testing.T) {
	_, err := processes.Version("UNKNOWN_PROCESS")
	if err == nil {
		t.Error("expected error for unknown process code, got nil")
	}
}

func TestIsKnown(t *testing.T) {
	for _, code := range processes.All {
		if !processes.IsKnown(code) {
			t.Errorf("IsKnown(%q) = false, want true", code)
		}
	}
	if processes.IsKnown("NOT_A_PROCESS") {
		t.Error("IsKnown(NOT_A_PROCESS) = true, want false")
	}
}

func TestAllProcessCodes(t *testing.T) {
	expected := map[string]bool{
		processes.AnforderungECON: true,
		processes.AnforderungECOF: true,
		processes.AnforderungCPF:  true,
		processes.AnforderungECP:  true,
		processes.AnforderungECC:  true,
		processes.AnforderungCCMO: true,
		processes.AnforderungGN:   true,
		processes.AufhebungCCMS:   true,
	}
	for _, code := range processes.All {
		if !expected[code] {
			t.Errorf("unexpected code in All: %q", code)
		}
		delete(expected, code)
	}
	for code := range expected {
		t.Errorf("code %q missing from All", code)
	}
}
