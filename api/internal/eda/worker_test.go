package eda

import (
	"testing"
	"time"

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

// On Mehrfachteilnahme meters the NB sends both G.01 (100% of the plant) and
// G.01T (scaled by the participation factor) — in arbitrary document order.
// G.01T must win regardless of position; wh_community derives from the scaled total.
func TestBuildReadingsFromCRMsg_ScaledTotalWinsRegardlessOfOrder(t *testing.T) {
	mpID := uuid.New()
	resolve := func(ts time.Time) (uuid.UUID, bool) { return mpID, true }

	orders := map[string][]edaxml.CRMsgEnergyData{
		"G.01 first (daily push order)": {
			genEnergyData("1-1:2.9.0 G.01", 2.0),   // 100% plant total
			genEnergyData("1-1:2.9.0 G.01T", 0.8),  // × 40% Teilnahmefaktor
			genEnergyData("1-1:2.9.0 P.01T", 0.3),  // Restnetzüberschuss (40% basis)
		},
		"G.01 last (CR_REQ_PT response order)": {
			genEnergyData("1-1:2.9.0 P.01T", 0.3),
			genEnergyData("1-1:2.9.0 G.01T", 0.8),
			genEnergyData("1-1:2.9.0 G.01", 2.0),
		},
	}

	for name, data := range orders {
		readings := buildReadingsFromCRMsg(resolve, crMsgRecordWith(data))
		if len(readings) != 1 {
			t.Fatalf("%s: expected 1 reading, got %d", name, len(readings))
		}
		r := readings[0]
		if r.WhTotal != 0.8 {
			t.Errorf("%s: wh_total = %v, want 0.8 (G.01T must win over G.01)", name, r.WhTotal)
		}
		if r.WhSelf != 0.3 {
			t.Errorf("%s: wh_self = %v, want 0.3 (P.01T)", name, r.WhSelf)
		}
		if diff := r.WhCommunity - 0.5; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("%s: wh_community = %v, want 0.5 (G.01T − P.01T)", name, r.WhCommunity)
		}
	}
}

// Meters without Mehrfachteilnahme only get G.01 — it must still fill wh_total.
func TestBuildReadingsFromCRMsg_PlainTotalWithoutScaled(t *testing.T) {
	mpID := uuid.New()
	resolve := func(ts time.Time) (uuid.UUID, bool) { return mpID, true }

	readings := buildReadingsFromCRMsg(resolve, crMsgRecordWith([]edaxml.CRMsgEnergyData{
		genEnergyData("1-1:2.9.0 G.01", 2.0),
		genEnergyData("1-1:2.9.0 P.01T", 0.3),
	}))
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
