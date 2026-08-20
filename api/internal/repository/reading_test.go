package repository

import (
	"reflect"
	"testing"
)

func TestCollapseDaysToCategorizedRanges(t *testing.T) {
	tests := []struct {
		name       string
		days       []string
		categories []GapCategory
		want       []CategorizedRange
	}{
		{
			name:       "empty input",
			days:       nil,
			categories: nil,
			want:       nil,
		},
		{
			name:       "single day",
			days:       []string{"2026-04-15"},
			categories: []GapCategory{GapNoData},
			want:       []CategorizedRange{{From: "2026-04-15", To: "2026-04-15", Category: GapNoData}},
		},
		{
			name:       "two consecutive days, same category",
			days:       []string{"2026-04-15", "2026-04-16"},
			categories: []GapCategory{GapNoData, GapNoData},
			want:       []CategorizedRange{{From: "2026-04-15", To: "2026-04-16", Category: GapNoData}},
		},
		{
			name:       "gap in the middle",
			days:       []string{"2026-01-01", "2026-01-02", "2026-01-05", "2026-01-06"},
			categories: []GapCategory{GapNoData, GapNoData, GapNoData, GapNoData},
			want: []CategorizedRange{
				{From: "2026-01-01", To: "2026-01-02", Category: GapNoData},
				{From: "2026-01-05", To: "2026-01-06", Category: GapNoData},
			},
		},
		{
			name:       "month boundary",
			days:       []string{"2026-01-31", "2026-02-01"},
			categories: []GapCategory{GapPartial, GapPartial},
			want:       []CategorizedRange{{From: "2026-01-31", To: "2026-02-01", Category: GapPartial}},
		},
		{
			name:       "consecutive days but different category are NOT merged",
			days:       []string{"2026-05-01", "2026-05-02", "2026-05-03"},
			categories: []GapCategory{GapNoData, GapNoData, GapL3Only},
			want: []CategorizedRange{
				{From: "2026-05-01", To: "2026-05-02", Category: GapNoData},
				{From: "2026-05-03", To: "2026-05-03", Category: GapL3Only},
			},
		},
		{
			name:       "category changes back and forth on consecutive days",
			days:       []string{"2026-06-01", "2026-06-02", "2026-06-03"},
			categories: []GapCategory{GapL3Only, GapPartial, GapL3Only},
			want: []CategorizedRange{
				{From: "2026-06-01", To: "2026-06-01", Category: GapL3Only},
				{From: "2026-06-02", To: "2026-06-02", Category: GapPartial},
				{From: "2026-06-03", To: "2026-06-03", Category: GapL3Only},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collapseDaysToCategorizedRanges(tt.days, tt.categories)
			if err != nil {
				t.Fatalf("collapseDaysToCategorizedRanges(%v, %v) returned error: %v", tt.days, tt.categories, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("collapseDaysToCategorizedRanges(%v, %v) = %+v, want %+v", tt.days, tt.categories, got, tt.want)
			}
		})
	}
}
