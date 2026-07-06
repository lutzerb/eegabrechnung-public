package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/eda/types"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
)

// PontonConfig holds configuration for the Ponton XP transport.
type PontonConfig struct {
	// OutboundURL is the Ponton XP outbound endpoint.
	OutboundURL string
	// InboundURL is an optional Ponton XP inbound polling endpoint.
	// If empty, inbound messages must be received via the HTTP handler.
	InboundURL string
}

// PontonTransport implements types.Transport using Ponton XP HTTP/SOAP.
type PontonTransport struct {
	cfg    PontonConfig
	client *http.Client
	log    *slog.Logger

	// inboundCh receives messages pushed via the HTTP handler.
	inboundCh chan *types.Message
}

// NewPontonTransport creates a new PontonTransport.
func NewPontonTransport(cfg PontonConfig, log *slog.Logger) (*PontonTransport, error) {
	if cfg.OutboundURL == "" {
		return nil, fmt.Errorf("EDA_PONTON_URL is required for PONTON transport")
	}
	return &PontonTransport{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		log:       log,
		inboundCh: make(chan *types.Message, 100),
	}, nil
}

// Send sends an EDA message via HTTP POST to the Ponton XP outbound endpoint.
func (t *PontonTransport) Send(ctx context.Context, msg *types.Message) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.cfg.OutboundURL,
		bytes.NewReader([]byte(msg.XMLPayload)))
	if err != nil {
		return fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml; charset=UTF-8")
	req.Header.Set("X-EDA-Process", msg.Process)
	req.Header.Set("X-EDA-GemeinschaftID", msg.GemeinschaftID)
	req.Header.Set("X-EDA-From", msg.From)
	req.Header.Set("X-EDA-To", msg.To)

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("http POST to ponton: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ponton returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	t.log.Info("EDA message sent via Ponton",
		"process", msg.Process,
		"gemeinschaft_id", msg.GemeinschaftID,
		"status", resp.StatusCode,
	)
	return nil
}

// Poll checks for inbound messages. If InboundURL is set, it polls the Ponton
// inbox endpoint. Otherwise it drains messages received via InboundHandler.
func (t *PontonTransport) Poll(ctx context.Context) ([]*types.Message, error) {
	if t.cfg.InboundURL != "" {
		return t.pollPontonInbox(ctx)
	}
	// Drain the channel of any pushed messages.
	var msgs []*types.Message
	for {
		select {
		case m := <-t.inboundCh:
			msgs = append(msgs, m)
		default:
			return msgs, nil
		}
	}
}

// pollPontonInbox polls the Ponton XP inbound endpoint for queued messages.
func (t *PontonTransport) pollPontonInbox(ctx context.Context) ([]*types.Message, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.cfg.InboundURL, nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest inbound: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http GET ponton inbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ponton inbox returned HTTP %d", resp.StatusCode)
	}

	// Ponton inbound response: JSON array of {id, payload} objects.
	var items []struct {
		ID      string `json:"id"`
		Payload string `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode ponton inbound response: %w", err)
	}

	var msgs []*types.Message
	for _, item := range items {
		msg, parseErr := edaxml.ParseAnforderung(item.Payload)
		if parseErr != nil {
			t.log.Warn("failed to parse EDA XML from Ponton inbox",
				"ponton_id", item.ID,
				"error", parseErr,
			)
			continue
		}
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		msg.ID = item.ID
		msgs = append(msgs, msg)
	}
	t.log.Info("Ponton inbox poll complete", "messages_received", len(msgs))
	return msgs, nil
}

// SendAck is a no-op for Ponton transport — Ponton XP handles acknowledgements internally.
func (t *PontonTransport) SendAck(_ context.Context, originalMsgID, _, _ string) error {
	t.log.Debug("SendAck called on PontonTransport (no-op)", "ref", originalMsgID)
	return nil
}

// InboundHandler returns an http.HandlerFunc that accepts messages pushed by
// Ponton XP to our /eda/inbound endpoint.
func (t *PontonTransport) InboundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
		if err != nil {
			t.log.Error("failed to read ponton inbound body", "error", err)
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}

		msg, err := edaxml.ParseAnforderung(string(body))
		if err != nil {
			t.log.Warn("failed to parse inbound EDA XML",
				"error", err,
				"body_snippet", truncate(string(body), 200),
			)
			http.Error(w, "invalid XML payload", http.StatusBadRequest)
			return
		}

		msgID := r.Header.Get("X-EDA-MessageID")
		if msgID == "" {
			msgID = uuid.NewString()
		}
		msg.ID = msgID

		select {
		case t.inboundCh <- msg:
			t.log.Info("Ponton inbound message queued",
				"message_id", msgID,
				"process", msg.Process,
			)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.log.Warn("Ponton inbound channel full, dropping message", "message_id", msgID)
			http.Error(w, "server busy", http.StatusServiceUnavailable)
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
