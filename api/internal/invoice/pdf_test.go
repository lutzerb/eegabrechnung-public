package invoice

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

func TestGeneratePDF(t *testing.T) {
	eegID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	memberID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
	invoiceID := uuid.MustParse("cccccccc-0000-0000-0000-000000000001")

	eeg := &domain.EEG{
		ID:          eegID,
		Name:        "Sonnenschein EEG",
		EnergyPrice: 0.15,
	}
	member := &domain.Member{
		ID:          memberID,
		EegID:       eegID,
		MitgliedsNr: "M001",
		Name1:       "Max",
		Name2:       "Mustermann",
		Email:       "max@example.com",
		IBAN:        "AT12 3456 7890 1234 5678",
	}
	inv := &domain.Invoice{
		ID:          invoiceID,
		MemberID:    memberID,
		EegID:       eegID,
		PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		TotalKwh:    123.456,
		TotalAmount: 18.5184,
	}

	data, err := GeneratePDF(inv, eeg, member, VATOptions{UseVat: false})
	if err != nil {
		t.Fatalf("GeneratePDF returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("GeneratePDF returned empty data")
	}
	// PDF magic bytes check
	if !strings.HasPrefix(string(data[:5]), "%PDF-") {
		t.Errorf("expected PDF magic bytes, got: %q", string(data[:8]))
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{1234.56, "1.234,56 €"},
		{0.50, "0,50 €"},
		{1000000.00, "1.000.000,00 €"},
		{-99.99, "-99,99 €"},
	}
	for _, c := range cases {
		got := formatAmount(c.in)
		if got != c.want {
			t.Errorf("formatAmount(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPeriodLabel(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := periodLabel(ts)
	if got != "Jänner 2026" {
		t.Errorf("periodLabel = %q, want %q", got, "Jänner 2026")
	}
}
