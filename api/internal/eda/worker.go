// Package eda implements the EDA worker for Austrian Marktkommunikation (MaKo)
// XML message exchange with edanet.at.
package eda

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/eda/transport"
	"github.com/lutzerb/eegabrechnung/internal/eda/types"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// viennaLoc is the Austrian timezone used for EDA date comparisons.
var viennaLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		loc = time.FixedZone("CET", 3600)
	}
	return loc
}()

// Worker polls inbound messages and processes outbound EDA jobs.
type Worker struct {
	db               *pgxpool.Pool
	transport        types.Transport // used for FILE and PONTON modes
	transportMode    string
	encKey           []byte // AES-256 key for decrypting per-EEG credentials
	pollInterval     time.Duration
	log              *slog.Logger
	webBaseURL       string // used to build portal link in member emails
	meterPointRepo   *repository.MeterPointRepository
	memberRepo       *repository.MemberRepository
	readingRepo      *repository.ReadingRepository
	eegRepo          *repository.EEGRepository
	edaMsgRepo       *repository.EDAMessageRepository
	edaProcRepo      *repository.EDAProcessRepository
	edaErrorRepo     *repository.EDAErrorRepository
	workerStatusRepo *repository.EDAWorkerStatusRepository
	onboardingRepo   *repository.OnboardingRepository
}

// NewWorker creates a Worker. pollInterval controls how often the transport
// is checked for new inbound messages. transportMode is stored for status reporting.
// For MAIL mode, tr may be nil — per-EEG MailTransports are created dynamically from DB credentials.
// encKey is the AES-256 key used to decrypt per-EEG credentials from the DB.
func NewWorker(db *pgxpool.Pool, tr types.Transport, transportMode string, encKey []byte, pollInterval time.Duration, log *slog.Logger, webBaseURL string) *Worker {
	return &Worker{
		db:               db,
		transport:        tr,
		transportMode:    transportMode,
		encKey:           encKey,
		pollInterval:     pollInterval,
		log:              log,
		webBaseURL:       webBaseURL,
		meterPointRepo:   repository.NewMeterPointRepository(db),
		memberRepo:       repository.NewMemberRepository(db),
		readingRepo:      repository.NewReadingRepository(db),
		eegRepo:          repository.NewEEGRepository(db, encKey),
		edaMsgRepo:       repository.NewEDAMessageRepository(db),
		edaProcRepo:      repository.NewEDAProcessRepository(db),
		edaErrorRepo:     repository.NewEDAErrorRepository(db),
		workerStatusRepo: repository.NewEDAWorkerStatusRepository(db),
		onboardingRepo:   repository.NewOnboardingRepository(db),
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	w.log.Info("EDA worker starting", "poll_interval", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Run once immediately, then tick.
	w.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("EDA worker stopping")
			return ctx.Err()
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

// PollOnce triggers a single poll cycle immediately. Used by the manual poll endpoint.
func (w *Worker) PollOnce(ctx context.Context) {
	w.poll(ctx)
}

// poll performs one iteration: receive inbound messages and process outbound jobs.
func (w *Worker) poll(ctx context.Context) {
	var pollErr string

	// Run receiveInbound in a goroutine with a hard 120s deadline so a stuck IMAP
	// connection (where even c.Close() blocks) cannot freeze the worker loop.
	inboundCtx, inboundCancel := context.WithTimeout(ctx, 120*time.Second)
	defer inboundCancel()
	type inboundResult struct{ err error }
	ch := make(chan inboundResult, 1)
	go func() {
		ch <- inboundResult{w.receiveInbound(inboundCtx)}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			w.log.Error("inbound poll error", "error", r.err)
			pollErr = r.err.Error()
		}
	case <-inboundCtx.Done():
		w.log.Warn("inbound poll timed out after 120s — skipping to outbound")
		pollErr = "inbound poll timeout"
	}
	if err := w.processOutbound(ctx); err != nil {
		w.log.Error("outbound processing error", "error", err)
		if pollErr == "" {
			pollErr = err.Error()
		}
	}
	// Update worker status in DB (best-effort).
	now := time.Now().UTC()
	_ = w.workerStatusRepo.Upsert(ctx, &domain.EDAWorkerStatus{
		TransportMode: w.transportMode,
		LastPollAt:    &now,
		LastError:     pollErr,
	})
}

// receiveInbound fetches messages from the transport, stores them in the DB,
// deduplicates, sends ebMS 2.0 Acks, and routes by message type.
// For MAIL mode, it polls each EEG's individual IMAP mailbox.
func (w *Worker) receiveInbound(ctx context.Context) error {
	if w.transportMode == "MAIL" {
		return w.receiveInboundPerEEG(ctx)
	}
	msgs, err := w.transport.Poll(ctx)
	if err != nil {
		return fmt.Errorf("transport.Poll: %w", err)
	}
	return w.processInboundMessages(ctx, msgs)
}

// receiveInboundPerEEG polls each EEG's individual IMAP mailbox in MAIL mode.
func (w *Worker) receiveInboundPerEEG(ctx context.Context) error {
	eegs, err := w.eegRepo.ListEEGsWithIMAPCredentials(ctx)
	if err != nil {
		return fmt.Errorf("ListEEGsWithIMAPCredentials: %w", err)
	}
	if len(eegs) == 0 {
		w.log.Debug("no EEGs with IMAP credentials configured")
		return nil
	}
	for _, eeg := range eegs {
		cfg := transport.MailConfig{
			IMAPHost:     eeg.EDAIMAPHost,
			IMAPUser:     eeg.EDAIMAPUser,
			IMAPPassword: eeg.EDAIMAPPassword,
			SMTPHost:     eeg.EDASmtpHost,
			SMTPUser:     eeg.EDASmtpUser,
			SMTPPassword: eeg.EDASmtpPassword,
			SMTPFrom:     eeg.EDASmtpFrom,
		}
		mt, err := transport.NewMailTransport(cfg, w.log)
		if err != nil {
			w.log.Warn("skipping EEG — MAIL transport not configured", "eeg_id", eeg.ID, "eeg_name", eeg.Name, "error", err)
			continue
		}
		msgs, err := mt.Poll(ctx)
		if err != nil {
			w.log.Error("IMAP poll failed", "eeg_id", eeg.ID, "eeg_name", eeg.Name, "error", err)
			continue
		}
		if err := w.processInboundMessages(ctx, msgs); err != nil {
			w.log.Error("processInboundMessages failed", "eeg_id", eeg.ID, "eeg_name", eeg.Name, "error", err)
		}
	}
	return nil
}

// processInboundMessages handles a slice of inbound messages from any transport.
func (w *Worker) processInboundMessages(ctx context.Context, msgs []*types.Message) error {

	for _, msg := range msgs {
		// DUPL deduplication: skip if we already processed this MessageID.
		if msg.ID != "" {
			dup, dupErr := w.edaMsgRepo.ExistsByMessageID(ctx, msg.ID)
			if dupErr != nil {
				w.log.Warn("DUPL check failed", "message_id", msg.ID, "error", dupErr)
			} else if dup {
				w.log.Info("skipping duplicate inbound message", "message_id", msg.ID)
				continue
			}
		}

		msgID, err := w.storeInboundMessage(ctx, msg)
		if err != nil {
			w.log.Error("failed to store inbound message",
				"message_id", msg.ID,
				"error", err,
			)
			w.storeError(ctx, nil, msg, err)
			continue
		}

		switch {
		case msg.Process == ProcessCRMsg:
			eegID, err := w.processCRMsg(ctx, msg)
			if err != nil {
				w.log.Error("CR_MSG energy import failed",
					"message_id", msg.ID,
					"error", err,
				)
				w.storeError(ctx, nil, msg, err)
				continue
			}
			if eegID != uuid.Nil {
				if err := w.edaMsgRepo.UpdateEegID(ctx, msgID, eegID); err != nil {
					w.log.Warn("failed to set eeg_id on CR_MSG", "id", msgID, "error", err)
				}
			}
			if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
				w.log.Warn("failed to mark CR_MSG as processed",
					"id", msgID,
					"error", err,
				)
			}
		case edaxml.IsCPDocument(msg.XMLPayload):
			eegID, err := w.processCPDocument(ctx, msg)
			if err != nil {
				w.log.Error("CPDocument processing failed",
					"message_id", msg.ID,
					"error", err,
				)
				w.storeError(ctx, nil, msg, err)
			} else {
				if eegID != uuid.Nil {
					if err := w.edaMsgRepo.UpdateEegID(ctx, msgID, eegID); err != nil {
						w.log.Warn("failed to set eeg_id on CPDocument", "id", msgID, "error", err)
					}
				}
				if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
					w.log.Warn("failed to mark CPDocument as processed",
						"id", msgID,
						"error", err,
					)
				}
			}
		case edaxml.IsCMNotification(msg.XMLPayload):
			eegID, err := w.processCMNotification(ctx, msgID, msg)
			if err != nil {
				w.log.Error("CMNotification processing failed",
					"message_id", msg.ID,
					"error", err,
				)
				w.storeError(ctx, nil, msg, err)
			} else {
				if eegID != uuid.Nil {
					if err := w.edaMsgRepo.UpdateEegID(ctx, msgID, eegID); err != nil {
						w.log.Warn("failed to set eeg_id on CMNotification", "id", msgID, "error", err)
					}
				}
				if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
					w.log.Warn("failed to mark CMNotification as processed",
						"id", msgID,
						"error", err,
					)
				}
			}
		case edaxml.IsECMPList(msg.XMLPayload):
			eegID, err := w.processECMPList(ctx, msgID, msg)
			if err != nil {
				w.log.Error("ECMPList processing failed",
					"message_id", msg.ID,
					"error", err,
				)
				w.storeError(ctx, nil, msg, err)
			} else {
				if eegID != uuid.Nil {
					if err := w.edaMsgRepo.UpdateEegID(ctx, msgID, eegID); err != nil {
						w.log.Warn("failed to set eeg_id on ECMPList", "id", msgID, "error", err)
					}
				}
				if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
					w.log.Warn("failed to mark ECMPList as processed",
						"id", msgID,
						"error", err,
					)
				}
			}
		case msg.Process == "ANTWORT_PT":
			// CPNotification = edanet transport-level delivery confirmation.
			// ANTWORT_PT (code 70) = edanet received our ANFORDERUNG_PT and forwarded it.
			// ABLEHNUNG_PT (e.g. code 55) = edanet rejected our request — must mark process rejected.
			notif := edaxml.ParseCPNotification(msg.XMLPayload)
			w.log.Info("CPNotification received",
				"message_code", notif.MessageCode,
				"response_code", notif.ResponseCode,
				"conversation_id", notif.ConversationID,
			)
			// Reconstruct UUID from stripped ConversationId and look up the process.
			var notifEegID uuid.UUID
			var notifProc *domain.EDAProcess
			if notif.ConversationID != "" {
				convID := notif.ConversationID
				if len(convID) == 32 {
					convID = convID[:8] + "-" + convID[8:12] + "-" + convID[12:16] + "-" + convID[16:20] + "-" + convID[20:]
				}
				if proc, err := w.edaProcRepo.GetByConversationID(ctx, convID); err == nil {
					notifEegID = proc.EegID
					notifProc = proc
				}
			}
			if notifEegID != uuid.Nil {
				if err := w.edaMsgRepo.UpdateEegID(ctx, msgID, notifEegID); err != nil {
					w.log.Warn("failed to set eeg_id on CPNotification", "id", msgID, "error", err)
				}
			}
			// ABLEHNUNG_PT = edanet rejected our outbound message (e.g. ResponseCode 55 = unknown Zählpunkt).
			// Mark the EDA process as rejected so it doesn't stay stuck in "sent".
			if strings.HasPrefix(notif.MessageCode, "ABLEHNUNG") && notifProc != nil {
				errMsg := fmt.Sprintf("ABLEHNUNG_PT response_code=%s", notif.ResponseCode)
				now := time.Now().UTC()
				if err := w.edaProcRepo.UpdateStatus(ctx, notifProc.ID, "rejected", &now, errMsg); err != nil {
					w.log.Warn("ABLEHNUNG_PT: failed to mark process rejected",
						"process_id", notifProc.ID,
						"error", err,
					)
				} else {
					w.log.Warn("ABLEHNUNG_PT: process rejected by edanet",
						"process_id", notifProc.ID,
						"process_type", notifProc.ProcessType,
						"response_code", notif.ResponseCode,
					)
					// Send error notification email to the EEG operator (non-blocking).
					notifProc.Status = "rejected"
					notifProc.ErrorMsg = errMsg
					if eeg, eegErr := w.eegRepo.GetByIDInternal(ctx, notifProc.EegID); eegErr == nil {
						go w.sendEDAErrorNotification(ctx, notifProc, eeg)
					}
				}
			}
			if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
				w.log.Warn("failed to mark CPNotification as processed", "id", msgID, "error", err)
			}
		case edaxml.IsEDASendError(msg.XMLPayload):
			w.handleEDASendError(ctx, msgID, msg)
			if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
				w.log.Warn("failed to mark EDASendError as processed", "id", msgID, "error", err)
			}
		case edaxml.IsCMRevoke(msg.XMLPayload):
			eegID, err := w.processCMRevoke(ctx, msg)
			if err != nil {
				w.log.Error("CMRevoke processing failed",
					"message_id", msg.ID,
					"error", err,
				)
				w.storeError(ctx, nil, msg, err)
			} else {
				if eegID != uuid.Nil {
					if err := w.edaMsgRepo.UpdateEegID(ctx, msgID, eegID); err != nil {
						w.log.Warn("failed to set eeg_id on CMRevoke", "id", msgID, "error", err)
					}
				}
				if err := w.edaMsgRepo.MarkProcessed(ctx, msgID); err != nil {
					w.log.Warn("failed to mark CMRevoke as processed", "id", msgID, "error", err)
				}
			}
		default:
			w.log.Warn("unknown inbound message type, stored in error log",
				"process", msg.Process,
				"message_id", msg.ID,
			)
			w.storeError(ctx, nil, msg, fmt.Errorf("unhandled message type: %q", msg.Process))
		}
	}
	return nil
}

// storeError persists a failed inbound message to eda_errors for operator review.
func (w *Worker) storeError(ctx context.Context, eegID *uuid.UUID, msg *types.Message, cause error) {
	e := &domain.EDAError{
		Direction:   types.DirectionInbound,
		MessageType: msg.Process,
		Subject:     msg.Subject,
		RawContent:  msg.XMLPayload,
		ErrorMsg:    cause.Error(),
	}
	if eegID != nil {
		e.EegID = eegID
	}
	if err := w.edaErrorRepo.Create(ctx, e); err != nil {
		w.log.Warn("failed to persist EDA error record", "error", err)
	}
}

// handleEDASendError processes an EDASendError response from the edanet gateway.
// It looks up the original outbound message by subject and marks the corresponding
// EDA process as failed with the gateway's reason text.
func (w *Worker) handleEDASendError(ctx context.Context, dbMsgID uuid.UUID, msg *types.Message) {
	edaErr, err := edaxml.ParseEDASendError(msg.XMLPayload)
	if err != nil {
		w.log.Warn("failed to parse EDASendError", "error", err)
		return
	}

	w.log.Warn("EDASendError received from edanet gateway",
		"mail_subject", edaErr.MailSubject,
		"reason", edaErr.ReasonText,
	)

	// Find the outbound message that triggered this error by matching the email subject.
	row := w.db.QueryRow(ctx,
		`SELECT id, eda_process_id, eeg_id FROM eda_messages
		 WHERE subject = $1 AND direction = 'outbound'
		 ORDER BY created_at DESC
		 LIMIT 1`,
		edaErr.MailSubject,
	)
	var outboundMsgID uuid.UUID
	var processIDPtr, eegIDPtr *uuid.UUID
	if err := row.Scan(&outboundMsgID, &processIDPtr, &eegIDPtr); err != nil {
		w.log.Warn("EDASendError: could not find matching outbound message",
			"subject", edaErr.MailSubject,
			"error", err,
		)
		// Set msg.Subject so the subject is preserved in the dead-letter entry.
		msg.Subject = edaErr.MailSubject
		w.storeError(ctx, nil, msg, fmt.Errorf("EDASendError (no matching process found): %s", edaErr.ReasonText))
		return
	}

	// Mark the outbound message itself as error so it's visible in the Nachrichten tab.
	if outboundMsgID != uuid.Nil {
		if err := w.edaMsgRepo.MarkError(ctx, outboundMsgID, edaErr.ReasonText); err != nil {
			w.log.Warn("EDASendError: failed to mark outbound message as error",
				"msg_id", outboundMsgID,
				"error", err,
			)
		}
	}

	// Mark the EDA process as error.
	if processIDPtr != nil && *processIDPtr != uuid.Nil {
		if err := w.edaProcRepo.UpdateStatus(ctx, *processIDPtr, "error", nil, edaErr.ReasonText); err != nil {
			w.log.Warn("EDASendError: failed to mark process as error",
				"process_id", *processIDPtr,
				"error", err,
			)
		} else {
			w.log.Info("EDASendError: process marked as error",
				"process_id", *processIDPtr,
				"reason", edaErr.ReasonText,
			)
			// Send error notification email to the EEG operator (non-blocking).
			if proc, lookupErr := w.edaProcRepo.GetByID(ctx, *processIDPtr); lookupErr == nil {
				if eeg, eegErr := w.eegRepo.GetByIDInternal(ctx, proc.EegID); eegErr == nil {
					go w.sendEDAErrorNotification(ctx, proc, eeg)
				}
			}
		}
	}

	// Update the eda_message record (the EDASendError itself) with the eeg_id if available.
	if eegIDPtr != nil && *eegIDPtr != uuid.Nil {
		if err := w.edaMsgRepo.UpdateEegID(ctx, dbMsgID, *eegIDPtr); err != nil {
			w.log.Warn("EDASendError: failed to set eeg_id", "error", err)
		}
	}

	// Set msg.Subject so it's preserved in the dead-letter entry (the EDASendError itself
	// doesn't have a meaningful subject — store the referenced outbound subject instead).
	msg.Subject = edaErr.MailSubject

	// Always store in eda_errors so the operator sees it in the Fehler tab.
	w.storeError(ctx, eegIDPtr, msg, fmt.Errorf("EDASendError: %s", edaErr.ReasonText))
}

// storeInboundMessage persists an inbound EDA message and returns its DB id.
func (w *Worker) storeInboundMessage(ctx context.Context, msg *types.Message) (uuid.UUID, error) {
	q := `INSERT INTO eda_messages
	        (message_id, process, message_type, subject, body, direction, status, xml_payload, error_msg, from_address, to_address)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	      RETURNING id`
	var id uuid.UUID
	err := w.db.QueryRow(ctx, q,
		msg.ID,
		msg.Process,
		msg.Process,
		msg.Subject,
		msg.EmailBody,
		msg.Direction,
		StatusPending,
		msg.XMLPayload,
		"",
		msg.From,
		msg.To,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert eda_message: %w", err)
	}
	return id, nil
}

// processCRMsg parses a ConsumptionRecord XML message and stores the energy
// readings in the energy_readings table. Returns the EEG ID if determinable.
func (w *Worker) processCRMsg(ctx context.Context, msg *types.Message) (uuid.UUID, error) {
	record, err := edaxml.ParseCRMsg(msg.XMLPayload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse CR_MSG XML: %w", err)
	}

	// Skip simulation documents — do not import into production data.
	if record.DocumentMode == "SIMU" {
		w.log.Info("skipping SIMU CR_MSG document", "zaehlpunkt", record.Zaehlpunkt)
		return uuid.Nil, nil
	}

	if record.Zaehlpunkt == "" {
		return uuid.Nil, fmt.Errorf("CR_MSG has no MeteringPoint (Zählpunkt)")
	}

	// Enforce transition date: skip blocks that predate it.
	// The DB stores eda_transition_date as a bare date (e.g. 2026-04-07), which pgx
	// loads as midnight UTC. EDA period starts are in Vienna local time (UTC+1/+2).
	// Re-interpret the date as Vienna midnight so that e.g. April 7 00:00 Vienna
	// (= April 6 22:00 UTC) is not incorrectly filtered when transition = April 7.
	if record.GemeinschaftID != "" {
		eeg, lookupErr := w.eegRepo.GetByGemeinschaftID(ctx, record.GemeinschaftID)
		if lookupErr == nil && eeg.EdaTransitionDate != nil {
			td := *eeg.EdaTransitionDate
			transitionVienna := time.Date(td.Year(), td.Month(), td.Day(), 0, 0, 0, 0, viennaLoc)
			filtered := record.Energies[:0]
			for _, energy := range record.Energies {
				if energy.PeriodStart.Before(transitionVienna) {
					w.log.Warn("CR_MSG block predates EDA transition date — skipping",
						"zaehlpunkt", record.Zaehlpunkt,
						"period_start", energy.PeriodStart,
						"transition_date_vienna", transitionVienna,
					)
					continue
				}
				filtered = append(filtered, energy)
			}
			record.Energies = filtered
		}
	}

	// Look up the meter point by Zählpunkt identifier.
	mp, err := w.meterPointRepo.GetByZaehlpunkt(ctx, record.Zaehlpunkt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("meter point %q not found: %w", record.Zaehlpunkt, err)
	}

	readings := buildReadingsFromCRMsg(mp.ID, record)
	if len(readings) == 0 {
		w.log.Info("CR_MSG produced no energy readings", "zaehlpunkt", record.Zaehlpunkt)
	} else {
		n, err := w.readingRepo.BulkUpsert(ctx, readings)
		if err != nil {
			return uuid.Nil, fmt.Errorf("store energy readings: %w", err)
		}
		w.log.Info("CR_MSG imported",
			"zaehlpunkt", record.Zaehlpunkt,
			"readings", n,
		)
	}

	// Always mark the corresponding EC_REQ_PT process as completed when a DATEN_CRMSG
	// arrives — even if all readings were filtered by the transition date.
	// Edanet does not echo our ConversationID in the response, so we match by Zählpunkt.
	if proc, err := w.edaProcRepo.FindSentReqPTByZaehlpunkt(ctx, record.Zaehlpunkt); err == nil {
		now := time.Now().UTC()
		if upErr := w.edaProcRepo.UpdateStatus(ctx, proc.ID, "completed", &now, ""); upErr != nil {
			w.log.Warn("CR_MSG: failed to mark EC_REQ_PT process completed",
				"zaehlpunkt", record.Zaehlpunkt,
				"error", upErr,
			)
		} else {
			w.log.Info("CR_MSG: EC_REQ_PT process marked completed",
				"zaehlpunkt", record.Zaehlpunkt,
				"process_id", proc.ID,
			)
		}
	}

	return mp.EegID, nil
}

// buildReadingsFromCRMsg maps CR_MSG OBIS blocks to domain.EnergyReading values.
//
// Field convention (matches XLSX import and report SQL):
//
//	CONSUMPTION meter  →  wh_self      = EEG share consumed ("Bezug EEG" / "Ausgetauscht")
//	                       wh_community = allocated EEG energy (G.02, informational)
//	                       wh_total     = Gesamtbezug
//
//	GENERATION meter   →  wh_community = EEG share fed in ("Einspeisung EEG")
//	                       wh_self      = Resteinspeisung ins öffentliche Netz
//	                       wh_total     = Gesamterzeugung
//
// OBIS mapping per schema 01.41 (ebutilities.at, December 2023):
//
//	Consumption meter (Bezugs-ZP, 1.9.0):
//	  G.01 / G.01T → wh_total     Gesamtbezug × Teilnahmefaktor
//	  G.02         → wh_community  Zuteilung (kann > Bezug sein — informational only)
//	  G.03 / G.03R → wh_self      Eigendeckung = tatsächlich bezogener EEG-Anteil
//
//	Generation meter (Einspeise-ZP, 2.9.0):
//	  G.01 / G.01T → wh_total     Gesamterzeugung × Teilnahmefaktor
//	  P.01T        → residual     Restnetzüberschuss (Resteinspeisung ins öffentliche Netz)
//	                 → wh_community = wh_total − P.01T  (Einspeisung in EEG)
//	                 → wh_self     = P.01T              (Resteinspeisung)
//
//	Old schema (≤01.30):
//	  Consumption 1.9.0 P.01 → wh_self      (EEG share, same convention as G.03)
//	  Generation  2.9.0 P.01 → wh_community (EEG share, same convention as new schema)
//
// Values are in kWh (UOM=KWH as per ConsumptionRecord schema).
func buildReadingsFromCRMsg(meterPointID uuid.UUID, record *edaxml.CRMsgRecord) []domain.EnergyReading {
	type accReading struct {
		domain.EnergyReading
		residual      float64 // P.01T scratch: Restnetzüberschuss (generation 01.41 only)
		hasResidual   bool
		hasOldGenComm bool // true when old-schema 2.9.0 P.01 set wh_community (generation meter)
	}

	byTS := map[time.Time]*accReading{}

	for _, block := range record.Energies {
		for _, ed := range block.Data {
			mc := ed.MeterCode
			isGenerationDir := strings.Contains(mc, "2.9.0")

			// Skip unrecognized OBIS codes (neither 1.9.0 nor 2.9.0).
			if !strings.Contains(mc, "1.9.0") && !isGenerationDir {
				continue
			}

			// P.01T (with T suffix): Restnetzüberschuss — generation 01.41 only.
			isResidualExport := strings.Contains(mc, "P.01T")
			// P.01 (no T): old schema EEG community share code.
			isOldCommunity := strings.Contains(mc, "P.01") && !strings.Contains(mc, "P.01T")
			// G.03 / G.03R: Eigendeckung — tatsächlich bezogener EEG-Anteil (consumption only).
			// Note: G.03/G.03R always carry the 2.9.0 OBIS prefix even on consumption meters —
			// do NOT gate on isConsumptionDir here.
			isSelfCoverage := strings.Contains(mc, "G.03")
			// G.02: allocated EEG energy (consumption only, can exceed actual consumption).
			// Same note: 2.9.0 prefix on consumption meter — do NOT gate on isConsumptionDir.
			isAllocation := strings.Contains(mc, "G.02")

			for _, pos := range ed.Positions {
				ts := pos.From
				if _, ok := byTS[ts]; !ok {
					byTS[ts] = &accReading{
						EnergyReading: domain.EnergyReading{
							MeterPointID: meterPointID,
							Ts:           ts,
							Source:       "eda",
						},
					}
				}
				r := byTS[ts]

				switch {
				case isResidualExport:
					// Generation 01.41: P.01T = Restnetzüberschuss.
					// wh_community = wh_total − P.01T, computed after all codes are processed.
					r.residual = pos.Value
					r.hasResidual = true

				case isAllocation:
					// G.02: EEG allocation (informational, can exceed G.03) → wh_community.
					// Inherently a consumption-meter code; 2.9.0 OBIS prefix is expected here.
					r.WhCommunity = pos.Value

				case isSelfCoverage:
					// G.03 / G.03R: Eigendeckung = tatsächlicher EEG-Anteil → wh_self.
					// Inherently a consumption-meter code; 2.9.0 OBIS prefix is expected here.
					r.WhSelf = pos.Value

				case isOldCommunity && isGenerationDir:
					// Old schema 2.9.0 P.01: EEG share fed in → wh_community.
					r.WhCommunity = pos.Value
					r.hasOldGenComm = true

				case isOldCommunity:
					// Old schema 1.9.0 P.01: EEG share consumed → wh_self.
					r.WhSelf = pos.Value

				default:
					// G.01 / G.01T (both directions): Gesamtbezug or Gesamterzeugung → wh_total.
					r.WhTotal = pos.Value
				}
			}
		}
	}

	result := make([]domain.EnergyReading, 0, len(byTS))
	for _, acc := range byTS {
		r := &acc.EnergyReading
		if acc.hasResidual {
			// Generation 01.41: derive Einspeisung in EEG and Resteinspeisung.
			if r.WhTotal > acc.residual {
				r.WhCommunity = r.WhTotal - acc.residual
			}
			r.WhSelf = acc.residual
		} else if acc.hasOldGenComm && r.WhTotal > 0 && r.WhCommunity > 0 && r.WhTotal > r.WhCommunity {
			// Old schema generation: P.01 → wh_community already set; derive Resteinspeisung.
			r.WhSelf = r.WhTotal - r.WhCommunity
		}
		// Consumption: wh_self (G.03) and wh_community (G.02) already set in the loop above.
		result = append(result, *r)
	}
	return result
}

// processCPDocument handles an incoming CPDocument confirmation from the Netzbetreiber,
// matching it to the open EDA process via ConversationID and updating its status.
// Returns the EEG ID of the matched process so the caller can tag the stored message.
func (w *Worker) processCPDocument(ctx context.Context, msg *types.Message) (uuid.UUID, error) {
	result, err := edaxml.ParseCPDocument(msg.XMLPayload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse CPDocument: %w", err)
	}

	if result.ConversationID == "" {
		w.log.Warn("CPDocument has no ConversationID — cannot match process",
			"message_code", result.MessageCode,
		)
		return uuid.Nil, nil
	}

	proc, err := w.edaProcRepo.GetByConversationID(ctx, result.ConversationID)
	if err != nil {
		// No matching process — log and ignore (might be from a previous deployment).
		w.log.Warn("CPDocument: no matching EDA process",
			"conversation_id", result.ConversationID,
			"message_code", result.MessageCode,
		)
		return uuid.Nil, nil
	}

	// Map EDA MessageCode to internal process status.
	var newStatus string
	var errMsg string
	switch result.MessageCode {
	case "ERSTE_ANM":
		newStatus = "first_confirmed"
	case "FINALE_ANM", "ABSCHLUSS_ECON":
		newStatus = "confirmed"
	default:
		if strings.HasPrefix(result.MessageCode, "ABGELEHNT") ||
			strings.HasPrefix(result.MessageCode, "ABLEHNUNG") {
			newStatus = "rejected"
			errMsg = result.MessageCode
		} else {
			// Unknown code — record it as a note but don't change status.
			w.log.Info("CPDocument: unhandled MessageCode",
				"message_code", result.MessageCode,
				"conversation_id", result.ConversationID,
			)
			return proc.EegID, nil
		}
	}

	now := time.Now().UTC()
	var completedAt *time.Time
	if newStatus == "confirmed" || newStatus == "rejected" {
		completedAt = &now
	}

	if err := w.edaProcRepo.UpdateStatus(ctx, proc.ID, newStatus, completedAt, errMsg); err != nil {
		return uuid.Nil, fmt.Errorf("update EDA process status: %w", err)
	}

	w.log.Info("CPDocument: EDA process updated",
		"process_id", proc.ID,
		"conversation_id", result.ConversationID,
		"message_code", result.MessageCode,
		"new_status", newStatus,
	)

	// Send error notification email to the EEG operator when process is rejected (non-blocking).
	if newStatus == "rejected" {
		proc.Status = newStatus
		proc.ErrorMsg = errMsg
		if eeg, eegErr := w.eegRepo.GetByIDInternal(ctx, proc.EegID); eegErr == nil {
			go w.sendEDAErrorNotification(ctx, proc, eeg)
		}
	}

	return proc.EegID, nil
}

// processCMNotification handles an incoming CMNotification from the Netzbetreiber.
// These arrive during the EC_REQ_ONL flow:
//
//	ANTWORT_ECON     — intermediate acknowledgement (no status change)
//	ZUSTIMMUNG_ECON  — consent granted → first_confirmed
//	ABLEHNUNG_ECON   — consent denied  → rejected
//
// Returns the EEG ID of the matched process so the caller can tag the stored message.
func (w *Worker) processCMNotification(ctx context.Context, msgID uuid.UUID, msg *types.Message) (uuid.UUID, error) {
	result, err := edaxml.ParseCMNotification(msg.XMLPayload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse CMNotification: %w", err)
	}

	// Store the concrete inbound CMNotification code in eda_messages so the UI
	// can distinguish ANTWORT_ECON, ZUSTIMMUNG_ECON and ABLEHNUNG_ECON.
	if result.MessageCode != "" {
		if err := w.edaMsgRepo.UpdateClassification(ctx, msgID, result.MessageCode); err != nil {
			w.log.Warn("CMNotification: failed to update message classification",
				"id", msgID,
				"message_code", result.MessageCode,
				"error", err,
			)
		}
	}

	if result.ConversationID == "" {
		w.log.Warn("CMNotification has no ConversationID — cannot match process",
			"message_code", result.MessageCode,
		)
		return uuid.Nil, nil
	}

	// The XML ConversationId strips hyphens (e.g. "4c3240df6a5742b9...").
	// Re-add hyphens so it matches the UUID stored in the DB.
	convID := result.ConversationID
	if len(convID) == 32 {
		convID = convID[:8] + "-" + convID[8:12] + "-" + convID[12:16] + "-" + convID[16:20] + "-" + convID[20:]
	}

	proc, err := w.edaProcRepo.GetByConversationID(ctx, convID)
	if err != nil {
		w.log.Warn("CMNotification: no matching EDA process",
			"conversation_id", convID,
			"message_code", result.MessageCode,
		)
		return uuid.Nil, nil
	}

	var newStatus string
	var errMsg string
	switch result.MessageCode {
	case "ANTWORT_ECON":
		// Intermediate step — NB acknowledged the request, no status change yet.
		w.log.Info("CMNotification: ANTWORT_ECON received (intermediate ack)",
			"conversation_id", result.ConversationID,
			"metering_point", result.MeteringPoint,
		)
		return proc.EegID, nil
	case "ZUSTIMMUNG_ECON":
		newStatus = "first_confirmed"
		// Store the NB-assigned ConsentId on the meter point so it can be used for CM_REV_SP.
		if result.ConsentID != "" && proc.MeterPointID != nil && *proc.MeterPointID != uuid.Nil {
			if err := w.meterPointRepo.UpdateConsentID(ctx, *proc.MeterPointID, result.ConsentID); err != nil {
				w.log.Warn("failed to store consent_id after ZUSTIMMUNG_ECON",
					"meter_point_id", *proc.MeterPointID, "error", err)
			}
		}
		// Advance onboarding status so the 72h "Datenfreigabe ausstehend" reminder no longer fires.
		// ZUSTIMMUNG_ECON means consent was granted; ABSCHLUSS_ECON only arrives close to valid_from.
		if proc.MeterPointID != nil && *proc.MeterPointID != uuid.Nil {
			if err := w.onboardingRepo.SetActiveByMeterPoint(ctx, *proc.MeterPointID); err != nil {
				w.log.Warn("failed to advance onboarding status after ZUSTIMMUNG_ECON",
					"meter_point_id", *proc.MeterPointID, "error", err)
			}
		}
	case "ABLEHNUNG_ECON":
		newStatus = "rejected"
		errMsg = fmt.Sprintf("ABLEHNUNG_ECON response_code=%s", result.ResponseCode)
	case "AUFHEBUNG_CCMS_OK":
		// CM_REV_SP confirmed — consent revocation acknowledged by NB.
		newStatus = "completed"
	case "AUFHEBUNG_CCMS_ABGEL":
		// CM_REV_SP rejected by NB.
		newStatus = "rejected"
		errMsg = fmt.Sprintf("AUFHEBUNG_CCMS_ABGEL response_code=%s", result.ResponseCode)
	default:
		w.log.Info("CMNotification: unhandled MessageCode",
			"message_code", result.MessageCode,
			"conversation_id", result.ConversationID,
		)
		return proc.EegID, nil
	}

	now := time.Now().UTC()
	var completedAt *time.Time
	if newStatus == "rejected" || newStatus == "completed" {
		completedAt = &now
	}

	if err := w.edaProcRepo.UpdateStatus(ctx, proc.ID, newStatus, completedAt, errMsg); err != nil {
		return uuid.Nil, fmt.Errorf("update EDA process status: %w", err)
	}

	w.log.Info("CMNotification: EDA process updated",
		"process_id", proc.ID,
		"conversation_id", result.ConversationID,
		"message_code", result.MessageCode,
		"response_code", result.ResponseCode,
		"new_status", newStatus,
	)

	// Post-confirmation actions for CM_REV_SP (AUFHEBUNG_CCMS_OK).
	if newStatus == "completed" && proc.ProcessType == "CM_REV_SP" &&
		proc.MeterPointID != nil && *proc.MeterPointID != uuid.Nil &&
		proc.ValidFrom != nil {
		if err := w.meterPointRepo.UpdateAbgemeldetAm(ctx, *proc.MeterPointID, *proc.ValidFrom); err != nil {
			w.log.Warn("failed to update abgemeldet_am after CM_REV_SP confirmation",
				"meter_point_id", *proc.MeterPointID, "error", err)
		}
	}

	// Send error notification email to the EEG operator when process is rejected (non-blocking).
	if newStatus == "rejected" {
		proc.Status = newStatus
		proc.ErrorMsg = errMsg
		if eeg, eegErr := w.eegRepo.GetByIDInternal(ctx, proc.EegID); eegErr == nil {
			go w.sendEDAErrorNotification(ctx, proc, eeg)
		}
	}

	return proc.EegID, nil
}

// processECMPList handles an incoming ECMPList message from the Netzbetreiber.
// These arrive at the end of the EC_REQ_ONL flow:
//
//	SENDEN_ECP    — periodic full list of current EC meter points (informational)
//	ABSCHLUSS_ECON — final confirmation of a registration/deregistration → confirmed
//
// Returns the EEG ID of the matched process so the caller can tag the stored message.
func (w *Worker) processECMPList(ctx context.Context, msgID uuid.UUID, msg *types.Message) (uuid.UUID, error) {
	result, err := edaxml.ParseECMPList(msg.XMLPayload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse ECMPList: %w", err)
	}

	// Store the concrete MessageCode (SENDEN_ECP / ABSCHLUSS_ECON) so the UI
	// can show a human-readable label instead of the generic "ECMPList".
	if result.MessageCode != "" {
		if err := w.edaMsgRepo.UpdateClassification(ctx, msgID, result.MessageCode); err != nil {
			w.log.Warn("ECMPList: failed to update message classification",
				"id", msgID,
				"message_code", result.MessageCode,
				"error", err,
			)
		}
	}

	w.log.Info("ECMPList received",
		"message_code", result.MessageCode,
		"ecid", result.ECID,
		"entries", len(result.Entries),
		"conversation_id", result.ConversationID,
	)

	// SENDEN_ECP is the response to our EC_PODLIST request (ANFORDERUNG_ECP).
	// It can also arrive as an unsolicited periodic push — in both cases we try
	// to match an open EC_PODLIST process by ConversationID.
	if result.MessageCode == "SENDEN_ECP" {
		// Backfill consent_ids for meter points that don't have one yet.
		// SENDEN_ECP entries contain the NB-assigned ConsentId for every active meter point.
		consentMap := make(map[string]string, len(result.Entries))
		for _, e := range result.Entries {
			if e.ConsentID != "" {
				consentMap[e.MeteringPoint] = e.ConsentID
			}
		}
		if len(consentMap) > 0 {
			n, backfillErr := w.meterPointRepo.BackfillConsentIDs(ctx, consentMap)
			if backfillErr != nil {
				w.log.Warn("SENDEN_ECP: consent_id backfill failed", "error", backfillErr)
			} else if n > 0 {
				w.log.Info("SENDEN_ECP: backfilled consent_ids", "updated", n)
			}
		}

		// Re-add hyphens if the ConversationID arrived without them (32 hex chars).
		convID := result.ConversationID
		if len(convID) == 32 {
			convID = convID[0:8] + "-" + convID[8:12] + "-" + convID[12:16] + "-" + convID[16:20] + "-" + convID[20:]
		}
		if convID != "" {
			proc, lookupErr := w.edaProcRepo.GetByConversationID(ctx, convID)
			if lookupErr == nil && proc.ProcessType == "EC_PODLIST" {
				newStatus := "completed"
				if err := w.edaProcRepo.UpdateStatus(ctx, proc.ID, newStatus, nil, ""); err != nil {
					w.log.Warn("failed to complete EC_PODLIST process",
						"process_id", proc.ID, "error", err,
					)
				} else {
					w.log.Info("EC_PODLIST completed via SENDEN_ECP",
						"process_id", proc.ID,
						"entries", len(result.Entries),
					)
				}
				return proc.EegID, nil
			}
		}
		// Unsolicited SENDEN_ECP — tag message by ECID if possible.
		if result.ECID != "" {
			eeg, lookupErr := w.eegRepo.GetByGemeinschaftID(ctx, result.ECID)
			if lookupErr == nil {
				return eeg.ID, nil
			}
		}
		return uuid.Nil, nil
	}

	// ABSCHLUSS_ECON — final confirmation: match open process via ConversationID.
	if result.ConversationID == "" {
		w.log.Warn("ECMPList ABSCHLUSS_ECON has no ConversationID — cannot match process",
			"message_code", result.MessageCode,
		)
		return uuid.Nil, nil
	}

	// Re-add hyphens if the ConversationID arrived without them (32 hex chars).
	// Outbound CMRequest/ECMPList strip hyphens from the UUID; the Netzbetreiber
	// echoes the same format back in ABSCHLUSS_ECON, so we normalise before lookup.
	abschlussConvID := result.ConversationID
	if len(abschlussConvID) == 32 {
		abschlussConvID = abschlussConvID[:8] + "-" + abschlussConvID[8:12] + "-" + abschlussConvID[12:16] + "-" + abschlussConvID[16:20] + "-" + abschlussConvID[20:]
	}

	proc, err := w.edaProcRepo.GetByConversationID(ctx, abschlussConvID)
	if err != nil {
		w.log.Warn("ECMPList: no matching EDA process",
			"conversation_id", abschlussConvID,
			"message_code", result.MessageCode,
		)
		return uuid.Nil, nil
	}

	now := time.Now().UTC()
	if err := w.edaProcRepo.UpdateStatus(ctx, proc.ID, "confirmed", &now, ""); err != nil {
		return uuid.Nil, fmt.Errorf("update EDA process status: %w", err)
	}

	// Post-confirmation actions for EC_REQ_ONL
	if proc.ProcessType == "EC_REQ_ONL" && proc.MeterPointID != nil && *proc.MeterPointID != uuid.Nil {
		mpID := *proc.MeterPointID

		// Update registriert_seit and consent_id from the ECMPList ABSCHLUSS_ECON entry.
		var confirmedDate time.Time
		for _, entry := range result.Entries {
			if entry.MeteringPoint == proc.Zaehlpunkt && !entry.DateFrom.IsZero() {
				confirmedDate = entry.DateFrom
				if err := w.meterPointRepo.UpdateRegistriertSeit(ctx, mpID, entry.DateFrom); err != nil {
					w.log.Warn("failed to update registriert_seit after EDA confirmation",
						"meter_point_id", mpID, "error", err)
				}
				// Also store consent_id if present (backup for ZUSTIMMUNG_ECON path).
				if entry.ConsentID != "" {
					if err := w.meterPointRepo.UpdateConsentID(ctx, mpID, entry.ConsentID); err != nil {
						w.log.Warn("failed to store consent_id from ECMPList",
							"meter_point_id", mpID, "error", err)
					}
				}
				break
			}
		}

		// Auto-activate onboarding request.
		if err := w.onboardingRepo.SetActiveByMeterPoint(ctx, mpID); err != nil {
			w.log.Warn("failed to auto-activate onboarding after EDA confirmation",
				"meter_point_id", mpID, "error", err)
		}

		// Transition member REGISTERED → ACTIVE.
		if err := w.memberRepo.ActivateByMeterPoint(ctx, mpID); err != nil {
			w.log.Warn("failed to activate member after EDA confirmation",
				"meter_point_id", mpID, "error", err)
		}

		// Send confirmation email to member.
		go w.sendAnmeldungConfirmationEmail(ctx, mpID, proc.EegID, proc.Zaehlpunkt, confirmedDate)
	}

	w.log.Info("ECMPList: EDA process confirmed",
		"process_id", proc.ID,
		"conversation_id", result.ConversationID,
		"entries", len(result.Entries),
	)
	return proc.EegID, nil
}

// maxJobRetries is the maximum number of times a failed EDA send is retried
// before the job is permanently marked as 'error'.
const maxJobRetries = 3

// edaJob holds the data from the jobs table relevant for EDA processing.
type edaJob struct {
	id         uuid.UUID
	jobType    string
	payload    edaJobPayload
	retryCount int
}

type edaJobPayload struct {
	Process        string    `json:"process"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	GemeinschaftID string    `json:"gemeinschaft_id"`
	ConversationID string    `json:"conversation_id"`
	XMLPayload     string    `json:"xml_payload"`
	EDAProcessID   uuid.UUID `json:"eda_process_id"`
	EegID          uuid.UUID `json:"eeg_id"`
}

// processOutbound picks up pending EDA jobs from the jobs table and sends them.
func (w *Worker) processOutbound(ctx context.Context) error {
	tx, err := w.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock up to 10 pending EDA jobs at once.
	rows, err := tx.Query(ctx, `
		SELECT id, type, payload, retry_count
		FROM jobs
		WHERE status = 'pending' AND type LIKE 'eda.%'
		ORDER BY created_at
		LIMIT 10
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []edaJob
	for rows.Next() {
		var j edaJob
		var payloadRaw []byte
		if err := rows.Scan(&j.id, &j.jobType, &payloadRaw, &j.retryCount); err != nil {
			return fmt.Errorf("scan job: %w", err)
		}
		if err := json.Unmarshal(payloadRaw, &j.payload); err != nil {
			w.log.Warn("failed to decode job payload",
				"job_id", j.id,
				"error", err,
			)
			continue
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}
	rows.Close()

	for _, job := range jobs {
		sentMsg, sendErr := w.sendJob(ctx, job)

		if sendErr == nil {
			// Send succeeded.
			w.log.Info("EDA job sent",
				"job_id", job.id,
				"process", job.payload.Process,
			)

			if _, err := tx.Exec(ctx,
				`UPDATE jobs SET status = 'sent', updated_at = now() WHERE id = $1`,
				job.id,
			); err != nil {
				w.log.Error("failed to update job status to sent", "job_id", job.id, "error", err)
			}

			// Advance eda_process from 'pending' to 'sent' (don't overwrite a later status
			// like 'first_confirmed' that may have arrived via an inbound CPDocument in the same poll).
			if job.payload.EDAProcessID != uuid.Nil {
				if _, err := tx.Exec(ctx,
					`UPDATE eda_processes SET status = 'sent', updated_at = now()
					 WHERE id = $1 AND status = 'pending'`,
					job.payload.EDAProcessID,
				); err != nil {
					w.log.Warn("failed to mark EDA process as sent",
						"process_id", job.payload.EDAProcessID,
						"error", err,
					)
				}
			}

			// Record sent message in eda_messages using the actual email subject.
			var eegIDArg *uuid.UUID
			if job.payload.EegID != uuid.Nil {
				id := job.payload.EegID
				eegIDArg = &id
			}
			var edaProcessIDArg *uuid.UUID
			if job.payload.EDAProcessID != uuid.Nil {
				id := job.payload.EDAProcessID
				edaProcessIDArg = &id
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO eda_messages
				   (eeg_id, process, message_type, subject, body, direction, status, xml_payload, error_msg, from_address, to_address, eda_process_id)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				eegIDArg,
				job.payload.Process,
				job.payload.Process,
				sentMsg.Subject,
				"",
				DirectionOutbound,
				StatusSent,
				job.payload.XMLPayload,
				"",
				job.payload.From,
				job.payload.To,
				edaProcessIDArg,
			); err != nil {
				w.log.Warn("failed to record outbound eda_message", "job_id", job.id, "error", err)
			}

			continue
		}

		// Send failed — apply retry logic.
		newRetryCount := job.retryCount + 1
		w.log.Error("failed to send EDA job",
			"job_id", job.id,
			"process", job.payload.Process,
			"attempt", newRetryCount,
			"max_retries", maxJobRetries,
			"error", sendErr,
		)

		if newRetryCount < maxJobRetries {
			// Keep job pending for the next poll cycle.
			if _, err := tx.Exec(ctx,
				`UPDATE jobs SET retry_count = $1, updated_at = now() WHERE id = $2`,
				newRetryCount, job.id,
			); err != nil {
				w.log.Error("failed to update job retry_count", "job_id", job.id, "error", err)
			}
			w.log.Warn("EDA job will be retried",
				"job_id", job.id,
				"process", job.payload.Process,
				"next_attempt", newRetryCount+1,
			)
			continue
		}

		// Retries exhausted — permanently fail the job.
		w.log.Error("EDA job permanently failed after max retries",
			"job_id", job.id,
			"process", job.payload.Process,
		)

		if _, err := tx.Exec(ctx,
			`UPDATE jobs SET status = 'error', retry_count = $1, updated_at = now() WHERE id = $2`,
			newRetryCount, job.id,
		); err != nil {
			w.log.Error("failed to update job status to error", "job_id", job.id, "error", err)
		}

		// Mark eda_process as error.
		if job.payload.EDAProcessID != uuid.Nil {
			if err := w.edaProcRepo.UpdateStatus(ctx, job.payload.EDAProcessID, "error", nil, sendErr.Error()); err != nil {
				w.log.Warn("failed to mark EDA process as error",
					"process_id", job.payload.EDAProcessID,
					"error", err,
				)
			} else {
				// Send error notification email to the EEG operator (non-blocking).
				if proc, lookupErr := w.edaProcRepo.GetByID(ctx, job.payload.EDAProcessID); lookupErr == nil {
					if eeg, eegErr := w.eegRepo.GetByIDInternal(ctx, proc.EegID); eegErr == nil {
						go w.sendEDAErrorNotification(ctx, proc, eeg)
					}
				}
			}
		}

		// Record permanent failure in eda_messages.
		// No real email subject exists since the send failed.
		var eegIDArg *uuid.UUID
		if job.payload.EegID != uuid.Nil {
			id := job.payload.EegID
			eegIDArg = &id
		}
		var edaProcessIDArgFail *uuid.UUID
		if job.payload.EDAProcessID != uuid.Nil {
			id := job.payload.EDAProcessID
			edaProcessIDArgFail = &id
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO eda_messages
			   (eeg_id, process, message_type, subject, body, direction, status, xml_payload, error_msg, from_address, to_address, eda_process_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			eegIDArg,
			job.payload.Process,
			job.payload.Process,
			fmt.Sprintf("[SENDEFEHLER] %s %s", job.payload.Process, job.payload.GemeinschaftID),
			"",
			DirectionOutbound,
			StatusError,
			job.payload.XMLPayload,
			sendErr.Error(),
			job.payload.From,
			job.payload.To,
			edaProcessIDArgFail,
		); err != nil {
			w.log.Warn("failed to record failed outbound eda_message", "job_id", job.id, "error", err)
		}
	}

	return tx.Commit(ctx)
}

// sendJob sends a single EDA job via the transport.
// For MAIL mode, creates a per-EEG MailTransport from the EEG's stored credentials.
// Returns the sent message (with Subject populated by the transport) and any error.
func (w *Worker) sendJob(ctx context.Context, job edaJob) (*types.Message, error) {
	msg := &types.Message{
		ID:             job.id.String(),
		Process:        job.payload.Process,
		Direction:      DirectionOutbound,
		From:           job.payload.From,
		To:             job.payload.To,
		GemeinschaftID: job.payload.GemeinschaftID,
		XMLPayload:     job.payload.XMLPayload,
		CreatedAt:      time.Now().UTC(),
	}

	tr := w.transport
	if w.transportMode == "MAIL" && job.payload.EegID != uuid.Nil {
		eeg, err := w.eegRepo.GetByIDInternal(ctx, job.payload.EegID)
		if err != nil || eeg == nil {
			return nil, fmt.Errorf("EEG %s not found for MAIL send: %w", job.payload.EegID, err)
		}
		cfg := transport.MailConfig{
			IMAPHost:     eeg.EDAIMAPHost,
			IMAPUser:     eeg.EDAIMAPUser,
			IMAPPassword: eeg.EDAIMAPPassword,
			SMTPHost:     eeg.EDASmtpHost,
			SMTPUser:     eeg.EDASmtpUser,
			SMTPPassword: eeg.EDASmtpPassword,
			SMTPFrom:     eeg.EDASmtpFrom,
		}
		mt, err := transport.NewMailTransport(cfg, w.log)
		if err != nil {
			return nil, fmt.Errorf("EEG %s MAIL transport not configured: %w", eeg.Name, err)
		}
		tr = mt
	}

	if tr == nil {
		return nil, fmt.Errorf("no transport available for send (transportMode=%s)", w.transportMode)
	}
	return msg, tr.Send(ctx, msg)
}

// processCMRevoke handles an inbound CMRevoke message:
//   - CM_REV_CUS (AUFHEBUNG_CCMS): customer revoked their own consent via the Netzbetreiber.
//   - CM_REV_IMP (AUFHEBUNG_CCMS_IMP): Netzbetreiber revokes due to impossibility.
//
// Looks up the original EC_REQ_ONL process by ConsentID, marks it completed, sets
// abgemeldet_am on the meter point from the ConsentEnd date, and sends an operator
// notification for CM_REV_IMP (involuntary revocation).
// Returns the EEG ID of the matched process so the caller can tag the stored message.
func (w *Worker) processCMRevoke(ctx context.Context, msg *types.Message) (uuid.UUID, error) {
	result, err := edaxml.ParseCMRevoke(msg.XMLPayload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse CMRevoke: %w", err)
	}

	isIMP := result.MessageCode == "AUFHEBUNG_CCMS_IMP"

	// The ConsentID in the CMRevoke references the ConversationID of the original process.
	consentID := result.ConsentID
	if consentID == "" {
		// Fall back to the message's own ConversationID.
		consentID = result.ConversationID
	}
	if consentID == "" {
		w.log.Warn("CMRevoke has no ConsentID or ConversationID — cannot match process",
			"message_code", result.MessageCode,
			"metering_point", result.MeteringPoint,
		)
		return uuid.Nil, nil
	}

	// Re-add UUID hyphens if the ID was stripped in the XML.
	if len(consentID) == 32 {
		consentID = consentID[:8] + "-" + consentID[8:12] + "-" + consentID[12:16] + "-" + consentID[16:20] + "-" + consentID[20:]
	}

	proc, err := w.edaProcRepo.GetByConversationID(ctx, consentID)
	if err != nil {
		// No matching process — log and continue; message is still stored.
		w.log.Warn("CMRevoke: no matching EDA process found",
			"message_code", result.MessageCode,
			"consent_id", consentID,
			"metering_point", result.MeteringPoint,
		)
		return uuid.Nil, nil
	}

	now := time.Now().UTC()
	errMsg := "Zustimmung durch Kunden widerrufen (CM_REV_CUS)"
	if isIMP {
		errMsg = "Anmeldung durch Netzbetreiber aufgehoben (CM_REV_IMP)"
	}
	if err := w.edaProcRepo.UpdateStatus(ctx, proc.ID, "completed", &now, errMsg); err != nil {
		return uuid.Nil, fmt.Errorf("update EDA process status after CMRevoke: %w", err)
	}

	// Set abgemeldet_am on the meter point using the ConsentEnd date from the message.
	if result.ConsentEnd != "" && proc.MeterPointID != nil && *proc.MeterPointID != uuid.Nil {
		if consentEndDate, parseErr := time.Parse("2006-01-02", result.ConsentEnd); parseErr == nil {
			if err := w.meterPointRepo.UpdateAbgemeldetAm(ctx, *proc.MeterPointID, consentEndDate.UTC()); err != nil {
				w.log.Warn("CMRevoke: failed to set abgemeldet_am",
					"meter_point_id", *proc.MeterPointID,
					"consent_end", result.ConsentEnd,
					"error", err,
				)
			} else {
				w.log.Info("CMRevoke: abgemeldet_am set on meter point",
					"meter_point_id", *proc.MeterPointID,
					"consent_end", result.ConsentEnd,
				)
			}
		}
	}

	w.log.Info("CMRevoke: EDA process marked completed",
		"process_id", proc.ID,
		"message_code", result.MessageCode,
		"consent_id", consentID,
		"metering_point", result.MeteringPoint,
		"consent_end", result.ConsentEnd,
	)

	// For CM_REV_IMP (involuntary revocation by NB): notify the operator.
	if isIMP {
		proc.Status = "completed"
		proc.ErrorMsg = errMsg
		if eeg, eegErr := w.eegRepo.GetByIDInternal(ctx, proc.EegID); eegErr == nil {
			go w.sendEDAErrorNotification(ctx, proc, eeg)
		}
	}

	return proc.EegID, nil
}

// sendEDAErrorNotification emails the EEG operator when an EDA process reaches
// status "error" or "rejected". Called in a goroutine, non-blocking.
// It is idempotent: if error_notification_sent_at is already set the email is skipped.
func (w *Worker) sendEDAErrorNotification(ctx context.Context, proc *domain.EDAProcess, eeg *domain.EEG) {
	if proc.ErrorNotificationSentAt != nil {
		// Notification already sent for this process — skip.
		return
	}
	if eeg.SMTPHost == "" || eeg.SMTPFrom == "" {
		w.log.Warn("EDA error notification: no SMTP configured for EEG — skipping",
			"eeg_id", eeg.ID,
			"process_id", proc.ID,
		)
		return
	}

	subject := fmt.Sprintf("[EDA Fehler] %s für %s – %s", proc.ProcessType, proc.Zaehlpunkt, proc.Status)
	errDetails := proc.ErrorMsg
	if errDetails == "" {
		errDetails = "(keine Fehlermeldung)"
	}
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #dc2626;">EDA-Prozess fehlgeschlagen</h2>
<p>Ein EDA-Prozess für die Energiegemeinschaft <strong>%s</strong> ist fehlgeschlagen.</p>
<table style="border-collapse: collapse; width: 100%%; font-size: 14px;">
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600; width: 40%%;">Prozesstyp</td>
      <td style="padding: 6px 12px;">%s</td></tr>
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600;">Zählpunkt</td>
      <td style="padding: 6px 12px;"><code>%s</code></td></tr>
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600;">Status</td>
      <td style="padding: 6px 12px; color: #dc2626; font-weight: 600;">%s</td></tr>
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600;">Fehlermeldung</td>
      <td style="padding: 6px 12px;">%s</td></tr>
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600;">Prozess-ID</td>
      <td style="padding: 6px 12px; font-size: 12px; color: #64748b;">%s</td></tr>
</table>
<p style="margin-top: 24px;">Bitte prüfen Sie den Prozess in der EDA-Verwaltung und leiten Sie ggf. eine Korrektur ein.</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Diese Nachricht wurde automatisch generiert.</p>
</body>
</html>`, eeg.Name, proc.ProcessType, proc.Zaehlpunkt, proc.Status, errDetails, proc.ID)

	var msgBuilder strings.Builder
	msgBuilder.WriteString("From: " + eeg.SMTPFrom + "\r\n")
	msgBuilder.WriteString("To: " + eeg.SMTPFrom + "\r\n")
	msgBuilder.WriteString("Subject: " + subject + "\r\n")
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	smtpHost := eeg.SMTPHost
	if idx := strings.Index(smtpHost, ":"); idx != -1 {
		smtpHost = smtpHost[:idx]
	}
	var smtpAuth smtp.Auth
	if eeg.SMTPUser != "" {
		smtpAuth = smtp.PlainAuth("", eeg.SMTPUser, eeg.SMTPPassword, smtpHost)
	}
	if err := smtp.SendMail(eeg.SMTPHost, smtpAuth, eeg.SMTPFrom, []string{eeg.SMTPFrom}, []byte(msgBuilder.String())); err != nil {
		w.log.Warn("EDA error notification email failed",
			"process_id", proc.ID,
			"eeg_id", eeg.ID,
			"error", err,
		)
		return
	}
	w.log.Info("EDA error notification sent",
		"process_id", proc.ID,
		"process_type", proc.ProcessType,
		"zaehlpunkt", proc.Zaehlpunkt,
		"status", proc.Status,
	)

	if err := w.edaProcRepo.SetErrorNotificationSent(ctx, proc.ID); err != nil {
		w.log.Warn("failed to set error_notification_sent_at",
			"process_id", proc.ID,
			"error", err,
		)
	}
}

// ResendConfirmationEmails re-sends the Anmeldung confirmation email for the
// given EDA process IDs. Intended for operator recovery after missed ABSCHLUSS_ECON.
// Only processes of type EC_REQ_ONL / EC_EINZEL_ANM with a linked meter point are handled.
func (w *Worker) ResendConfirmationEmails(ctx context.Context, processIDs []uuid.UUID) []error {
	var errs []error
	for _, id := range processIDs {
		proc, err := w.edaProcRepo.GetByID(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("process %s not found: %w", id, err))
			continue
		}
		if proc.MeterPointID == nil || *proc.MeterPointID == uuid.Nil {
			errs = append(errs, fmt.Errorf("process %s has no meter_point_id", id))
			continue
		}
		mp, err := w.meterPointRepo.GetByID(ctx, *proc.MeterPointID)
		if err != nil {
			errs = append(errs, fmt.Errorf("meter point for process %s not found: %w", id, err))
			continue
		}
		confirmedDate := time.Time{}
		if mp.RegistriertSeit != nil {
			confirmedDate = *mp.RegistriertSeit
		}
		w.sendAnmeldungConfirmationEmail(ctx, *proc.MeterPointID, proc.EegID, proc.Zaehlpunkt, confirmedDate)
		w.log.Info("ResendConfirmationEmails: sent", "process_id", id, "zaehlpunkt", proc.Zaehlpunkt)
	}
	return errs
}

// sendAnmeldungConfirmationEmail notifies the member that their Zählpunkt has
// been confirmed by the Netzbetreiber. Called in a goroutine after ABSCHLUSS_ECON.
func (w *Worker) sendAnmeldungConfirmationEmail(ctx context.Context, meterPointID, eegID uuid.UUID, zaehlpunkt string, confirmedDate time.Time) {
	mp, err := w.meterPointRepo.GetByID(ctx, meterPointID)
	if err != nil {
		w.log.Warn("anmeldung email: meter point not found", "meter_point_id", meterPointID, "error", err)
		return
	}
	member, err := w.memberRepo.GetByID(ctx, mp.MemberID)
	if err != nil || member.Email == "" {
		w.log.Warn("anmeldung email: member not found or no email", "member_id", mp.MemberID, "error", err)
		return
	}
	eeg, err := w.eegRepo.GetByIDInternal(ctx, eegID)
	if err != nil || eeg.SMTPHost == "" {
		return // no SMTP configured — skip silently
	}

	dateStr := ""
	if !confirmedDate.IsZero() {
		dateStr = confirmedDate.Format("02.01.2006")
	}
	dateLine := ""
	if dateStr != "" {
		dateLine = fmt.Sprintf(`<p>Ihr Zählpunkt ist ab <strong>%s</strong> aktiv.</p>`, dateStr)
	}

	portalSection := ""
	if w.webBaseURL != "" {
		portalSection = fmt.Sprintf(`<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p>Im <strong>Mitglieder-Portal</strong> können Sie Ihre Energiedaten und Rechnungen jederzeit einsehen:</p>
<p><a href="%s/portal" style="color: #1e40af;">%s/portal</a></p>`, w.webBaseURL, w.webBaseURL)
	}

	subject := fmt.Sprintf("Ihr Zählpunkt wurde von %s bestätigt", eeg.Name)
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #1e40af;">Zählpunkt erfolgreich angemeldet</h2>
<p>Liebe(r) %s,</p>
<p>Ihr Zählpunkt <strong><code>%s</code></strong> wurde vom Netzbetreiber bestätigt und ist nun in der Energiegemeinschaft <strong>%s</strong> aktiv.</p>
%s
<p style="color: #475569; font-size: 14px;">Die Verrechnung der gemeinschaftlich erzeugten Energie beginnt ab dem bestätigten Datum automatisch.</p>
%s
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Bei Fragen wenden Sie sich bitte direkt an die Energiegemeinschaft.</p>
</body>
</html>`, member.Name1, zaehlpunkt, eeg.Name, dateLine, portalSection)

	var msgBuilder strings.Builder
	msgBuilder.WriteString("From: " + eeg.SMTPFrom + "\r\n")
	msgBuilder.WriteString("To: " + member.Email + "\r\n")
	msgBuilder.WriteString("Subject: " + subject + "\r\n")
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	host := eeg.SMTPHost
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	var smtpAuth smtp.Auth
	if eeg.SMTPUser != "" {
		smtpAuth = smtp.PlainAuth("", eeg.SMTPUser, eeg.SMTPPassword, host)
	}
	if err := smtp.SendMail(eeg.SMTPHost, smtpAuth, eeg.SMTPFrom, []string{member.Email}, []byte(msgBuilder.String())); err != nil {
		w.log.Warn("anmeldung confirmation email failed", "member_id", member.ID, "error", err)
	} else {
		w.log.Info("anmeldung confirmation email sent", "member_id", member.ID, "zaehlpunkt", zaehlpunkt)
	}
}
