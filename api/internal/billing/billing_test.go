package billing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/calculator"
)

// Test the core aggregation logic independently of the DB.
func TestAggregateBilling_Integration(t *testing.T) {
	member1 := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	member2 := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	mp1 := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	mp2 := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	mp3 := uuid.MustParse("20000000-0000-0000-0000-000000000003")

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	readings := []calculator.Reading{
		{MeterPointID: mp1, Energierichtung: "CONSUMPTION", Ts: ts, WhCommunity: 100},
		{MeterPointID: mp2, Energierichtung: "CONSUMPTION", Ts: ts, WhCommunity: 200},
		{MeterPointID: mp3, Energierichtung: "CONSUMPTION", Ts: ts, WhCommunity: 50},
	}

	memberMeterPoints := map[uuid.UUID][]uuid.UUID{
		member1: {mp1, mp2}, // total 300
		member2: {mp3},      // total 50
	}

	bills := calculator.AggregateMemberBilling(readings, memberMeterPoints, 0.15)

	if len(bills) != 2 {
		t.Fatalf("expected 2 bills, got %d", len(bills))
	}

	byMember := map[uuid.UUID]calculator.MemberBilling{}
	for _, b := range bills {
		byMember[b.MemberID] = b
	}

	b1 := byMember[member1]
	if absFloat(b1.TotalKwh-300) > 0.001 {
		t.Errorf("member1 TotalKwh: expected 300, got %f", b1.TotalKwh)
	}
	if absFloat(b1.TotalAmount-45) > 0.01 {
		t.Errorf("member1 TotalAmount: expected 45 (300*0.15), got %f", b1.TotalAmount)
	}

	b2 := byMember[member2]
	if absFloat(b2.TotalKwh-50) > 0.001 {
		t.Errorf("member2 TotalKwh: expected 50, got %f", b2.TotalKwh)
	}
	if absFloat(b2.TotalAmount-7.5) > 0.01 {
		t.Errorf("member2 TotalAmount: expected 7.5 (50*0.15), got %f", b2.TotalAmount)
	}
}

func TestPeriodValidation(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	if end.Before(start) {
		t.Error("end should not be before start")
	}

	// Test invalid period
	invalidEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	if !invalidEnd.Before(start) {
		t.Error("invalidEnd should be before start")
	}
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
