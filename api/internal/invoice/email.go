package invoice

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/mailutil"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host     string // e.g. "eegabrechnung-postfix:25" or "localhost:25"
	From     string // e.g. "noreply@eeg.at"
	Username string // empty = no auth (Postfix relay)
	Password string
}

// InvoiceSubject returns the email subject line for an invoice or credit note.
// If eeg.InvoiceEmailSubject is set, it is used as a {{placeholder}} template
// instead of the default text (see applyInvoicePlaceholders).
func InvoiceSubject(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member) string {
	docLabel := "Rechnung"
	if inv.DocumentType == "credit_note" {
		docLabel = "Gutschrift"
	}
	if eeg.InvoiceEmailSubject != "" {
		periodRange := inv.PeriodStart.Format("02.01.2006") + " – " + inv.PeriodEnd.Format("02.01.2006")
		return applyInvoicePlaceholders(eeg.InvoiceEmailSubject, inv, eeg, member, periodRange, inv.DocumentType == "credit_note", "")
	}
	return fmt.Sprintf("Ihre %s – %s – %s", docLabel, eeg.DisplayNameOrName(),
		inv.PeriodStart.Format("02.01.2006")+" – "+inv.PeriodEnd.Format("02.01.2006"))
}

// BuildInvoiceMessage builds the MIME email bytes for an invoice. webBaseURL
// (e.g. "https://eegabrechnung.example.com") is used to point the member at
// their self-service portal; pass "" to omit that hint.
func BuildInvoiceMessage(from, toEmail string, inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, pdfData []byte, webBaseURL string) ([]byte, error) {
	isCredit := inv.DocumentType == "credit_note"
	docLabel := "Rechnung"
	if isCredit {
		docLabel = "Gutschrift"
	}
	subject := InvoiceSubject(inv, eeg, member)
	periodRange := inv.PeriodStart.Format("02.01.2006") + " – " + inv.PeriodEnd.Format("02.01.2006")
	attachmentName := fmt.Sprintf("%s_%s.pdf", docLabel, shortID(inv.ID.String()))
	body := buildPlainBody(inv, eeg, member, periodRange, isCredit, webBaseURL)
	return buildMIMEMessage(from, toEmail, subject, body, attachmentName, pdfData)
}

// SendInvoice sends the invoice PDF to the member's email address and logs the attempt.
func SendInvoice(ctx context.Context, logger EmailLogger, cfg SMTPConfig, member *domain.Member, eeg *domain.EEG, inv *domain.Invoice, pdfData []byte, webBaseURL string) error {
	msgBytes, err := BuildInvoiceMessage(mailutil.FormatAddress(eeg.DisplayNameOrName(), cfg.From), member.Email, inv, eeg, member, pdfData, webBaseURL)
	if err != nil {
		return fmt.Errorf("build mime message: %w", err)
	}
	memID := member.ID
	invID := inv.ID
	return SendLogged(ctx, logger, cfg, eeg.ID, "invoice", member.Email, InvoiceSubject(inv, eeg, member), &memID, &invID, msgBytes)
}

// applyInvoicePlaceholders replaces {{placeholder}} tokens in a custom invoice
// email subject/body template (eeg.InvoiceEmailSubject / eeg.InvoiceEmailBody)
// with invoice-specific values. Supported: {{name}}, {{vorname}}, {{nachname}},
// {{mitglieds_nr}}, {{eeg_name}}, {{dokumenttyp}}, {{rechnungsnummer}},
// {{zeitraum}}, {{zeitraum_von}}, {{zeitraum_bis}}, {{betrag}}, {{bezug_kwh}},
// {{einspeisung_kwh}}, {{zahlungshinweis}}, {{portal_link}}.
// {{bezug_zeile}}/{{einspeisung_zeile}} are the whole-line counterparts (label +
// value + unit + trailing newline, already baked in) that resolve to "" — the
// entire line, not just the number — when the respective kWh amount is 0;
// placed directly adjacent to the next line (no template-side line break
// needed), they reproduce the old hardcoded body's line-hiding behavior exactly.
// {{gesamtbetrag_zeile}} is likewise a whole-line placeholder (see gesamtbetragLine).
func applyInvoicePlaceholders(s string, inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, periodRange string, isCredit bool, webBaseURL string) string {
	docLabel := "Rechnung"
	if isCredit {
		docLabel = "Gutschrift"
	}
	var name, vorname, nachname, mitgliedsNr string
	if member != nil {
		vorname = member.Name1
		nachname = member.Name2
		name = strings.TrimSpace(member.Name1 + " " + member.Name2)
		mitgliedsNr = member.MitgliedsNr
	}
	rechnungsnummer := ""
	if inv.InvoiceNumber != nil {
		prefix := eeg.InvoiceNumberPrefix
		digits := eeg.InvoiceNumberDigits
		if isCredit {
			prefix = eeg.CreditNoteNumberPrefix
			digits = eeg.CreditNoteNumberDigits
		}
		if digits == 0 {
			digits = 5
		}
		rechnungsnummer = fmt.Sprintf("%s%0*d", prefix, digits, *inv.InvoiceNumber)
	}
	bezugKwh := ""
	bezugZeile := ""
	if inv.ConsumptionKwh > 0 {
		bezugKwh = formatKwh(inv.ConsumptionKwh)
		bezugZeile = fmt.Sprintf("Bezug:                %s kWh\n", bezugKwh)
	}
	einspeisungKwh := ""
	einspeisungZeile := ""
	if inv.GenerationKwh > 0 {
		einspeisungKwh = formatKwh(inv.GenerationKwh)
		einspeisungZeile = fmt.Sprintf("Einspeisung:          %s kWh\n", einspeisungKwh)
	}
	portalLink := ""
	if webBaseURL != "" {
		portalLink = webBaseURL + "/portal"
	}
	return strings.NewReplacer(
		"{{name}}", name,
		"{{vorname}}", vorname,
		"{{nachname}}", nachname,
		"{{mitglieds_nr}}", mitgliedsNr,
		"{{eeg_name}}", eeg.DisplayNameOrName(),
		"{{dokumenttyp}}", docLabel,
		"{{rechnungsnummer}}", rechnungsnummer,
		"{{zeitraum}}", periodRange,
		"{{zeitraum_von}}", inv.PeriodStart.Format("02.01.2006"),
		"{{zeitraum_bis}}", inv.PeriodEnd.Format("02.01.2006"),
		"{{betrag}}", formatAmount(inv.TotalAmount),
		"{{bezug_kwh}}", bezugKwh,
		"{{einspeisung_kwh}}", einspeisungKwh,
		"{{bezug_zeile}}", bezugZeile,
		"{{einspeisung_zeile}}", einspeisungZeile,
		"{{gesamtbetrag_zeile}}", gesamtbetragLine(inv, member, isCredit),
		"{{zahlungshinweis}}", paymentNoticeText(inv, eeg, isCredit),
		"{{portal_link}}", portalLink,
	).Replace(s)
}

// paymentNoticeText returns the payment-notice sentence used by the default
// invoice email body (SEPA-Lastschrift / Überweisung / Gutschrift-Rücküberweisung),
// or "" when no notice applies. Also exposed as the {{zahlungshinweis}} placeholder.
func paymentNoticeText(inv *domain.Invoice, eeg *domain.EEG, isCredit bool) string {
	if isCredit {
		amount := inv.TotalAmount
		if amount < 0 {
			amount = -amount
		}
		return fmt.Sprintf("Der Betrag von %s wird auf Ihr hinterlegtes Konto überwiesen.", formatAmount(amount))
	}
	if inv.TotalAmount < 0 {
		return fmt.Sprintf("Der Betrag von %s wird auf Ihr hinterlegtes Konto überwiesen.", formatAmount(-inv.TotalAmount))
	}
	if inv.TotalAmount > 0 && eeg.InvoicePaymentNoticeMode != "none" {
		if eeg.InvoicePaymentNoticeMode == "ueberweisung" {
			return fmt.Sprintf("Bitte überweisen Sie den Betrag von %s auf folgendes Konto: IBAN: %s", formatAmount(inv.TotalAmount), eeg.IBAN)
		}
		return fmt.Sprintf("Der Betrag von %s wird mittels SEPA-Lastschrift von Ihrem Konto eingezogen.", formatAmount(inv.TotalAmount))
	}
	return ""
}

// gesamtbetragLine returns the "Gesamtbetrag" line (label + formatted amount +
// trailing newline) for the invoice email body, labelling the payment direction
// the same way as the PDF's Saldo line (pdf.go/pdf_theme.go GeneratePDF): "EEG
// zahlt an Sie" when the EEG owes the member money, "Zahlungsbetrag" otherwise.
// Exposed as the {{gesamtbetrag_zeile}} placeholder.
func gesamtbetragLine(inv *domain.Invoice, member *domain.Member, isCredit bool) string {
	saldo := inv.TotalAmount
	if GenerationReverseCharge(member) {
		saldo += inv.GenerationVatAmount
	}
	if isCredit || saldo < 0 {
		amt := saldo
		if amt < 0 {
			amt = -amt
		}
		return fmt.Sprintf("Gesamtbetrag (EEG zahlt an Sie): %s\n", formatAmount(amt))
	}
	return fmt.Sprintf("Gesamtbetrag (Zahlungsbetrag): %s\n", formatAmount(saldo))
}

// buildPlainBody returns the plain-text invoice email body. If eeg.InvoiceEmailBody
// is set, it is used as a {{placeholder}} template instead of the default text.
func buildPlainBody(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, periodRange string, isCredit bool, webBaseURL string) string {
	if eeg.InvoiceEmailBody != "" {
		return applyInvoicePlaceholders(eeg.InvoiceEmailBody, inv, eeg, member, periodRange, isCredit, webBaseURL)
	}
	return buildDefaultPlainBody(inv, eeg, member, periodRange, isCredit, webBaseURL)
}

func buildDefaultPlainBody(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, periodRange string, isCredit bool, webBaseURL string) string {
	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	docLabel := "Rechnung"
	if isCredit {
		docLabel = "Gutschrift"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sehr geehrte/r %s,\n\n", fullName))
	sb.WriteString(fmt.Sprintf("anbei erhalten Sie Ihre %s für den Abrechnungszeitraum\n", docLabel))
	sb.WriteString(fmt.Sprintf("%s.\n\n", periodRange))
	sb.WriteString(fmt.Sprintf("Energiegemeinschaft:  %s\n", eeg.DisplayNameOrName()))
	sb.WriteString(fmt.Sprintf("Mitgliedsnummer:      %s\n", member.MitgliedsNr))
	if inv.InvoiceNumber != nil {
		prefix := eeg.InvoiceNumberPrefix
		digits := eeg.InvoiceNumberDigits
		if isCredit {
			prefix = eeg.CreditNoteNumberPrefix
			digits = eeg.CreditNoteNumberDigits
		}
		if digits == 0 {
			digits = 5
		}
		nr := fmt.Sprintf("%s%0*d", prefix, digits, *inv.InvoiceNumber)
		sb.WriteString(fmt.Sprintf("%snummer:       %s\n", docLabel, nr))
	}
	if inv.ConsumptionKwh > 0 {
		sb.WriteString(fmt.Sprintf("Bezug:                %s kWh\n", formatKwh(inv.ConsumptionKwh)))
	}
	if inv.GenerationKwh > 0 {
		sb.WriteString(fmt.Sprintf("Einspeisung:          %s kWh\n", formatKwh(inv.GenerationKwh)))
	}
	sb.WriteString(gesamtbetragLine(inv, member, isCredit) + "\n")

	if notice := paymentNoticeText(inv, eeg, isCredit); notice != "" {
		sb.WriteString(notice + "\n\n")
	}

	sb.WriteString("Die Details entnehmen Sie bitte dem beigefügten PDF.\n\n")
	if webBaseURL != "" {
		sb.WriteString(fmt.Sprintf("Im Mitgliederportal (%s/portal) können Sie jederzeit Ihre Energiedaten und bisherigen Rechnungen einsehen.\n\n", webBaseURL))
	}
	sb.WriteString("Mit freundlichen Grüßen\n")
	sb.WriteString(eeg.DisplayNameOrName() + "\n")
	sb.WriteString("\n-- \nErstellt von eegabrechnung\n")
	return sb.String()
}

func buildMIMEMessage(from, to, subject, plainBody, attachmentName string, pdfData []byte) ([]byte, error) {
	var buf bytes.Buffer

	mw := multipart.NewWriter(&buf)
	boundary := mw.Boundary()

	buf.Reset()
	var header bytes.Buffer
	header.WriteString(mailutil.Headers(from, to, subject))
	header.WriteString("MIME-Version: 1.0\r\n")
	header.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n")
	header.WriteString("\r\n")

	pw, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, fmt.Errorf("create text part: %w", err)
	}
	if _, err := pw.Write([]byte(plainBody)); err != nil {
		return nil, fmt.Errorf("write text part: %w", err)
	}

	ah := textproto.MIMEHeader{
		"Content-Type":              {"application/pdf"},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {fmt.Sprintf("attachment; filename=\"%s\"", attachmentName)},
	}
	aw, err := mw.CreatePart(ah)
	if err != nil {
		return nil, fmt.Errorf("create pdf part: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(pdfData)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		if _, err := aw.Write([]byte(encoded[i:end] + "\r\n")); err != nil {
			return nil, fmt.Errorf("write pdf part: %w", err)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	result := append(header.Bytes(), buf.Bytes()...)
	return result, nil
}
