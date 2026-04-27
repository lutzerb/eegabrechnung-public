package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/eda/types"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// MailConfig holds IMAP and SMTP configuration.
type MailConfig struct {
	IMAPHost     string
	IMAPUser     string
	IMAPPassword string
	IMAPPlain    bool // use plain (non-TLS) IMAP — for test servers like Mailpit
	SMTPHost     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPInsecure bool // skip SMTP auth — for test servers like Mailpit (no auth required)
}

// MailTransport implements types.Transport using IMAP (receive) and SMTP (send).
type MailTransport struct {
	cfg MailConfig
	log *slog.Logger
}

// NewMailTransport creates a new MailTransport. Returns an error if required
// credentials are missing, but the caller may decide to log and continue.
func NewMailTransport(cfg MailConfig, log *slog.Logger) (*MailTransport, error) {
	if cfg.IMAPHost == "" || cfg.IMAPUser == "" || cfg.IMAPPassword == "" {
		return nil, fmt.Errorf("IMAP credentials not configured (EDA_IMAP_HOST, EDA_IMAP_USER, EDA_IMAP_PASSWORD)")
	}
	if cfg.SMTPInsecure {
		// Insecure mode (test servers): only SMTPHost and SMTPFrom are required.
		if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
			return nil, fmt.Errorf("SMTP host/from not configured (EDA_SMTP_HOST, EDA_SMTP_FROM)")
		}
	} else {
		if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPassword == "" || cfg.SMTPFrom == "" {
			return nil, fmt.Errorf("SMTP credentials not configured (EDA_SMTP_HOST, EDA_SMTP_USER, EDA_SMTP_PASSWORD, EDA_SMTP_FROM)")
		}
	}
	return &MailTransport{cfg: cfg, log: log}, nil
}

// smtpTimeout is the total timeout for a single SMTP send operation.
const smtpTimeout = 30 * time.Second

// edanetProzessID maps internal EDA process codes to the Prozess-Id string
// required in the edanet.at email Subject header:
// [<Prozess-Id> MessageId=<MessageId>]
// See: https://www.ebutilities.at/ for the full process list.
var edanetProzessID = map[string]string{
	"EC_PRTFACT_CHG": "EC_PRTFACT_CHANGE_01.00",
	"EC_REQ_ONL":     "EC_REQ_ONL_02.30",
	"EC_REQ_PT":      "CR_REQ_PT_04.10",
	"EC_PODLIST":     "EC_PODLIST_01.00",
	"CM_REV_SP":      "CM_REV_SP_01.00",
}

// edanetAddress appends @edanet.at to a bare Marktpartner-ID.
// Full addresses (containing @) are returned unchanged.
func edanetAddress(id string) string {
	if strings.Contains(id, "@") {
		return id
	}
	return id + "@edanet.at"
}

// Send sends an EDA message via SMTP.
// The email format follows the edanet.at gateway specification:
//   - To: <ATCode>@edanet.at
//   - Subject: [<Prozess-Id> MessageId=<MessageId>]
//   - Body: XML payload as attachment
func (t *MailTransport) Send(ctx context.Context, msg *types.Message) error {
	prozessID := edanetProzessID[msg.Process]
	if prozessID == "" {
		prozessID = msg.Process
	}
	msgID := uuid.NewString()
	subject := fmt.Sprintf("[%s MessageId=%s]", prozessID, msgID)
	to := edanetAddress(msg.To)
	body := buildMIMEMessage(t.cfg.SMTPFrom, to, subject, msg.XMLPayload)

	if err := t.smtpSend(ctx, to, body); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	// Write the actual email subject back so callers can store it.
	msg.Subject = subject
	t.log.Info("EDA message sent via SMTP",
		"process", msg.Process,
		"to", to,
		"subject", subject,
		"gemeinschaft_id", msg.GemeinschaftID,
	)
	return nil
}

// smtpSend delivers a pre-built MIME body to a single recipient.
// It supports both implicit TLS (port 465, SMTPS) and STARTTLS (port 587/25).
// A 30-second context deadline is applied so the call never blocks indefinitely.
func (t *MailTransport) smtpSend(ctx context.Context, to string, body []byte) error {
	host, port, err := net.SplitHostPort(t.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("invalid SMTP host %q: %w", t.cfg.SMTPHost, err)
	}

	ctx, cancel := context.WithTimeout(ctx, smtpTimeout)
	defer cancel()

	var conn net.Conn
	dialer := &net.Dialer{}
	if port == "465" {
		// SMTPS: implicit TLS from the start.
		tlsDialer := &tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{ServerName: host},
		}
		conn, err = tlsDialer.DialContext(ctx, "tcp", t.cfg.SMTPHost)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", t.cfg.SMTPHost)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if port != "465" {
		// STARTTLS for submission ports (587, 25).
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if !t.cfg.SMTPInsecure {
		if err := client.Auth(smtp.PlainAuth("", t.cfg.SMTPUser, t.cfg.SMTPPassword, host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(t.cfg.SMTPFrom); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

// imapTimeout is the total time budget for a single IMAP poll cycle.
const imapTimeout = 90 * time.Second

// Poll connects to IMAP, fetches unseen messages, marks them as seen, and
// returns parsed EDA messages. Parsing errors are logged but not fatal.
func (t *MailTransport) Poll(ctx context.Context) ([]*types.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, imapTimeout)
	defer cancel()

	var c *imapclient.Client

	if t.cfg.IMAPPlain {
		// Plain (non-TLS) IMAP — for test servers like Mailpit.
		var err error
		c, err = imapclient.DialInsecure(t.cfg.IMAPHost, nil)
		if err != nil {
			return nil, fmt.Errorf("imap dial: %w", err)
		}
	} else {
		// imapclient.DialTLS has no context parameter and blocks forever if the
		// server is slow. Wrap it in a goroutine and use select to enforce the
		// context timeout over the entire dial + IMAP greeting phase.
		type dialResult struct {
			c   *imapclient.Client
			err error
		}
		ch := make(chan dialResult, 1)
		go func() {
			opts := &imapclient.Options{
				TLSConfig: &tls.Config{ServerName: imapHostname(t.cfg.IMAPHost)},
			}
			rc, rerr := imapclient.DialTLS(t.cfg.IMAPHost, opts)
			ch <- dialResult{rc, rerr}
		}()
		select {
		case r := <-ch:
			if r.err != nil {
				return nil, fmt.Errorf("imap dial: %w", r.err)
			}
			c = r.c
		case <-ctx.Done():
			return nil, fmt.Errorf("imap dial timeout: %w", ctx.Err())
		}
	}
	defer c.Logout()

	// Close the connection when the context is cancelled (or times out).
	// go-imap v2 Wait() calls do not respect context cancellation on their own,
	// so we force-close the client to unblock any pending command.
	go func() {
		<-ctx.Done()
		c.Close()
	}()

	if err := c.Login(t.cfg.IMAPUser, t.cfg.IMAPPassword).Wait(); err != nil {
		return nil, fmt.Errorf("imap login: %w", err)
	}

	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("imap select INBOX: %w", err)
	}

	// Search for unseen messages.
	searchData, err := c.Search(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("imap search: %w", err)
	}

	if len(searchData.AllSeqNums()) == 0 {
		t.log.Info("IMAP poll complete", "messages_received", 0)
		return nil, nil
	}

	allSeqSet := imap.SeqSetNum(searchData.AllSeqNums()...)
	fetchOptions := &imap.FetchOptions{
		Flags:    true,
		Envelope: true,
		// Use PEEK so the server does NOT auto-mark messages as \Seen on fetch.
		// We explicitly mark them as \Seen only after successful processing.
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}

	fetchCmd := c.Fetch(allSeqSet, fetchOptions)
	defer fetchCmd.Close()

	// processedSeqNums tracks every message we handled (success or error) so we
	// can mark exactly those as \Seen. Messages not in this list were never read
	// from the stream and remain unread for the next poll cycle.
	var processedSeqNums []uint32
	var msgs []*types.Message

	for {
		fetchMsg := fetchCmd.Next()
		if fetchMsg == nil {
			break
		}
		seqNum := fetchMsg.SeqNum

		var rawBody []byte
		var messageID, emailSubject string

		for {
			item := fetchMsg.Next()
			if item == nil {
				break
			}
			switch it := item.(type) {
			case imapclient.FetchItemDataBodySection:
				data, readErr := io.ReadAll(it.Literal)
				if readErr == nil {
					rawBody = data
				}
			case imapclient.FetchItemDataEnvelope:
				if it.Envelope != nil {
					messageID = it.Envelope.MessageID
					emailSubject = it.Envelope.Subject
				}
			}
		}

		if messageID == "" {
			messageID = uuid.NewString()
		}

		if len(rawBody) == 0 {
			t.log.Warn("IMAP message has empty body — storing as parse error", "message_id", messageID)
			processedSeqNums = append(processedSeqNums, seqNum)
			msgs = append(msgs, &types.Message{
				ID:        messageID,
				Process:   types.ProcessParseError,
				Direction: types.DirectionInbound,
				Subject:   emailSubject,
				XMLPayload: "(empty IMAP message body)",
				CreatedAt: time.Now().UTC(),
			})
			continue
		}

		xmlBody, emailBody := extractPartsFromMIME(rawBody)
		if xmlBody == "" {
			t.log.Warn("no XML body found in IMAP message — storing as parse error", "message_id", messageID)
			processedSeqNums = append(processedSeqNums, seqNum)
			msgs = append(msgs, &types.Message{
				ID:         messageID,
				Process:    types.ProcessParseError,
				Direction:  types.DirectionInbound,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: string(rawBody),
				CreatedAt:  time.Now().UTC(),
			})
			continue
		}

		var edaMsg *types.Message

		if edaxml.IsCRMsg(xmlBody) {
			record, parseErr := edaxml.ParseCRMsg(xmlBody)
			if parseErr != nil {
				t.log.Warn("failed to parse CR_MSG from email — storing as parse error",
					"message_id", messageID,
					"error", parseErr,
				)
				processedSeqNums = append(processedSeqNums, seqNum)
				msgs = append(msgs, &types.Message{
					ID:         messageID,
					Process:    types.ProcessParseError,
					Direction:  types.DirectionInbound,
					Subject:    emailSubject,
					EmailBody:  emailBody,
					XMLPayload: xmlBody,
					CreatedAt:  time.Now().UTC(),
				})
				continue
			}
			edaMsg = &types.Message{
				Process:        types.ProcessCRMsg,
				Direction:      types.DirectionInbound,
				From:           record.From,
				To:             record.To,
				GemeinschaftID: record.GemeinschaftID,
				Subject:        emailSubject,
				EmailBody:      emailBody,
				XMLPayload:     xmlBody,
				CreatedAt:      time.Now().UTC(),
			}
		} else if edaxml.IsCPNotification(xmlBody) {
			// CPNotification = edanet transport-level delivery confirmation.
			// Pass through as-is without attempting to parse as Anforderung.
			notif := edaxml.ParseCPNotification(xmlBody)
			edaMsg = &types.Message{
				Process:    "ANTWORT_PT",
				Direction:  types.DirectionInbound,
				From:       notif.From,
				To:         notif.To,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: xmlBody,
				CreatedAt:  time.Now().UTC(),
			}
		} else if edaxml.IsCMNotification(xmlBody) {
			// CMNotification = Netzbetreiber consent response (ZUSTIMMUNG_ECON / ABLEHNUNG_ECON).
			edaMsg = &types.Message{
				Process:    "CMNotification",
				Direction:  types.DirectionInbound,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: xmlBody,
				CreatedAt:  time.Now().UTC(),
			}
			if r, err := edaxml.ParseCMNotification(xmlBody); err == nil {
				edaMsg.From = r.From
				edaMsg.To = r.To
			}
		} else if edaxml.IsECMPList(xmlBody) {
			// ECMPList = Zählpunktliste response from Netzbetreiber.
			edaMsg = &types.Message{
				Process:    "ECMPList",
				Direction:  types.DirectionInbound,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: xmlBody,
				CreatedAt:  time.Now().UTC(),
			}
			if r, err := edaxml.ParseECMPList(xmlBody); err == nil {
				edaMsg.From = r.From
				edaMsg.To = r.To
			}
		} else if edaxml.IsEDASendError(xmlBody) {
			// EDASendError = edanet gateway XML validation rejection.
			edaMsg = &types.Message{
				Process:    "EDASendError",
				Direction:  types.DirectionInbound,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: xmlBody,
				CreatedAt:  time.Now().UTC(),
			}
		} else if edaxml.IsCPDocument(xmlBody) {
			// CPDocument = CPDocument confirmation from edanet.
			edaMsg = &types.Message{
				Process:    "CPDocument",
				Direction:  types.DirectionInbound,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: xmlBody,
				CreatedAt:  time.Now().UTC(),
			}
			if r, err := edaxml.ParseCPDocument(xmlBody); err == nil {
				edaMsg.From = r.From
				edaMsg.To = r.To
			}
		} else if edaxml.IsCMRevoke(xmlBody) {
			// CMRevoke = Netzbetreiber notifies us of consent revocation.
			// CM_REV_CUS (AUFHEBUNG_CCMS): customer revoked voluntarily.
			// CM_REV_IMP (AUFHEBUNG_CCMS_IMP): Netzbetreiber revokes due to impossibility.
			process := "CM_REV_CUS"
			edaMsg = &types.Message{
				Process:    process,
				Direction:  types.DirectionInbound,
				Subject:    emailSubject,
				EmailBody:  emailBody,
				XMLPayload: xmlBody,
				CreatedAt:  time.Now().UTC(),
			}
			if r, err := edaxml.ParseCMRevoke(xmlBody); err == nil {
				edaMsg.From = r.From
				edaMsg.To = r.To
				if r.MessageCode == "AUFHEBUNG_CCMS_IMP" {
					edaMsg.Process = "CM_REV_IMP"
				}
			}
		} else {
			var parseErr error
			edaMsg, parseErr = edaxml.ParseAnforderung(xmlBody)
			if parseErr != nil {
				t.log.Warn("failed to parse EDA XML from email — storing as parse error",
					"message_id", messageID,
					"error", parseErr,
				)
				processedSeqNums = append(processedSeqNums, seqNum)
				msgs = append(msgs, &types.Message{
					ID:         messageID,
					Process:    types.ProcessParseError,
					Direction:  types.DirectionInbound,
					Subject:    emailSubject,
					EmailBody:  emailBody,
					XMLPayload: xmlBody,
					CreatedAt:  time.Now().UTC(),
				})
				continue
			}
			edaMsg.Subject = emailSubject
			edaMsg.EmailBody = emailBody
		}

		edaMsg.ID = messageID
		processedSeqNums = append(processedSeqNums, seqNum)
		msgs = append(msgs, edaMsg)
	}

	if err := fetchCmd.Close(); err != nil {
		t.log.Warn("imap fetch close error", "error", err)
	}

	// Mark only the messages we actually processed (success + parse errors) as \Seen.
	// Messages that could not be streamed at all remain unseen for the next poll.
	if len(processedSeqNums) > 0 {
		seenSet := imap.SeqSetNum(processedSeqNums...)
		storeFlags := imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Flags:  []imap.Flag{imap.FlagSeen},
			Silent: true,
		}
		if err := c.Store(seenSet, &storeFlags, nil).Close(); err != nil {
			t.log.Warn("failed to mark messages as seen", "error", err)
		}
	}

	t.log.Info("IMAP poll complete", "messages_received", len(msgs))
	return msgs, nil
}

// SendAck sends an ebMS 2.0 acknowledgement back to the message sender via SMTP.
// The Ack references the MessageID of the original inbound message.
func (t *MailTransport) SendAck(ctx context.Context, originalMsgID, from, to string) error {
	ackMsgID := uuid.NewString()
	subject := fmt.Sprintf("[ebutilites3.00 MessageId=%s]", ackMsgID)
	toAddr := edanetAddress(to)
	ackXML := buildAckXML(originalMsgID, t.cfg.SMTPFrom, to)
	body := buildMIMEMessage(t.cfg.SMTPFrom, toAddr, subject, ackXML)
	if err := t.smtpSend(ctx, toAddr, body); err != nil {
		return fmt.Errorf("smtp send ack: %w", err)
	}
	t.log.Info("EDA Ack sent", "original_msg_id", originalMsgID, "to", toAddr)
	return nil
}

// buildAckXML builds a minimal ebMS 2.0 acknowledgement XML referencing the original MessageID.
func buildAckXML(originalMsgID, from, to string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<Acknowledgement xmlns="http://www.ebutilities.at/schemata/customerprocesses/acknowledgement/01p00">` + "\n" +
		`  <RefToMessageId>` + originalMsgID + `</RefToMessageId>` + "\n" +
		`  <From>` + from + `</From>` + "\n" +
		`  <To>` + to + `</To>` + "\n" +
		`  <Timestamp>` + time.Now().UTC().Format(time.RFC3339) + `</Timestamp>` + "\n" +
		`</Acknowledgement>`
}

// buildMIMEMessage constructs a multipart/mixed MIME email with the XML as an
// attachment. The edanet.at gateway requires a proper MIME attachment — sending
// the XML as the message body causes "kein Anhang gefunden" errors.
func buildMIMEMessage(from, to, subject, xmlBody string) []byte {
	boundary := "edaboundary-" + uuid.NewString()
	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n")
	buf.WriteString("\r\n")
	// XML attachment part
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: application/xml; charset=UTF-8\r\n")
	buf.WriteString("Content-Disposition: attachment; filename=\"mako.xml\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(xmlBody)
	buf.WriteString("\r\n--" + boundary + "--\r\n")
	return buf.Bytes()
}

// extractPartsFromMIME parses a raw RFC822 MIME email and returns the first XML
// body/attachment and the plain-text body (if any), properly decoding
// Content-Transfer-Encoding for each part.
func extractPartsFromMIME(raw []byte) (xmlPart, textPart string) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		// Fallback: naive byte search for non-MIME messages.
		return xmlSearchString(string(raw)), ""
	}

	ct := msg.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		body, _ := io.ReadAll(msg.Body)
		decoded := string(decodeMIMEPart(body, msg.Header.Get("Content-Transfer-Encoding")))
		return xmlSearchString(decoded), ""
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		for {
			part, partErr := mr.NextRawPart()
			if partErr != nil {
				break
			}
			partCT := part.Header.Get("Content-Type")
			partMediaType, _, _ := mime.ParseMediaType(partCT)
			data, _ := io.ReadAll(part)
			decoded := string(decodeMIMEPart(data, part.Header.Get("Content-Transfer-Encoding")))
			if xml := xmlSearchString(decoded); xml != "" && xmlPart == "" {
				xmlPart = xml
			} else if partMediaType == "text/plain" && textPart == "" {
				textPart = strings.TrimSpace(decoded)
			}
		}
		return xmlPart, textPart
	}

	body, _ := io.ReadAll(msg.Body)
	decoded := string(decodeMIMEPart(body, msg.Header.Get("Content-Transfer-Encoding")))
	if strings.HasPrefix(mediaType, "text/plain") {
		return xmlSearchString(decoded), strings.TrimSpace(decoded)
	}
	return xmlSearchString(decoded), ""
}

// decodeMIMEPart decodes a MIME body part according to its Content-Transfer-Encoding.
func decodeMIMEPart(data []byte, cte string) []byte {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		// Strip whitespace (line breaks within base64 blocks).
		clean := bytes.ReplaceAll(data, []byte("\r\n"), []byte{})
		clean = bytes.ReplaceAll(clean, []byte("\n"), []byte{})
		decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(clean)))
		if err != nil {
			// Try without padding.
			decoded, _ = io.ReadAll(base64.NewDecoder(base64.RawStdEncoding, bytes.NewReader(clean)))
		}
		return decoded
	case "quoted-printable":
		decoded, _ := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(data)))
		return decoded
	default:
		return data
	}
}

// xmlSearchString looks for a known XML root marker in s and returns from that point on.
func xmlSearchString(s string) string {
	for _, marker := range []string{
		"<?xml",
		"<cr:ConsumptionRecord",
		"<ConsumptionRecord",
		"<req:Anforderung",
		"<Anforderung",
		"<CPNotification",
		"<cp:CMRevoke",
		"<CMRevoke",
	} {
		if idx := strings.Index(s, marker); idx >= 0 {
			return strings.TrimSpace(s[idx:])
		}
	}
	return ""
}

// imapHostname extracts the hostname from host:port.
func imapHostname(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return h
}
