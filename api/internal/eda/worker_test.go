package eda

import (
	"testing"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/google/uuid"
)

// crMsgRecordWith builds a generation-meter CRMsgRecord with one 15-min slot and
// the given EnergyData blocks in the given document order.
func crMsgRecordWith(data []edaxml.CRMsgEnergyData) *edaxml.CRMsgRecord {
	return &edaxml.CRMsgRecord{
		Zaehlpunkt: "AT0020000000000000000000100344898",
		Energies: []edaxml.CRMsgBlock{
			{
				PeriodStart: time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
				PeriodEnd:   time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
				Intervall:   "QH",
				Data:        data,
			},
		},
	}
}

func genEnergyData(meterCode string, value float64) edaxml.CRMsgEnergyData {
	return edaxml.CRMsgEnergyData{
		MeterCode: meterCode,
		UOM:       "KWH",
		Positions: []edaxml.CRMsgPosition{
			{From: time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC), Value: value, Quality: "L1"},
		},
	}
}

// Generation meters (2.9.0): wh_total is always the plain G.01 (100% plant total),
// regardless of whether G.01T is present. wh_community is derived separately as
// G.01T − P.01T (scaled basis) when G.01T is present — in arbitrary document order.
func TestBuildReadingsFromCRMsg_GenerationTotalAlwaysPlainCommunityFromScaled(t *testing.T) {
	mpID := uuid.New()
	resolve := func(ts time.Time) (uuid.UUID, bool) { return mpID, true }

	orders := map[string][]edaxml.CRMsgEnergyData{
		"G.01 first (daily push order)": {
			genEnergyData("1-1:2.9.0 G.01", 2.0),  // 100% plant total
			genEnergyData("1-1:2.9.0 G.01T", 0.8), // × 40% Teilnahmefaktor
			genEnergyData("1-1:2.9.0 P.01T", 0.3), // Restnetzüberschuss (40% basis)
		},
		"G.01 last (CR_REQ_PT response order)": {
			genEnergyData("1-1:2.9.0 P.01T", 0.3),
			genEnergyData("1-1:2.9.0 G.01T", 0.8),
			genEnergyData("1-1:2.9.0 G.01", 2.0),
		},
	}

	for name, data := range orders {
		readings := buildReadingsFromCRMsg(resolve, crMsgRecordWith(data), time.Time{})
		if len(readings) != 1 {
			t.Fatalf("%s: expected 1 reading, got %d", name, len(readings))
		}
		r := readings[0]
		if r.WhTotal != 2.0 {
			t.Errorf("%s: wh_total = %v, want 2.0 (plain G.01, regardless of G.01T)", name, r.WhTotal)
		}
		if r.WhSelf != 0.3 {
			t.Errorf("%s: wh_self = %v, want 0.3 (P.01T)", name, r.WhSelf)
		}
		if diff := r.WhCommunity - 0.5; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("%s: wh_community = %v, want 0.5 (G.01T − P.01T)", name, r.WhCommunity)
		}
	}
}

// Generation meters without Mehrfachteilnahme only get G.01 — it must still fill
// wh_total, and wh_community must fall back to the plain G.01 basis (no G.01T).
func TestBuildReadingsFromCRMsg_GenerationPlainTotalWithoutScaled(t *testing.T) {
	mpID := uuid.New()
	resolve := func(ts time.Time) (uuid.UUID, bool) { return mpID, true }

	readings := buildReadingsFromCRMsg(resolve, crMsgRecordWith([]edaxml.CRMsgEnergyData{
		genEnergyData("1-1:2.9.0 G.01", 2.0),
		genEnergyData("1-1:2.9.0 P.01T", 0.3),
	}), time.Time{})
	if len(readings) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(readings))
	}
	r := readings[0]
	if r.WhTotal != 2.0 {
		t.Errorf("wh_total = %v, want 2.0", r.WhTotal)
	}
	if diff := r.WhCommunity - 1.7; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("wh_community = %v, want 1.7", r.WhCommunity)
	}
}

// Consumption meters (1.9.0): G.01T is unreliable on some Netzbetreiber — reported
// as a persistent 0 while G.01 has the real total (observed 2026-09-04,
// AT0082300807...19832). Unlike generation, G.01 always wins here regardless of
// G.01T's value or document order.
func TestBuildReadingsFromCRMsg_ConsumptionPlainTotalAlwaysWinsOverScaled(t *testing.T) {
	mpID := uuid.New()
	resolve := func(ts time.Time) (uuid.UUID, bool) { return mpID, true }

	orders := map[string][]edaxml.CRMsgEnergyData{
		"G.01 first, G.01T nonzero": {
			genEnergyData("1-1:1.9.0 G.01", 2.0),
			genEnergyData("1-1:1.9.0 G.01T", 0.8),
		},
		"G.01 last, G.01T nonzero": {
			genEnergyData("1-1:1.9.0 G.01T", 0.8),
			genEnergyData("1-1:1.9.0 G.01", 2.0),
		},
		"G.01T reported as 0 (Netzbetreiber quirk)": {
			genEnergyData("1-1:1.9.0 G.01", 2.0),
			genEnergyData("1-1:1.9.0 G.01T", 0),
		},
	}

	for name, data := range orders {
		readings := buildReadingsFromCRMsg(resolve, crMsgRecordWith(data), time.Time{})
		if len(readings) != 1 {
			t.Fatalf("%s: expected 1 reading, got %d", name, len(readings))
		}
		if r := readings[0]; r.WhTotal != 2.0 {
			t.Errorf("%s: wh_total = %v, want 2.0 (consumption G.01 must win over G.01T)", name, r.WhTotal)
		}
	}
}

func eegWithIMAPCreds(name, host, user, password string) *domain.EEG {
	return &domain.EEG{
		ID:              uuid.New(),
		Name:            name,
		EDAIMAPHost:     host,
		EDAIMAPUser:     user,
		EDAIMAPPassword: password,
	}
}

func TestGroupEEGsByIMAPCredentials_SharedAccountGroupedTogether(t *testing.T) {
	a := eegWithIMAPCreds("Gießenberg", "mail.edanet.at:993", "gc104929", "sekret")
	b := eegWithIMAPCreds("Ziegelstraße", "mail.edanet.at:993", "gc104929", "sekret")
	c := eegWithIMAPCreds("Anderer Account", "mail.edanet.at:993", "rc999999", "andereswort")

	groups := groupEEGsByIMAPCredentials([]*domain.EEG{a, b, c})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if len(groups[0]) != 2 {
		t.Fatalf("expected first group (shared account) to have 2 EEGs, got %d", len(groups[0]))
	}
	if groups[0][0].ID != a.ID || groups[0][1].ID != b.ID {
		t.Errorf("expected first group to contain a then b in encounter order, got %v", groups[0])
	}
	if len(groups[1]) != 1 || groups[1][0].ID != c.ID {
		t.Fatalf("expected second group to contain only c, got %v", groups[1])
	}
}

func TestGroupEEGsByIMAPCredentials_NoSharedCredentials(t *testing.T) {
	a := eegWithIMAPCreds("A", "host1", "user1", "pw1")
	b := eegWithIMAPCreds("B", "host2", "user2", "pw2")

	groups := groupEEGsByIMAPCredentials([]*domain.EEG{a, b})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (no shared credentials), got %d", len(groups))
	}
}

func TestGroupEEGsByIMAPCredentials_Empty(t *testing.T) {
	groups := groupEEGsByIMAPCredentials(nil)
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestBackoffDuration_IncreasesWithRetryCountAndStaysWithinJitterBounds(t *testing.T) {
	cases := []struct {
		retryCount int
		base       time.Duration
	}{
		{retryCount: 1, base: 1 * time.Minute},
		{retryCount: 2, base: 5 * time.Minute},
		{retryCount: 3, base: 5 * time.Minute}, // clamped to the last tier
	}
	for _, c := range cases {
		for i := 0; i < 20; i++ { // sample the jitter a few times
			d := backoffDuration(c.retryCount)
			maxD := c.base + c.base/4
			if d < c.base || d > maxD {
				t.Fatalf("retryCount=%d: backoffDuration=%v, want in [%v, %v]", c.retryCount, d, c.base, maxD)
			}
		}
	}
}
