package calculator

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	meterA = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	meterB = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	meterC = uuid.MustParse("00000000-0000-0000-0000-000000000003") // GENERATION
	ts1    = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2    = time.Date(2026, 1, 1, 0, 15, 0, 0, time.UTC)
)

func TestCalculate_BasicAllocation(t *testing.T) {
	readings := []Reading{
		// ts1: generation=100, consumptionA=60, consumptionB=40
		{MeterPointID: meterC, Energierichtung: "GENERATION", Ts: ts1, WhTotal: 100},
		{MeterPointID: meterA, Energierichtung: "CONSUMPTION", Ts: ts1, WhTotal: 60},
		{MeterPointID: meterB, Energierichtung: "CONSUMPTION", Ts: ts1, WhTotal: 40},
	}

	results := Calculate(readings)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Find results for each meter
	allocs := map[uuid.UUID]float64{}
	for _, r := range results {
		allocs[r.MeterPointID] = r.Allocated
	}

	// meterA: share = 60/100 = 0.6, allocated = min(100*0.6, 60) = min(60,60) = 60
	if abs(allocs[meterA]-60) > 0.001 {
		t.Errorf("meterA: expected 60, got %f", allocs[meterA])
	}
	// meterB: share = 40/100 = 0.4, allocated = min(100*0.4, 40) = min(40,40) = 40
	if abs(allocs[meterB]-40) > 0.001 {
		t.Errorf("meterB: expected 40, got %f", allocs[meterB])
	}
}

func TestCalculate_PartialGeneration(t *testing.T) {
	// Generation < total consumption: allocation is capped by generation
	readings := []Reading{
		// ts1: generation=50, consumptionA=60, consumptionB=40
		{MeterPointID: meterC, Energierichtung: "GENERATION", Ts: ts1, WhTotal: 50},
		{MeterPointID: meterA, Energierichtung: "CONSUMPTION", Ts: ts1, WhTotal: 60},
		{MeterPointID: meterB, Energierichtung: "CONSUMPTION", Ts: ts1, WhTotal: 40},
	}

	results := Calculate(readings)

	allocs := map[uuid.UUID]float64{}
	for _, r := range results {
		allocs[r.MeterPointID] = r.Allocated
	}

	// meterA: share = 60/100 = 0.6, allocated = min(50*0.6, 60) = min(30, 60) = 30
	if abs(allocs[meterA]-30) > 0.001 {
		t.Errorf("meterA: expected 30, got %f", allocs[meterA])
	}
	// meterB: share = 40/100 = 0.4, allocated = min(50*0.4, 40) = min(20, 40) = 20
	if abs(allocs[meterB]-20) > 0.001 {
		t.Errorf("meterB: expected 20, got %f", allocs[meterB])
	}
}

func TestCalculate_ZeroConsumption(t *testing.T) {
	readings := []Reading{
		{MeterPointID: meterC, Energierichtung: "GENERATION", Ts: ts1, WhTotal: 100},
		{MeterPointID: meterA, Energierichtung: "CONSUMPTION", Ts: ts1, WhTotal: 0},
	}

	results := Calculate(readings)

	for _, r := range results {
		if r.Allocated != 0 {
			t.Errorf("expected 0 allocation when consumption=0, got %f", r.Allocated)
		}
	}
}

func TestCalculate_MultipleTimestamps(t *testing.T) {
	readings := []Reading{
		{MeterPointID: meterC, Energierichtung: "GENERATION", Ts: ts1, WhTotal: 100},
		{MeterPointID: meterA, Energierichtung: "CONSUMPTION", Ts: ts1, WhTotal: 100},
		{MeterPointID: meterC, Energierichtung: "GENERATION", Ts: ts2, WhTotal: 200},
		{MeterPointID: meterA, Energierichtung: "CONSUMPTION", Ts: ts2, WhTotal: 100},
	}

	results := Calculate(readings)

	// Should have 2 results (one per timestamp for meterA)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	ts1Alloc := 0.0
	ts2Alloc := 0.0
	for _, r := range results {
		if r.Ts.Equal(ts1) {
			ts1Alloc = r.Allocated
		}
		if r.Ts.Equal(ts2) {
			ts2Alloc = r.Allocated
		}
	}

	// ts1: allocated = min(100*1.0, 100) = 100
	if abs(ts1Alloc-100) > 0.001 {
		t.Errorf("ts1: expected 100, got %f", ts1Alloc)
	}
	// ts2: share=1.0, allocated = min(200*1.0, 100) = 100 (capped by consumption)
	if abs(ts2Alloc-100) > 0.001 {
		t.Errorf("ts2: expected 100, got %f", ts2Alloc)
	}
}

func TestAggregateMemberBilling(t *testing.T) {
	memberID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	mp1 := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	mp2 := uuid.MustParse("00000000-0000-0000-0000-000000000012")

	readings := []Reading{
		{MeterPointID: mp1, Energierichtung: "CONSUMPTION", Ts: ts1, WhCommunity: 100},
		{MeterPointID: mp1, Energierichtung: "CONSUMPTION", Ts: ts2, WhCommunity: 50},
		{MeterPointID: mp2, Energierichtung: "CONSUMPTION", Ts: ts1, WhCommunity: 200},
	}

	memberMeterPoints := map[uuid.UUID][]uuid.UUID{
		memberID: {mp1, mp2},
	}

	bills := AggregateMemberBilling(readings, memberMeterPoints, 0.12)

	if len(bills) != 1 {
		t.Fatalf("expected 1 bill, got %d", len(bills))
	}

	bill := bills[0]
	// totalWh = 100 + 50 + 200 = 350
	if abs(bill.TotalKwh-350) > 0.001 {
		t.Errorf("expected TotalKwh=350, got %f", bill.TotalKwh)
	}
	// totalAmount = 350 * 0.12 = 42
	if abs(bill.TotalAmount-42) > 0.01 {
		t.Errorf("expected TotalAmount=42, got %f", bill.TotalAmount)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
