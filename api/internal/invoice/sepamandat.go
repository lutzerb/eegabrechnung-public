package invoice

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// GenerateSepaMandatPDF creates a SEPA mandate PDF for the given member.
// The mandate includes all data required for §§ 58–59 ZaDiG 2018:
// creditor ID, mandate reference, debtor name/address/IBAN, signature date,
// and the contract text accepted during onboarding.
func GenerateSepaMandatPDF(eeg *domain.EEG, member *domain.Member) ([]byte, error) {
	pdf := newPDF()
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	// ── Header ────────────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 16)
	pdf.SetTextColor(40, 80, 40)
	pdf.CellFormat(0, 10, "SEPA-Lastschriftmandat", "", 1, "C", false, 0, "")

	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 6, "(SEPA Direct Debit Mandate)", "", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(6)

	// ── Creditor section ──────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetFillColor(240, 245, 240)
	pdf.CellFormat(0, 7, "Gläubiger (Creditor)", "0", 1, "L", true, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.Ln(1)

	row := func(label, value string) {
		pdf.SetFont("DejaVu", "B", 9)
		pdf.CellFormat(60, 6, label, "0", 0, "L", false, 0, "")
		pdf.SetFont("DejaVu", "", 9)
		pdf.CellFormat(0, 6, value, "0", 1, "L", false, 0, "")
	}

	row("Gläubiger-Bezeichnung:", eeg.Name)
	creditorID := eeg.SepaCreditorID
	if creditorID == "" {
		creditorID = "(noch nicht konfiguriert)"
	}
	row("Creditor-ID:", creditorID)
	if eeg.IBAN != "" {
		row("Gläubiger IBAN:", eeg.IBAN)
	}
	eegAddr := buildAddress(eeg.Strasse, eeg.Plz, eeg.Ort)
	if eegAddr != "" {
		row("Adresse:", eegAddr)
	}
	pdf.Ln(5)

	// ── Mandate reference section ─────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetFillColor(240, 245, 240)
	pdf.CellFormat(0, 7, "Mandatsreferenz (Mandate Reference)", "0", 1, "L", true, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.Ln(1)

	mandateRef := member.MitgliedsNr
	if mandateRef == "" {
		mandateRef = member.ID.String()[:8]
	}
	row("Mandatsreferenz:", mandateRef)
	row("Zahlungsart:", "Wiederkehrende Zahlung (SEPA CORE)")
	pdf.Ln(5)

	// ── Debtor section ────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetFillColor(240, 245, 240)
	pdf.CellFormat(0, 7, "Zahlungspflichtiger (Debtor)", "0", 1, "L", true, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.Ln(1)

	name := strings.TrimSpace(member.Name1 + " " + member.Name2)
	row("Name:", name)
	memberAddr := buildAddress(member.Strasse, member.Plz, member.Ort)
	if memberAddr != "" {
		row("Adresse:", memberAddr)
	}
	if member.IBAN != "" {
		row("IBAN:", member.IBAN)
	}
	pdf.Ln(5)

	// ── Signature / acceptance section ───────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetFillColor(240, 245, 240)
	pdf.CellFormat(0, 7, "Unterzeichnung (Acceptance)", "0", 1, "L", true, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.Ln(1)

	signedAt := member.SepaMandateSignedAt
	if signedAt == nil {
		t := member.CreatedAt
		signedAt = &t
	}
	row("Datum der Unterzeichnung:", signedAt.Format("02.01.2006 15:04:05 MST"))
	ip := member.SepaMandateSignedIP
	if ip == "" {
		ip = "(nicht aufgezeichnet)"
	}
	row("IP-Adresse:", ip)
	pdf.Ln(5)

	// ── Contract text ─────────────────────────────────────────────────────────
	contractText := member.SepaMandateText
	if contractText == "" {
		// Fallback for members created before this feature: use current EEG template
		contractText = eeg.OnboardingContractText
		if contractText != "" {
			// Fill placeholders with current member data
			contractText = strings.ReplaceAll(contractText, "{iban}", member.IBAN)
			if signedAt != nil {
				contractText = strings.ReplaceAll(contractText, "{datum}", signedAt.Format("02.01.2006"))
			}
		}
	}

	if contractText != "" {
		pdf.SetFont("DejaVu", "B", 10)
		pdf.SetFillColor(240, 245, 240)
		pdf.CellFormat(0, 7, "Beitrittserklärung / Mandatstext", "0", 1, "L", true, 0, "")
		pdf.SetFont("DejaVu", "", 9)
		pdf.Ln(1)
		pdf.MultiCell(0, 5, contractText, "", "L", false)
		pdf.Ln(3)
	}

	// ── Footer ─────────────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.Ln(4)
	pdf.MultiCell(0, 4,
		fmt.Sprintf("Dokument erstellt am %s — %s", time.Now().Format("02.01.2006"), eeg.Name),
		"", "L", false,
	)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("sepa mandat PDF: %w", err)
	}
	return buf.Bytes(), nil
}

func buildAddress(strasse, plz, ort string) string {
	parts := []string{}
	if strasse != "" {
		parts = append(parts, strasse)
	}
	plzOrt := strings.TrimSpace(plz + " " + ort)
	if plzOrt != "" {
		parts = append(parts, plzOrt)
	}
	return strings.Join(parts, ", ")
}
