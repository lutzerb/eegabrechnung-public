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
func InvoiceSubject(inv *domain.Invoice, eeg *domain.EEG) string {
	docLabel := "Rechnung"
	if inv.DocumentType == "credit_note" {
		docLabel = "Gutschrift"
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
	subject := InvoiceSubject(inv, eeg)
	periodRange := inv.PeriodStart.Format("02.01.2006") + " – " + inv.PeriodEnd.Format("02.01.2006")
	attachmentName := fmt.Sprintf("%s_%s.pdf", docLabel, shortID(inv.ID.String()))
	body := buildPlainBody(inv, eeg, member, periodRange, isCredit, webBaseURL)
	return buildMIMEMessage(from, toEmail, subject, body, attachmentName, pdfData)
}

// SendInvoice sends the invoice PDF to the member's email address and logs the attempt.
func SendInvoice(ctx context.Context, logger EmailLogger, cfg SMTPConfig, member *domain.Member, eeg *domain.EEG, inv *domain.Invoice, pdfData []byte, webBaseURL string) error {
	msgBytes, err := BuildInvoiceMessage(cfg.From, member.Email, inv, eeg, member, pdfData, webBaseURL)
	if err != nil {
		return fmt.Errorf("build mime message: %w", err)
	}
	memID := member.ID
	invID := inv.ID
	return SendLogged(ctx, logger, cfg, eeg.ID, "invoice", member.Email, InvoiceSubject(inv, eeg), &memID, &invID, msgBytes)
}

func buildPlainBody(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, periodRange string, isCredit bool, webBaseURL string) string {
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
	sb.WriteString(fmt.Sprintf("Gesamtbetrag:         %s\n\n", formatAmount(inv.TotalAmount)))

	if isCredit {
		amount := inv.TotalAmount
		if amount < 0 {
			amount = -amount
		}
		sb.WriteString(fmt.Sprintf("Der Betrag von %s wird auf Ihr hinterlegtes Konto überwiesen.\n\n", formatAmount(amount)))
	} else if inv.TotalAmount < 0 {
		sb.WriteString(fmt.Sprintf("Der Betrag von %s wird auf Ihr hinterlegtes Konto überwiesen.\n\n", formatAmount(-inv.TotalAmount)))
	} else if inv.TotalAmount > 0 && eeg.InvoicePaymentNoticeMode != "none" {
		if eeg.InvoicePaymentNoticeMode == "ueberweisung" {
			sb.WriteString(fmt.Sprintf("Bitte überweisen Sie den Betrag von %s auf folgendes Konto: IBAN: %s\n\n", formatAmount(inv.TotalAmount), eeg.IBAN))
		} else {
			sb.WriteString(fmt.Sprintf("Der Betrag von %s wird mittels SEPA-Lastschrift von Ihrem Konto eingezogen.\n\n", formatAmount(inv.TotalAmount)))
		}
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
