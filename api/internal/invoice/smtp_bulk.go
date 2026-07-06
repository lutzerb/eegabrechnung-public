package invoice

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// BulkSender sends multiple messages over a single SMTP connection, avoiding
// the connect+TLS+auth round-trip that net/smtp.SendMail pays for every call.
type BulkSender struct {
	client *smtp.Client
	from   string
	dead   bool
}

// NewBulkSender dials cfg.Host, upgrades to TLS via STARTTLS if offered, and
// authenticates once. The connection is reused by subsequent Send calls.
func NewBulkSender(cfg SMTPConfig) (*BulkSender, error) {
	c, err := smtp.Dial(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.Host, err)
	}

	host := cfg.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			c.Close()
			return nil, fmt.Errorf("starttls: %w", err)
		}
	}

	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, host)
		if err := c.Auth(auth); err != nil {
			c.Close()
			return nil, fmt.Errorf("auth: %w", err)
		}
	}

	return &BulkSender{client: c, from: cfg.From}, nil
}

// Send transmits one message to a single recipient over the shared connection.
// Returns ErrConnectionDead if the connection could not be recovered after a
// failure — the caller should stop issuing further Send calls in that case.
func (b *BulkSender) Send(to string, msgBytes []byte) error {
	if b.dead {
		return errConnectionDead
	}

	sendErr := b.sendOnce(to, msgBytes)
	if sendErr == nil {
		return nil
	}

	// Try to reset the transaction state so the connection can be reused for
	// the next message; if that also fails the connection is unusable.
	if resetErr := b.client.Reset(); resetErr != nil {
		b.dead = true
	}
	return sendErr
}

func (b *BulkSender) sendOnce(to string, msgBytes []byte) error {
	if err := b.client.Mail(b.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := b.client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	wc, err := b.client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := wc.Write(msgBytes); err != nil {
		wc.Close()
		return fmt.Errorf("write message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("close message: %w", err)
	}
	return nil
}

// Dead reports whether the connection is no longer usable.
func (b *BulkSender) Dead() bool {
	return b.dead
}

// Close terminates the SMTP session.
func (b *BulkSender) Close() error {
	if b.dead {
		return b.client.Close()
	}
	return b.client.Quit()
}

var errConnectionDead = fmt.Errorf("smtp connection unavailable")
