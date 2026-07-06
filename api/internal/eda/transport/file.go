package transport

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/types"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// FileTransport is a Transport that reads XML files from InboxDir on Poll()
// and writes outbound XML to OutboxDir on Send(). Useful for local development
// and testing without a real IMAP/SMTP server.
//
// Inbox layout:
//
//	<InboxDir>/*.xml         — files to be consumed on next Poll()
//	<InboxDir>/processed/   — files moved here after Poll() reads them
type FileTransport struct {
	InboxDir  string
	OutboxDir string
	log       *slog.Logger
}

// NewFileTransport creates a FileTransport and ensures all required directories exist.
func NewFileTransport(inboxDir, outboxDir string, log *slog.Logger) (*FileTransport, error) {
	for _, dir := range []string{inboxDir, filepath.Join(inboxDir, "processed"), outboxDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %q: %w", dir, err)
		}
	}
	return &FileTransport{InboxDir: inboxDir, OutboxDir: outboxDir, log: log}, nil
}

// Poll reads all *.xml files from InboxDir, parses them into Messages, and
// moves each file to InboxDir/processed/ so it is not re-consumed.
func (t *FileTransport) Poll(_ context.Context) ([]*types.Message, error) {
	entries, err := os.ReadDir(t.InboxDir)
	if err != nil {
		return nil, fmt.Errorf("read inbox dir %q: %w", t.InboxDir, err)
	}

	var msgs []*types.Message
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".xml") {
			continue
		}

		path := filepath.Join(t.InboxDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.log.Warn("EDA inbox: failed to read file", "file", path, "error", err)
			continue
		}

		xmlPayload := string(data)
		process := detectFileProcess(xmlPayload)

		msgs = append(msgs, &types.Message{
			ID:         entry.Name(),
			Process:    process,
			Direction:  types.DirectionInbound,
			XMLPayload: xmlPayload,
			CreatedAt:  time.Now().UTC(),
		})

		// Move to processed/ so it is not read again.
		dest := filepath.Join(t.InboxDir, "processed", entry.Name())
		if err := os.Rename(path, dest); err != nil {
			t.log.Warn("EDA inbox: failed to move processed file", "src", path, "dst", dest, "error", err)
		} else {
			t.log.Info("EDA inbox: file consumed", "file", entry.Name(), "process", process)
		}
	}

	return msgs, nil
}

// Send writes the outbound message XML to OutboxDir with a timestamped filename.
func (t *FileTransport) Send(_ context.Context, msg *types.Message) error {
	ts := time.Now().UTC().Format("20060102T150405Z")
	filename := fmt.Sprintf("%s_%s.xml", ts, msg.Process)
	path := filepath.Join(t.OutboxDir, filename)

	if err := os.WriteFile(path, []byte(msg.XMLPayload), 0o644); err != nil {
		return fmt.Errorf("write outbox file %q: %w", path, err)
	}
	t.log.Info("EDA outbox: message written", "file", path, "process", msg.Process)
	return nil
}

// SendAck writes an acknowledgement file to OutboxDir (FILE transport mock).
func (t *FileTransport) SendAck(_ context.Context, originalMsgID, from, to string) error {
	ts := time.Now().UTC().Format("20060102T150405Z")
	filename := fmt.Sprintf("%s_ACK.xml", ts)
	path := filepath.Join(t.OutboxDir, filename)
	content := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<Acknowledgement xmlns="http://www.ebutilities.at/schemata/customerprocesses/acknowledgement/01p00">` + "\n" +
		`  <RefToMessageId>` + originalMsgID + `</RefToMessageId>` + "\n" +
		`  <From>` + from + `</From>` + "\n" +
		`  <To>` + to + `</To>` + "\n" +
		`  <Timestamp>` + time.Now().UTC().Format(time.RFC3339) + `</Timestamp>` + "\n" +
		`</Acknowledgement>`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write ack file: %w", err)
	}
	t.log.Info("EDA outbox: Ack written", "file", path, "ref", originalMsgID)
	return nil
}

// detectFileProcess determines the EDA process type from the raw XML payload.
func detectFileProcess(xmlPayload string) string {
	if edaxml.IsCRMsg(xmlPayload) {
		return types.ProcessCRMsg
	}
	if edaxml.IsCPDocument(xmlPayload) {
		return "CPDOCUMENT"
	}
	return "UNKNOWN"
}
