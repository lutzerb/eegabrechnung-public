package invoice

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

func mustBePDF(t *testing.T, data []byte, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("generation returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("generation returned empty data")
	}
	if !strings.HasPrefix(string(data[:5]), "%PDF-") {
		t.Errorf("expected PDF magic bytes, got: %q", string(data[:8]))
	}
}

func TestGeneratePDFThemed(t *testing.T) {
	eegID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002")
	memberID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
	invoiceID := uuid.MustParse("cccccccc-0000-0000-0000-000000000002")

	eeg := &domain.EEG{ID: eegID, Name: "Sonnenschein EEG", EnergyPrice: 0.15}
	member := &domain.Member{ID: memberID, EegID: eegID, MitgliedsNr: "M001", Name1: "Max", Name2: "Mustermann"}
	inv := &domain.Invoice{
		ID: invoiceID, MemberID: memberID, EegID: eegID,
		PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		TotalKwh:    123.456, TotalAmount: 18.5184,
	}
	energyRows := []EnergyPeriodRow{
		{Zaehlpunkt: "AT0010000000000000000000000000001", ZeitraumVon: inv.PeriodStart, ZeitraumBis: inv.PeriodEnd, GesamtverbrauchKwh: 150, NetzbezugKwh: 26.5, CommunityVerbrauchKwh: 123.456},
		{Zaehlpunkt: "AT0010000000000000000000000000002", ZeitraumVon: inv.PeriodStart, ZeitraumBis: inv.PeriodEnd, GesamtverbrauchKwh: 80, NetzbezugKwh: 10, CommunityVerbrauchKwh: 70},
	}
	generationRows := []GenerationPeriodRow{
		{Zaehlpunkt: "AT0010000000000000000000000000003", ZeitraumVon: inv.PeriodStart, ZeitraumBis: inv.PeriodEnd, GesamteinspeisungKwh: 60, AbnahmeKwh: 45, ResteinspeisungKwh: 15},
	}

	for _, theme := range []InvoiceTheme{
		DefaultOikosTheme(),
		{AccentR: 10, AccentG: 20, AccentB: 30, LogoLeft: false, FontFamily: "roboto", BaseFontSize: 8},
		{AccentR: 200, AccentG: 200, AccentB: 200, LogoLeft: true, FontFamily: "opensans", BaseFontSize: 12},
		{AccentR: 50, AccentG: 50, AccentB: 50, LogoLeft: true, FontFamily: "ptserif", BaseFontSize: 10},
	} {
		data, err := GeneratePDFThemed(inv, eeg, member, VATOptions{UseVat: false}, nil, energyRows, generationRows, theme)
		mustBePDF(t, data, err)
	}
}

func TestGenerateCreditNotePDFThemed(t *testing.T) {
	eegID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000003")
	memberID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000003")
	invoiceID := uuid.MustParse("cccccccc-0000-0000-0000-000000000003")

	eeg := &domain.EEG{ID: eegID, Name: "Sonnenschein EEG", ProducerPrice: 8, GenerateCreditNotes: true}
	member := &domain.Member{ID: memberID, EegID: eegID, MitgliedsNr: "P001", Name1: "Erika", Name2: "Musterfrau"}
	inv := &domain.Invoice{
		ID: invoiceID, MemberID: memberID, EegID: eegID,
		PeriodStart:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:     time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		GenerationKwh: 500, TotalAmount: -40, DocumentType: "credit_note",
	}

	data, err := GenerateCreditNotePDFThemed(inv, eeg, member, 8, 500, nil, nil, nil, DefaultOikosTheme())
	mustBePDF(t, data, err)
}

// TestCreditNotePaymentNoticeMode verifies the fix for GenerateCreditNotePDF's
// "Auszahlung" section, which used to ignore InvoicePaymentNoticeMode entirely
// (always rendered). "none" must now omit the section (shorter PDF); "custom"
// must render without error using the free-text template.
func TestCreditNotePaymentNoticeMode(t *testing.T) {
	eegID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000005")
	memberID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000005")
	invoiceID := uuid.MustParse("cccccccc-0000-0000-0000-000000000005")
	member := &domain.Member{ID: memberID, EegID: eegID, MitgliedsNr: "P001", Name1: "Erika", Name2: "Musterfrau", IBAN: "AT00 0000 0000 0000 0000"}
	inv := &domain.Invoice{
		ID: invoiceID, MemberID: memberID, EegID: eegID,
		PeriodStart:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:     time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		GenerationKwh: 500, TotalAmount: -40, DocumentType: "credit_note",
	}

	base := &domain.EEG{ID: eegID, Name: "Sonnenschein EEG", ProducerPrice: 8, GenerateCreditNotes: true, InvoicePaymentNoticeMode: "sepa_lastschrift"}
	withNotice, err := GenerateCreditNotePDF(inv, base, member, 8, 500, nil, nil, nil)
	mustBePDF(t, withNotice, err)

	none := &domain.EEG{ID: eegID, Name: "Sonnenschein EEG", ProducerPrice: 8, GenerateCreditNotes: true, InvoicePaymentNoticeMode: "none"}
	withoutNotice, err := GenerateCreditNotePDF(inv, none, member, 8, 500, nil, nil, nil)
	mustBePDF(t, withoutNotice, err)
	if len(withoutNotice) >= len(withNotice) {
		t.Errorf("expected mode=none PDF to be shorter (Auszahlung section omitted): none=%d bytes, sepa_lastschrift=%d bytes", len(withoutNotice), len(withNotice))
	}

	custom := &domain.EEG{
		ID: eegID, Name: "Sonnenschein EEG", ProducerPrice: 8, GenerateCreditNotes: true,
		InvoicePaymentNoticeMode: "custom",
		InvoicePaymentNoticeText: "Auszahlung von {betrag} an {iban} von {eeg_iban}.",
	}
	customData, err := GenerateCreditNotePDF(inv, custom, member, 8, 500, nil, nil, nil)
	mustBePDF(t, customData, err)
}

func TestRenderPaymentNoticeTemplate(t *testing.T) {
	datum := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	got := renderPaymentNoticeTemplate("Betrag {betrag}, IBAN {iban}, EEG-IBAN {eeg_iban}, BIC {eeg_bic}, Datum {datum}",
		18.17, "AT11 1111 1111 1111 1111", "AT22 2222 2222 2222 2222", "RLNWATWW", datum)
	want := "Betrag 18,17 €, IBAN AT11 1111 1111 1111 1111, EEG-IBAN AT22 2222 2222 2222 2222, BIC RLNWATWW, Datum 17.07.2026"
	if got != want {
		t.Errorf("renderPaymentNoticeTemplate = %q, want %q", got, want)
	}

	// Zero time → {datum} substituted with empty string, not left literal.
	got2 := renderPaymentNoticeTemplate("Fällig am {datum}.", 1, "", "", "", time.Time{})
	if got2 != "Fällig am ." {
		t.Errorf("renderPaymentNoticeTemplate with zero datum = %q, want %q", got2, "Fällig am .")
	}
}

func TestGenerateStornorechnungThemed(t *testing.T) {
	eegID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000004")
	memberID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000004")
	invoiceID := uuid.MustParse("cccccccc-0000-0000-0000-000000000004")

	eeg := &domain.EEG{ID: eegID, Name: "Sonnenschein EEG"}
	member := &domain.Member{ID: memberID, EegID: eegID, MitgliedsNr: "M001", Name1: "Max", Name2: "Mustermann"}
	inv := &domain.Invoice{
		ID: invoiceID, MemberID: memberID, EegID: eegID,
		PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		NetAmount:   15.14, VatAmount: 3.03, VatPctApplied: 20, TotalAmount: 18.17,
		DocumentType: "invoice",
	}

	data, err := GenerateStornorechnungThemed(inv, eeg, member, DefaultOikosTheme())
	mustBePDF(t, data, err)
}

func TestThemeFromEEG(t *testing.T) {
	eeg := &domain.EEG{InvoiceAccentColor: "#1a2b3c", InvoiceLogoLeft: false, InvoiceFontFamily: "roboto", InvoiceFontSize: 11}
	theme := ThemeFromEEG(eeg)
	if theme.AccentR != 0x1a || theme.AccentG != 0x2b || theme.AccentB != 0x3c {
		t.Errorf("ThemeFromEEG accent = %d,%d,%d, want 26,43,60", theme.AccentR, theme.AccentG, theme.AccentB)
	}
	if theme.LogoLeft {
		t.Error("expected LogoLeft=false")
	}
	if theme.FontFamily != "roboto" || theme.BaseFontSize != 11 {
		t.Errorf("ThemeFromEEG font = %s/%v, want roboto/11", theme.FontFamily, theme.BaseFontSize)
	}

	// Invalid/empty hex color falls back to the default accent.
	fallback := ThemeFromEEG(&domain.EEG{InvoiceAccentColor: "not-a-color"})
	def := DefaultOikosTheme()
	if fallback.AccentR != def.AccentR || fallback.AccentG != def.AccentG || fallback.AccentB != def.AccentB {
		t.Error("expected fallback to DefaultOikosTheme's accent color on invalid hex")
	}
}
