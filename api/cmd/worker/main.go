package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/db"
	"github.com/lutzerb/eegabrechnung/internal/eda"
	"github.com/lutzerb/eegabrechnung/internal/eda/transport"
	"github.com/lutzerb/eegabrechnung/internal/eda/types"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	dsn := getEnv("DATABASE_URL", "postgres://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung?sslmode=disable")
	transportMode := strings.ToUpper(getEnv("EDA_TRANSPORT", "MAIL"))
	pollIntervalStr := getEnv("EDA_POLL_INTERVAL", "60s")

	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil {
		log.Warn("invalid EDA_POLL_INTERVAL, using 60s", "value", pollIntervalStr)
		pollInterval = 60 * time.Second
	}

	// Credential encryption key (AES-256, base64-encoded 32 bytes).
	encKey := mustDecodeEncKey(log, getEnv("CREDENTIAL_ENCRYPTION_KEY", ""))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect to database (runs migrations).
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Build transport.
	var tr types.Transport
	var pontonTr *transport.PontonTransport

	switch transportMode {
	case "FILE":
		inboxDir := getEnv("EDA_INBOX_DIR", "./test/eda-inbox")
		outboxDir := getEnv("EDA_OUTBOX_DIR", "./test/eda-outbox")
		ft, err := transport.NewFileTransport(inboxDir, outboxDir, log)
		if err != nil {
			log.Error("failed to create FILE transport", "error", err)
			os.Exit(1)
		}
		tr = ft
		log.Info("EDA transport: FILE", "inbox", inboxDir, "outbox", outboxDir)

	case "PONTON":
		cfg := transport.PontonConfig{
			OutboundURL: getEnv("EDA_PONTON_URL", "http://ponton-xp:6060/ponton/eda/webservice/outbound"),
			InboundURL:  getEnv("EDA_PONTON_INBOUND_URL", ""),
		}
		pt, err := transport.NewPontonTransport(cfg, log)
		if err != nil {
			log.Error("failed to create Ponton transport", "error", err)
			os.Exit(1)
		}
		pontonTr = pt
		tr = pt
		log.Info("EDA transport: PONTON", "url", cfg.OutboundURL)

	default: // MAIL — per-EEG transports are created dynamically from DB credentials
		log.Info("EDA transport: MAIL (per-EEG credentials from DB)")
		tr = &noopTransport{log: log} // fallback; never used directly in MAIL mode
	}

	webBaseURL := getEnv("WEB_BASE_URL", "")

	// Start the EDA worker.
	worker := eda.NewWorker(pool, tr, transportMode, encKey, pollInterval, log, webBaseURL)

	// Set up HTTP server for Ponton inbound handler (and health check).
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"eda-worker"}`))
	})

	r.Post("/eda/poll-now", func(w http.ResponseWriter, r *http.Request) {
		go worker.PollOnce(context.Background())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"polling"}`))
	})

	// POST /eda/resend-confirmation
	// Body: {"process_ids": ["<uuid>", ...]}
	// Re-sends Anmeldung confirmation emails for the given EDA process IDs.
	r.Post("/eda/resend-confirmation", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ProcessIDs []uuid.UUID `json:"process_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.ProcessIDs) == 0 {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		errs := worker.ResendConfirmationEmails(r.Context(), req.ProcessIDs)
		w.Header().Set("Content-Type", "application/json")
		if len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			resp, _ := json.Marshal(map[string]any{"sent": len(req.ProcessIDs) - len(errs), "errors": msgs})
			w.WriteHeader(http.StatusMultiStatus)
			w.Write(resp)
			return
		}
		resp, _ := json.Marshal(map[string]any{"sent": len(req.ProcessIDs)})
		w.Write(resp)
	})

	if pontonTr != nil {
		r.Post("/eda/inbound", pontonTr.InboundHandler())
		log.Info("Ponton inbound HTTP handler registered at /eda/inbound")
	}

	httpPort := getEnv("PORT", "8081")
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: r,
	}

	go func() {
		log.Info("EDA worker HTTP server starting", "port", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	// Run the worker (blocks until ctx is cancelled).
	if err := worker.Run(ctx); err != nil && err != context.Canceled {
		log.Error("worker exited with error", "error", err)
	}

	// Graceful HTTP shutdown.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}

	log.Info("EDA worker stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustDecodeEncKey(log *slog.Logger, b64str string) []byte {
	if b64str == "" {
		log.Error("CREDENTIAL_ENCRYPTION_KEY is not set — refusing to start")
		os.Exit(1)
	}
	key, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil || len(key) != 32 {
		log.Error("CREDENTIAL_ENCRYPTION_KEY must be base64-encoded exactly 32 bytes (AES-256)")
		os.Exit(1)
	}
	return key
}

// noopTransport is a no-op Transport used when credentials are not configured.
type noopTransport struct {
	log *slog.Logger
}

func (n *noopTransport) Send(ctx context.Context, msg *types.Message) error {
	n.log.Warn("noopTransport.Send called, transport not configured", "process", msg.Process)
	return nil
}

func (n *noopTransport) Poll(ctx context.Context) ([]*types.Message, error) {
	n.log.Debug("noopTransport.Poll called, transport not configured")
	return nil, nil
}

func (n *noopTransport) SendAck(_ context.Context, originalMsgID, _, _ string) error {
	n.log.Debug("noopTransport.SendAck called, transport not configured", "ref", originalMsgID)
	return nil
}
