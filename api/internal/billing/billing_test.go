package billing

import (
	"testing"
	"time"
)

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

// autoBillingPeriod must produce Vienna-local calendar boundaries with an
// inclusive end (last day 23:59:59), matching the manual billing handler.
// UTC boundaries or an exclusive midnight end would double-bill the first
// 15-min slot of the following month (sum queries use ts <= end).
func TestAutoBillingPeriod(t *testing.T) {
	vienna, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatalf("load Europe/Vienna: %v", err)
	}

	cases := []struct {
		name      string
		period    string
		today     time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "monthly",
			period:    "monthly",
			today:     time.Date(2026, 7, 15, 6, 0, 0, 0, vienna),
			wantStart: time.Date(2026, 6, 1, 0, 0, 0, 0, vienna),
			wantEnd:   time.Date(2026, 6, 30, 23, 59, 59, 0, vienna),
		},
		{
			name:      "monthly across DST spring-forward (March)",
			period:    "monthly",
			today:     time.Date(2026, 4, 1, 6, 0, 0, 0, vienna),
			wantStart: time.Date(2026, 3, 1, 0, 0, 0, 0, vienna),
			wantEnd:   time.Date(2026, 3, 31, 23, 59, 59, 0, vienna),
		},
		{
			name:      "quarterly",
			period:    "quarterly",
			today:     time.Date(2026, 4, 1, 6, 0, 0, 0, vienna),
			wantStart: time.Date(2026, 1, 1, 0, 0, 0, 0, vienna),
			wantEnd:   time.Date(2026, 3, 31, 23, 59, 59, 0, vienna),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := autoBillingPeriod(tc.period, tc.today)
			if !start.Equal(tc.wantStart) {
				t.Errorf("start: expected %v, got %v", tc.wantStart, start)
			}
			if !end.Equal(tc.wantEnd) {
				t.Errorf("end: expected %v, got %v", tc.wantEnd, end)
			}
			// The end must be strictly before the next period's start so that a
			// reading exactly on the next month's first slot is never included.
			nextStart := tc.wantEnd.Add(time.Second)
			if !end.Before(nextStart) {
				t.Errorf("end %v must be before next period start %v", end, nextStart)
			}
		})
	}
}
