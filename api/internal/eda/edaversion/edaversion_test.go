package edaversion

import (
	"testing"
	"time"
)

func TestResolve(t *testing.T) {
	cutover := time.Date(2026, 10, 5, 0, 0, 0, 0, Vienna)
	history := []Entry[string]{
		{Value: "OLD"},
		{From: cutover, Value: "NEW"},
	}

	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"long before cutover", time.Date(2020, 1, 1, 0, 0, 0, 0, Vienna), "OLD"},
		{"one second before cutover", cutover.Add(-time.Second), "OLD"},
		{"exactly at cutover", cutover, "NEW"},
		{"after cutover", cutover.Add(24 * time.Hour), "NEW"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(history, c.now); got != c.want {
				t.Errorf("Resolve(%v) = %q, want %q", c.now, got, c.want)
			}
		})
	}
}

func TestResolveEmptyHistory(t *testing.T) {
	if got := Resolve([]Entry[string](nil), time.Now()); got != "" {
		t.Errorf("Resolve(nil) = %q, want zero value", got)
	}
}
