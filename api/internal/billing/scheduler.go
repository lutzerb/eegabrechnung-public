package billing

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"
	_ "time/tzdata" // embed IANA timezone database (required in Alpine containers)

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// Scheduler runs daily auto-billing checks and creates draft billing runs.
type Scheduler struct {
	eegRepo     *repository.EEGRepository
	readingRepo *repository.ReadingRepository
	billingSvc  *Service
	log         *slog.Logger
}

// NewScheduler creates a billing scheduler.
func NewScheduler(
	eegRepo *repository.EEGRepository,
	readingRepo *repository.ReadingRepository,
	billingSvc *Service,
) *Scheduler {
	return &Scheduler{
		eegRepo:     eegRepo,
		readingRepo: readingRepo,
		billingSvc:  billingSvc,
		log:         slog.Default().With("component", "billing-scheduler"),
	}
}

// Run starts the scheduler loop. It fires once per day at 06:00 Vienna time.
// Call this in a goroutine; it runs until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	viennaLoc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		s.log.Error("failed to load Europe/Vienna timezone", "error", err)
		return
	}

	for {
		// Calculate time until next 06:00 Vienna
		now := time.Now().In(viennaLoc)
		next := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, viennaLoc)
		if !now.Before(next) {
			// Already past 06:00 today, schedule for tomorrow
			next = next.AddDate(0, 0, 1)
		}
		delay := time.Until(next)
		s.log.Info("auto-billing scheduler: next check", "at", next.Format("2006-01-02 15:04 MST"), "in", delay.Round(time.Minute))

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			s.runChecks(ctx, viennaLoc)
		}
	}
}

// runChecks iterates all EEGs with auto-billing enabled and acts on qualifying ones.
func (s *Scheduler) runChecks(ctx context.Context, viennaLoc *time.Location) {
	eegs, err := s.eegRepo.ListAutoBillingEEGs(ctx)
	if err != nil {
		s.log.Error("auto-billing: failed to list EEGs", "error", err)
		return
	}

	today := time.Now().In(viennaLoc)
	for _, eeg := range eegs {
		s.checkEEG(ctx, eeg, today)
	}
}

// checkEEG processes one EEG for the auto-billing run.
func (s *Scheduler) checkEEG(ctx context.Context, eeg *domain.EEG, today time.Time) {
	log := s.log.With("eeg_id", eeg.ID, "eeg_name", eeg.Name)

	// 1. Is today the configured billing day?
	if eeg.AutoBillingDayOfMonth == 0 || today.Day() != eeg.AutoBillingDayOfMonth {
		return
	}

	// 2. Was the last run less than 20 days ago? (prevents double-run after restart)
	if eeg.AutoBillingLastRunAt != nil {
		daysSince := today.Sub(*eeg.AutoBillingLastRunAt).Hours() / 24
		if daysSince < 20 {
			log.Info("auto-billing: skipping — last run was too recent", "days_since", daysSince)
			return
		}
	}

	// 3. Determine billing period
	periodStart, periodEnd := autoBillingPeriod(eeg.AutoBillingPeriod, today)
	log.Info("auto-billing: checking EEG",
		"period_start", periodStart.Format("2006-01-02"),
		"period_end", periodEnd.Format("2006-01-02"),
	)

	// 4. Data completeness check
	missingZPs, err := s.readingRepo.MissingReadingDays(ctx, eeg.ID, periodStart, periodEnd)
	if err != nil {
		log.Error("auto-billing: completeness check failed", "error", err)
		return
	}
	if len(missingZPs) > 0 {
		log.Warn("auto-billing: data gaps found — no run created", "missing_zaehlpunkte", missingZPs)
		s.sendWarningEmail(eeg, periodStart, periodEnd, missingZPs)
		return
	}

	// 5. Create draft billing run (no force — if overlap, skip)
	result, err := s.billingSvc.RunBilling(ctx, eeg.ID, RunOptions{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	if err != nil {
		if isOverlapError(err) {
			log.Info("auto-billing: billing run overlap — skipping", "error", err)
		} else {
			log.Error("auto-billing: failed to create billing run", "error", err)
			s.sendErrorEmail(eeg, periodStart, periodEnd, err)
		}
		return
	}

	// 6. Update last_run_at
	if updateErr := s.eegRepo.UpdateAutoBillingLastRun(ctx, eeg.ID); updateErr != nil {
		log.Error("auto-billing: failed to update last_run_at", "error", updateErr)
	}

	log.Info("auto-billing: draft billing run created",
		"billing_run_id", result.BillingRun.ID,
		"invoice_count", len(result.Invoices),
	)
	s.sendSuccessEmail(eeg, result.BillingRun.ID, periodStart, periodEnd, len(result.Invoices))
}

// autoBillingPeriod returns [start, end) for the previous calendar month or quarter.
func autoBillingPeriod(period string, today time.Time) (time.Time, time.Time) {
	y, m := today.Year(), today.Month()
	if period == "quarterly" {
		// Previous quarter
		qEnd := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC) // first day of current month
		qStart := qEnd.AddDate(0, -3, 0)
		return qStart, qEnd
	}
	// Monthly (default): previous calendar month
	periodEnd := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	periodStart := periodEnd.AddDate(0, -1, 0)
	return periodStart, periodEnd
}

func isOverlapError(err error) bool {
	_, ok := err.(*OverlapError)
	return ok
}

func (s *Scheduler) sendSuccessEmail(eeg *domain.EEG, runID uuid.UUID, from, to time.Time, invoiceCount int) {
	if eeg.SMTPHost == "" || eeg.SMTPFrom == "" {
		return
	}
	subject := fmt.Sprintf("[Auto-Abrechnung] Entwurf erstellt für %s – %s bis %s",
		eeg.Name, from.Format("01/2006"), to.AddDate(0, 0, -1).Format("01/2006"))
	body := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #16a34a;">Auto-Abrechnung: Entwurf erstellt</h2>
<p>Für die Energiegemeinschaft <strong>%s</strong> wurde automatisch ein Abrechnungsentwurf erstellt.</p>
<table style="border-collapse: collapse; width: 100%%; font-size: 14px;">
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600; width: 40%%;">Zeitraum</td>
      <td style="padding: 6px 12px;">%s bis %s</td></tr>
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600;">Rechnungen</td>
      <td style="padding: 6px 12px;">%d Entwürfe erstellt</td></tr>
  <tr><td style="padding: 6px 12px; background: #f1f5f9; font-weight: 600;">Lauf-ID</td>
      <td style="padding: 6px 12px; font-size: 12px; color: #64748b;">%s</td></tr>
</table>
<p style="margin-top: 24px; color: #d97706;"><strong>Bitte prüfen und finalisieren Sie den Abrechnungslauf in der Anwendung.</strong></p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Diese Nachricht wurde automatisch generiert.</p>
</body></html>`,
		eeg.Name,
		from.Format("02.01.2006"), to.AddDate(0, 0, -1).Format("02.01.2006"),
		invoiceCount, runID)
	s.sendHTMLEmail(eeg, subject, body)
}

func (s *Scheduler) sendWarningEmail(eeg *domain.EEG, from, to time.Time, missingZPs []string) {
	if eeg.SMTPHost == "" || eeg.SMTPFrom == "" {
		s.log.Warn("auto-billing: no SMTP configured — cannot send warning email", "eeg_id", eeg.ID)
		return
	}
	subject := fmt.Sprintf("[Auto-Abrechnung] ABGEBROCHEN — fehlende Daten für %s", eeg.Name)
	zpList := "<ul>"
	for _, zp := range missingZPs {
		zpList += fmt.Sprintf("<li><code>%s</code></li>", zp)
	}
	zpList += "</ul>"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #dc2626;">Auto-Abrechnung abgebrochen: fehlende Daten</h2>
<p>Die automatische Abrechnung für <strong>%s</strong> konnte nicht durchgeführt werden.</p>
<p><strong>Zeitraum:</strong> %s bis %s</p>
<p>Folgende Zählpunkte haben unvollständige Readings für den Abrechnungszeitraum:</p>
%s
<p>Bitte prüfen Sie den Datenimport und führen Sie die Abrechnung manuell durch, sobald alle Daten vorliegen.</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Diese Nachricht wurde automatisch generiert.</p>
</body></html>`,
		eeg.Name, from.Format("02.01.2006"), to.AddDate(0, 0, -1).Format("02.01.2006"), zpList)
	s.sendHTMLEmail(eeg, subject, body)
}

func (s *Scheduler) sendErrorEmail(eeg *domain.EEG, from, to time.Time, runErr error) {
	if eeg.SMTPHost == "" || eeg.SMTPFrom == "" {
		return
	}
	subject := fmt.Sprintf("[Auto-Abrechnung] FEHLER für %s", eeg.Name)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #dc2626;">Auto-Abrechnung fehlgeschlagen</h2>
<p>Bei der automatischen Abrechnung für <strong>%s</strong> ist ein Fehler aufgetreten.</p>
<p><strong>Zeitraum:</strong> %s bis %s</p>
<p><strong>Fehler:</strong> %s</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Diese Nachricht wurde automatisch generiert.</p>
</body></html>`,
		eeg.Name, from.Format("02.01.2006"), to.AddDate(0, 0, -1).Format("02.01.2006"), runErr)
	s.sendHTMLEmail(eeg, subject, body)
}

func (s *Scheduler) sendHTMLEmail(eeg *domain.EEG, subject, htmlBody string) {
	var msg strings.Builder
	msg.WriteString("From: " + eeg.SMTPFrom + "\r\n")
	msg.WriteString("To: " + eeg.SMTPFrom + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	host := eeg.SMTPHost
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	var auth smtp.Auth
	if eeg.SMTPUser != "" {
		auth = smtp.PlainAuth("", eeg.SMTPUser, eeg.SMTPPassword, host)
	}
	if err := smtp.SendMail(eeg.SMTPHost, auth, eeg.SMTPFrom, []string{eeg.SMTPFrom}, []byte(msg.String())); err != nil {
		s.log.Warn("auto-billing: email send failed", "eeg_id", eeg.ID, "error", err)
	}
}
