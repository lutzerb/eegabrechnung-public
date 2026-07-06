package xml_test

import (
	"strings"
	"testing"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/processes"
	"github.com/lutzerb/eegabrechnung/internal/eda/types"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

func TestRoundTrip(t *testing.T) {
	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC)

	p := edaxml.BuildParams{
		From:           "absender@eeg.at",
		To:             "empfaenger@edanet.at",
		GemeinschaftID: "EEG-GC-12345",
		Process:        processes.AnforderungECON,
		FromDate:       from,
		ToDate:         to,
	}

	xmlStr, err := edaxml.BuildAnforderung(p)
	if err != nil {
		t.Fatalf("BuildAnforderung: %v", err)
	}

	msg, err := edaxml.ParseAnforderung(xmlStr)
	if err != nil {
		t.Fatalf("ParseAnforderung: %v", err)
	}

	if msg.Process != processes.AnforderungECON {
		t.Errorf("Process: got %q, want %q", msg.Process, processes.AnforderungECON)
	}
	if msg.From != "absender@eeg.at" {
		t.Errorf("From: got %q, want %q", msg.From, "absender@eeg.at")
	}
	if msg.To != "empfaenger@edanet.at" {
		t.Errorf("To: got %q, want %q", msg.To, "empfaenger@edanet.at")
	}
	if msg.GemeinschaftID != "EEG-GC-12345" {
		t.Errorf("GemeinschaftID: got %q, want %q", msg.GemeinschaftID, "EEG-GC-12345")
	}
	if msg.Direction != types.DirectionInbound {
		t.Errorf("Direction: got %q, want %q", msg.Direction, types.DirectionInbound)
	}
	if msg.XMLPayload == "" {
		t.Error("XMLPayload should not be empty")
	}
}

func TestParseAnforderung_AllProcessCodes(t *testing.T) {
	for _, code := range processes.All {
		t.Run(code, func(t *testing.T) {
			p := edaxml.BuildParams{
				From:           "from@test.at",
				To:             "to@test.at",
				GemeinschaftID: "TEST-GC",
				Process:        code,
				FromDate:       time.Now(),
				ToDate:         time.Now().Add(24 * time.Hour),
			}
			xmlStr, err := edaxml.BuildAnforderung(p)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			msg, err := edaxml.ParseAnforderung(xmlStr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if msg.Process != code {
				t.Errorf("got process %q, want %q", msg.Process, code)
			}
		})
	}
}

func TestParseAnforderung_InvalidXML(t *testing.T) {
	_, err := edaxml.ParseAnforderung("not xml at all <<<")
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

func TestParseAnforderung_MissingProzess(t *testing.T) {
	xmlStr := `<?xml version="1.0" encoding="UTF-8"?>
<req:Anforderung xmlns:req="http://www.ebutilities.at/saldomodell/EDA/1.0">
  <req:Marktteilnehmer>
    <req:Absender>from@test.at</req:Absender>
    <req:Empfaenger>to@test.at</req:Empfaenger>
  </req:Marktteilnehmer>
  <req:Gemeinschaft>
    <req:GemeinschaftsID>TEST-GC</req:GemeinschaftsID>
  </req:Gemeinschaft>
</req:Anforderung>`

	_, err := edaxml.ParseAnforderung(xmlStr)
	if err == nil {
		t.Error("expected error for missing Prozess element")
	}
	if !strings.Contains(err.Error(), "Prozess") {
		t.Errorf("error should mention Prozess, got: %v", err)
	}
}
