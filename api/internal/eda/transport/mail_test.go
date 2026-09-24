package transport

import (
	"testing"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/edaversion"
)

func TestResolveProzessID(t *testing.T) {
	before := edaversion.Cutover2026_10_05.Add(-time.Hour)
	after := edaversion.Cutover2026_10_05

	cases := []struct {
		process string
		now     time.Time
		want    string
	}{
		{"EC_PRTFACT_CHG", before, "EC_PRTFACT_CHANGE_01.00"},
		{"EC_PRTFACT_CHG", after, "EC_PRTFACT_CHANGE_01.10"},
		{"EC_REQ_ONL", before, "EC_REQ_ONL_02.30"},
		{"EC_REQ_ONL", after, "EC_REQ_ONL_02.40"},
		{"CR_REQ_PT", before, "CR_REQ_PT_04.10"},
		{"CR_REQ_PT", after, "CR_REQ_PT_04.30"},
		{"EC_PODLIST", before, "EC_PODLIST_01.00"},
		{"EC_PODLIST", after, "EC_PODLIST_02.10"},
		{"CM_REV_SP", before, "CM_REV_SP_01.00"},
		{"CM_REV_SP", after, "CM_REV_SP_01.30"},
		{"UNKNOWN_PROCESS", after, ""},
	}

	for _, c := range cases {
		if got := resolveProzessID(c.process, c.now); got != c.want {
			t.Errorf("resolveProzessID(%q, %v) = %q, want %q", c.process, c.now, got, c.want)
		}
	}
}
