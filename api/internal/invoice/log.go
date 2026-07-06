package invoice

import (
	"context"
	"net/smtp"
	"strings"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// EmailLogger is implemented by repository.EmailLogRepository.
type EmailLogger interface {
	Log(ctx context.Context, e domain.EmailLogEntry)
}

// SendLogged sends an email via smtp and records the attempt in the email log.
// eegID, emailType, subject, memberID, and invoiceID are metadata for the log entry.
func SendLogged(
	ctx context.Context,
	logger EmailLogger,
	cfg SMTPConfig,
	eegID uuid.UUID,
	emailType string,
	to string,
	subject string,
	memberID *uuid.UUID,
	invoiceID *uuid.UUID,
	msgBytes []byte,
) error {
	var auth smtp.Auth
	if cfg.Username != "" {
		host := cfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}

	sendErr := smtp.SendMail(cfg.Host, auth, cfg.From, []string{to}, msgBytes)

	entry := domain.EmailLogEntry{
		EegID:     eegID,
		EmailType: emailType,
		ToAddress: to,
		Subject:   subject,
		MemberID:  memberID,
		InvoiceID: invoiceID,
	}
	if sendErr != nil {
		entry.Status = "failed"
		entry.ErrorMsg = sendErr.Error()
	} else {
		entry.Status = "sent"
	}
	logger.Log(ctx, entry)

	return sendErr
}
