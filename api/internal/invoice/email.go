package invoice

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host     string // e.g. "eegabrechnung-postfix:25" or "localhost:25"
	From     string // e.g. "noreply@eeg.at"
	Username string // empty = no auth (Postfix relay)
	Password string
}

// SendInvoice sends the invoice PDF to the member's email address.
func SendInvoice(cfg SMTPConfig, member *domain.Member, eeg *domain.EEG, inv *domain.Invoice, pdfData []byte) error {
	periodRange := fmt.Sprintf("%s – %s",
		inv.PeriodStart.Format("02.01.2006"),
		inv.PeriodEnd.Format("02.01.2006"),
	)
	isCredit := inv.DocumentType == "credit_note"
	docLabel := "Rechnung"
	if isCredit {
		docLabel = "Gutschrift"
	}
	subject := fmt.Sprintf("Ihre %s – %s – %s", docLabel, eeg.Name, periodRange)
	invoiceNr := shortID(inv.ID.String())
	attachmentName := fmt.Sprintf("%s_%s.pdf", docLabel, invoiceNr)

	// Build plain text body
	body := buildPlainBody(inv, eeg, member, periodRange, isCredit)

	// Build MIME message
	msgBytes, err := buildMIMEMessage(cfg.From, member.Email, subject, body, attachmentName, pdfData)
	if err != nil {
		return fmt.Errorf("build mime message: %w", err)
	}

	// Determine auth
	var auth smtp.Auth
	if cfg.Username != "" {
		host := cfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}

	to := []string{member.Email}
	return smtp.SendMail(cfg.Host, auth, cfg.From, to, msgBytes)
}

func buildPlainBody(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, periodRange string, isCredit bool) string {
	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	docLabel := "Rechnung"
	if isCredit {
		docLabel = "Gutschrift"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sehr geehrte/r %s,\n\n", fullName))
	sb.WriteString(fmt.Sprintf("anbei erhalten Sie Ihre %s für den Abrechnungszeitraum\n", docLabel))
	sb.WriteString(fmt.Sprintf("%s.\n\n", periodRange))
	sb.WriteString(fmt.Sprintf("Energiegemeinschaft:  %s\n", eeg.Name))
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

	// Payment notice — matches the PDF text
	if isCredit {
		// Actual credit note (Storno): always an outgoing transfer
		amount := inv.TotalAmount
		if amount < 0 {
			amount = -amount
		}
		sb.WriteString(fmt.Sprintf("Der Betrag von %s wird auf Ihr hinterlegtes Konto überwiesen.\n\n", formatAmount(amount)))
	} else if inv.TotalAmount < 0 {
		// Regular invoice with negative saldo (e.g. producer gets more than they owe)
		sb.WriteString(fmt.Sprintf("Der Betrag von %s wird auf Ihr hinterlegtes Konto überwiesen.\n\n", formatAmount(-inv.TotalAmount)))
	} else if inv.TotalAmount > 0 {
		sb.WriteString(fmt.Sprintf("Der Betrag von %s wird mittels SEPA-Lastschrift von Ihrem Konto eingezogen.\n\n", formatAmount(inv.TotalAmount)))
	}

	sb.WriteString("Die Details entnehmen Sie bitte dem beigefügten PDF.\n\n")
	sb.WriteString("Mit freundlichen Grüßen\n")
	sb.WriteString(eeg.Name + "\n")
	sb.WriteString("\n-- \nErstellt von eegabrechnung\n")
	return sb.String()
}

func buildMIMEMessage(from, to, subject, plainBody, attachmentName string, pdfData []byte) ([]byte, error) {
	var buf bytes.Buffer

	// Top-level multipart/mixed writer
	mw := multipart.NewWriter(&buf)
	boundary := mw.Boundary()

	// Headers
	buf.Reset() // reset before writing headers so they come first
	var header bytes.Buffer
	header.WriteString("From: " + from + "\r\n")
	header.WriteString("To: " + to + "\r\n")
	header.WriteString("Subject: " + subject + "\r\n")
	header.WriteString("MIME-Version: 1.0\r\n")
	header.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n")
	header.WriteString("\r\n")

	// Plain text part
	pw, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, fmt.Errorf("create text part: %w", err)
	}
	if _, err := pw.Write([]byte(plainBody)); err != nil {
		return nil, fmt.Errorf("write text part: %w", err)
	}

	// PDF attachment part
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
	// Write encoded in 76-char lines per MIME spec
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

	// Combine headers + body
	result := append(header.Bytes(), buf.Bytes()...)
	return result, nil
}
