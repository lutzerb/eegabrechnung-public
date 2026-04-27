package xml_test

import (
	"strings"
	"testing"
	"time"

	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/eda/processes"
)

func TestBuildAnforderung(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	for _, code := range processes.All {
		t.Run(code, func(t *testing.T) {
			p := edaxml.BuildParams{
				From:           "sender@eeg.at",
				To:             "edanet@edanet.at",
				GemeinschaftID: "EEG-AT-TESTGC",
				Process:        code,
				FromDate:       from,
				ToDate:         to,
			}
			got, err := edaxml.BuildAnforderung(p)
			if err != nil {
				t.Fatalf("BuildAnforderung(%s) error: %v", code, err)
			}
			if !strings.Contains(got, "<?xml") {
				t.Errorf("expected XML header, got: %s", got[:50])
			}
			if !strings.Contains(got, code) {
				t.Errorf("expected process code %s in output", code)
			}
			if !strings.Contains(got, "EEG-AT-TESTGC") {
				t.Errorf("expected GemeinschaftsID in output")
			}
			if !strings.Contains(got, "sender@eeg.at") {
				t.Errorf("expected From address in output")
			}
			if !strings.Contains(got, "edanet@edanet.at") {
				t.Errorf("expected To address in output")
			}
		})
	}
}

func TestBuildAnforderung_MissingFields(t *testing.T) {
	base := edaxml.BuildParams{
		From:           "sender@eeg.at",
		To:             "edanet@edanet.at",
		GemeinschaftID: "EEG-AT-TESTGC",
		Process:        processes.AnforderungECON,
		FromDate:       time.Now(),
		ToDate:         time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name   string
		mutate func(*edaxml.BuildParams)
	}{
		{"missing From", func(p *edaxml.BuildParams) { p.From = "" }},
		{"missing To", func(p *edaxml.BuildParams) { p.To = "" }},
		{"missing GemeinschaftID", func(p *edaxml.BuildParams) { p.GemeinschaftID = "" }},
		{"unknown process", func(p *edaxml.BuildParams) { p.Process = "UNKNOWN_CODE" }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			_, err := edaxml.BuildAnforderung(p)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
