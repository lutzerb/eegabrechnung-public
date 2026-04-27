package billing

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// GapChecker runs hourly and detects registered meter points without recent energy readings.
type GapChecker struct {
	eegRepo        *repository.EEGRepository
	meterPointRepo *repository.MeterPointRepository
	log            *slog.Logger
}

// NewGapChecker creates a new GapChecker.
func NewGapChecker(eegRepo *repository.EEGRepository, meterPointRepo *repository.MeterPointRepository) *GapChecker {
	return &GapChecker{
		eegRepo:        eegRepo,
		meterPointRepo: meterPointRepo,
		log:            slog.Default().With("component", "gap-checker"),
	}
}

// Run starts the hourly gap-check loop. Call this in a goroutine; it runs until ctx is cancelled.
func (g *GapChecker) Run(ctx context.Context) {
	for {
		// Fire at the next whole hour
		now := time.Now()
		next := now.Truncate(time.Hour).Add(time.Hour)
		delay := time.Until(next)
		g.log.Info("gap-checker: next check", "at", next.Format("2006-01-02 15:04 MST"), "in", delay.Round(time.Minute))

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			g.runChecks(ctx)
		}
	}
}

func (g *GapChecker) runChecks(ctx context.Context) {
	eegs, err := g.eegRepo.ListGapAlertEEGs(ctx)
	if err != nil {
		g.log.Error("gap-checker: failed to list EEGs", "error", err)
		return
	}
	for _, eeg := range eegs {
		g.checkEEG(ctx, eeg)
	}
}

func (g *GapChecker) checkEEG(ctx context.Context, eeg *domain.EEG) {
	log := g.log.With("eeg_id", eeg.ID, "eeg_name", eeg.Name)

	threshold := eeg.GapAlertThresholdDays
	if threshold <= 0 {
		threshold = 5
	}

	// 1a. Clear alerts for meter points that are no longer eligible (member inactive, abgemeldet, etc.).
	if err := g.meterPointRepo.ClearGapAlertForInactivePoints(ctx, eeg.ID); err != nil {
		log.Error("gap-checker: failed to clear inactive alerts", "error", err)
	}

	// 1b. Reset alerts for meter points that now have recent readings.
	if err := g.meterPointRepo.ResetGapAlertIfReadingsRecent(ctx, eeg.ID, threshold); err != nil {
		log.Error("gap-checker: failed to reset stale alerts", "error", err)
	}

	// 2. Find meter points with a gap.
	gaps, err := g.meterPointRepo.GetMeterPointsWithGap(ctx, eeg.ID, threshold)
	if err != nil {
		log.Error("gap-checker: failed to query gaps", "error", err)
		return
	}
	if len(gaps) == 0 {
		return
	}

	log.Warn("gap-checker: gap(s) detected", "count", len(gaps))

	// 3. Send email notification (if SMTP configured and not demo).
	if !eeg.IsDemo {
		g.sendGapEmail(eeg, gaps)
	}

	// 4. Mark each meter point so we don't spam.
	for _, gap := range gaps {
		if err := g.meterPointRepo.SetGapAlertSent(ctx, gap.ID); err != nil {
			log.Error("gap-checker: failed to set gap_alert_sent_at", "meter_point_id", gap.ID, "error", err)
		}
	}
}

func (g *GapChecker) sendGapEmail(eeg *domain.EEG, gaps []repository.MeterPointGap) {
	if eeg.SMTPHost == "" || eeg.SMTPFrom == "" {
		g.log.Warn("gap-checker: no SMTP configured — skipping email", "eeg_id", eeg.ID)
		return
	}

	subject := fmt.Sprintf("[Datenlücke] %d Zählpunkt(e) ohne Readings — %s",
		len(gaps), eeg.Name)

	rows := ""
	for _, gap := range gaps {
		lastStr := "noch keine Readings"
		if gap.LastReadingAt != nil {
			daysSince := time.Since(*gap.LastReadingAt).Hours() / 24
			lastStr = fmt.Sprintf("%s (vor %.0f Tag(en))", gap.LastReadingAt.Format("02.01.2006"), daysSince)
		}
		rows += fmt.Sprintf(`
  <tr>
    <td style="padding:6px 12px; border-bottom:1px solid #e2e8f0; font-family:monospace; font-size:12px;">%s</td>
    <td style="padding:6px 12px; border-bottom:1px solid #e2e8f0; font-size:13px;">%s</td>
    <td style="padding:6px 12px; border-bottom:1px solid #e2e8f0; font-size:13px; color:#dc2626;">%s</td>
  </tr>`, gap.Zaehlpunkt, gap.MemberName, lastStr)
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 650px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #ea580c;">Datenlücken-Alarm: fehlende Readings</h2>
<p>Bei der Energiegemeinschaft <strong>%s</strong> wurden %d Zählpunkt(e) erkannt,
   die seit mehr als <strong>%d Tagen</strong> keine Energiedaten geliefert haben.</p>
<table style="border-collapse: collapse; width: 100%%; font-size: 13px; margin-top: 16px;">
  <thead>
    <tr style="background:#f1f5f9;">
      <th style="text-align:left; padding:8px 12px; font-size:12px; font-weight:600; color:#475569;">Zählpunkt</th>
      <th style="text-align:left; padding:8px 12px; font-size:12px; font-weight:600; color:#475569;">Mitglied</th>
      <th style="text-align:left; padding:8px 12px; font-size:12px; font-weight:600; color:#475569;">Letztes Reading</th>
    </tr>
  </thead>
  <tbody>%s</tbody>
</table>
<p style="margin-top:24px; color:#d97706;">
  <strong>Bitte prüfen Sie den Datenimport oder die EDA-Verbindung für diese Zählpunkte.</strong>
</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">
  Diese Benachrichtigung wurde automatisch generiert. Der Alarm wird nicht wiederholt,
  bis wieder aktuelle Readings vorhanden sind.
</p>
</body></html>`,
		eeg.Name, len(gaps), eeg.GapAlertThresholdDays, rows)

	g.sendHTMLEmail(eeg, subject, body)
}

func (g *GapChecker) sendHTMLEmail(eeg *domain.EEG, subject, htmlBody string) {
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
		g.log.Warn("gap-checker: email send failed", "eeg_id", eeg.ID, "error", err)
	}
}
