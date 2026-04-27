package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// MemberPortalHandler handles the member self-service portal endpoints.
type MemberPortalHandler struct {
	portalRepo     *repository.MemberPortalRepository
	memberRepo     *repository.MemberRepository
	meterPointRepo *repository.MeterPointRepository
	participRepo   *repository.ParticipationRepository
	readingRepo    *repository.ReadingRepository
	invoiceRepo    *repository.InvoiceRepository
	eegRepo        *repository.EEGRepository
	edaProcRepo    *repository.EDAProcessRepository
	jobRepo        *repository.JobRepository
	webBaseURL     string
}

// NewMemberPortalHandler creates a MemberPortalHandler.
func NewMemberPortalHandler(
	portalRepo *repository.MemberPortalRepository,
	memberRepo *repository.MemberRepository,
	meterPointRepo *repository.MeterPointRepository,
	participRepo *repository.ParticipationRepository,
	readingRepo *repository.ReadingRepository,
	invoiceRepo *repository.InvoiceRepository,
	eegRepo *repository.EEGRepository,
	edaProcRepo *repository.EDAProcessRepository,
	jobRepo *repository.JobRepository,
	webBaseURL string,
) *MemberPortalHandler {
	return &MemberPortalHandler{
		portalRepo:     portalRepo,
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		participRepo:   participRepo,
		readingRepo:    readingRepo,
		invoiceRepo:    invoiceRepo,
		eegRepo:        eegRepo,
		edaProcRepo:    edaProcRepo,
		jobRepo:        jobRepo,
		webBaseURL:     webBaseURL,
	}
}

// RequestLink handles POST /api/v1/public/portal/request-link
// Body: {"email": "...", "eeg_id": "..."} — eeg_id is optional.
// If eeg_id is omitted and the email matches multiple EEGs, returns a choices list
// for the frontend to show a selection step. Always returns HTTP 200.
//
//	@Summary		Request a magic login link for the member portal
//	@Description	Looks up the member by email (and optionally eeg_id) and sends a magic login link by email. If the email is associated with multiple EEGs and no eeg_id is given, returns a list of EEG choices for the user to select. Always returns HTTP 200 to avoid email enumeration.
//	@Tags			Mitgliederportal
//	@Accept			json
//	@Produce		json
//	@Param			body	body		object{email=string,eeg_id=string}	true	"Member email and optional EEG ID"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Router			/public/portal/request-link [post]
func (h *MemberPortalHandler) RequestLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		EegID string `json:"eeg_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// --- Case: specific EEG selected by the user ---
	if req.EegID != "" {
		eegID, err := uuid.Parse(req.EegID)
		if err != nil {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		member, err := h.portalRepo.FindMemberByEmailAndEEG(r.Context(), req.Email, eegID)
		if err != nil || member == nil {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		token, err := h.portalRepo.CreateLinkSession(r.Context(), member.ID, member.EegID)
		if err != nil {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		eeg, _ := h.eegRepo.GetByIDInternal(r.Context(), eegID)
		if eeg == nil || !eeg.IsDemo {
			portalLink := fmt.Sprintf("%s/portal/%s", h.webBaseURL, token)
			go h.sendPortalLinkEmail(eegID, member.Email, member.Name1+" "+member.Name2, portalLink)
		}
		w.Write([]byte(`{"ok":true}`))
		return
	}

	// --- Case: no EEG specified — look up all matching members ---
	choices, err := h.portalRepo.FindMembersByEmail(r.Context(), req.Email)
	if err != nil || len(choices) == 0 {
		w.Write([]byte(`{"ok":true}`))
		return
	}

	// Exactly one match → send link immediately (no selection needed)
	if len(choices) == 1 {
		c := choices[0]
		token, err := h.portalRepo.CreateLinkSession(r.Context(), c.MemberID, c.EegID)
		if err != nil {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		if !c.IsDemo {
			portalLink := fmt.Sprintf("%s/portal/%s", h.webBaseURL, token)
			go h.sendPortalLinkEmail(c.EegID, c.Email, c.Name1+" "+c.Name2, portalLink)
		}
		w.Write([]byte(`{"ok":true}`))
		return
	}

	// Multiple matches → ask frontend to show EEG selection
	type choiceItem struct {
		EegID   string `json:"eeg_id"`
		EegName string `json:"eeg_name"`
	}
	items := make([]choiceItem, len(choices))
	for i, c := range choices {
		items[i] = choiceItem{EegID: c.EegID.String(), EegName: c.EegName}
	}
	resp := struct {
		OK      bool         `json:"ok"`
		Choices []choiceItem `json:"choices"`
	}{OK: true, Choices: items}
	json.NewEncoder(w).Encode(resp)
}

func (h *MemberPortalHandler) sendPortalLinkEmail(eegID uuid.UUID, toEmail, name, link string) {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), eegID)
	if err != nil || eeg == nil || eeg.SMTPHost == "" {
		slog.Error("sendPortalLinkEmail: EEG SMTP not configured", "eeg_id", eegID)
		return
	}

	fullName := strings.TrimSpace(name)
	if fullName == "" {
		fullName = "Mitglied"
	}
	subject := "Ihr Mitglieder-Portal Login-Link"
	body := fmt.Sprintf(`Sehr geehrte/r %s,

Sie haben einen Zugangslink für Ihr Mitglieder-Portal angefordert.

Klicken Sie auf den folgenden Link um sich einzuloggen:

%s

Dieser Link ist 30 Minuten gültig und kann nur einmal verwendet werden.

Falls Sie keinen Login-Link angefordert haben, können Sie diese E-Mail ignorieren.

Mit freundlichen Grüßen
Ihr EEG-Team

--
Dieser Link läuft in 30 Minuten ab.`, fullName, link)

	msg := []byte("From: " + eeg.SMTPFrom + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body)

	var auth smtp.Auth
	if eeg.SMTPUser != "" {
		host := eeg.SMTPHost
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", eeg.SMTPUser, eeg.SMTPPassword, host)
	}
	if err := smtp.SendMail(eeg.SMTPHost, auth, eeg.SMTPFrom, []string{toEmail}, msg); err != nil {
		slog.Error("sendPortalLinkEmail: failed to send", "error", err, "to", toEmail)
	}
}

// ExchangeToken handles POST /api/v1/public/portal/exchange
// Body: {"token": "..."}
// Returns: {"session_token": "...", "member_id": "...", "eeg_id": "..."}
//
//	@Summary		Exchange magic link token for a portal session
//	@Description	Exchanges a one-time magic link token (from the login email) for a long-lived session token. The session token must be passed as X-Portal-Session header on subsequent portal requests.
//	@Tags			Mitgliederportal
//	@Accept			json
//	@Produce		json
//	@Param			body	body		object{token=string}	true	"One-time magic link token"
//	@Success		200		{object}	map[string]string	"session_token, member_id, eeg_id"
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string	"Link invalid or expired"
//	@Router			/public/portal/exchange [post]
func (h *MemberPortalHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	sessionToken, memberID, eegID, err := h.portalRepo.ExchangeLinkToken(r.Context(), req.Token)
	if err != nil {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_token": sessionToken,
		"member_id":     memberID.String(),
		"eeg_id":        eegID.String(),
	})
}

// portalAuth is a helper that validates the X-Portal-Session header and returns member/eeg IDs.
func (h *MemberPortalHandler) portalAuth(r *http.Request) (memberID, eegID uuid.UUID, ok bool) {
	token := r.Header.Get("X-Portal-Session")
	if token == "" {
		return uuid.Nil, uuid.Nil, false
	}
	mID, eID, err := h.portalRepo.FindBySessionToken(r.Context(), token)
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return mID, eID, true
}

// GetMe handles GET /api/v1/public/portal/me
//
//	@Summary		Get authenticated member's profile and EEG info
//	@Description	Returns the member record and associated EEG details for the portal session identified by the X-Portal-Session header.
//	@Tags			Mitgliederportal
//	@Produce		json
//	@Param			X-Portal-Session	header		string	true	"Portal session token"
//	@Success		200					{object}	map[string]interface{}	"member and eeg fields"
//	@Failure		401					{object}	map[string]string
//	@Failure		404					{object}	map[string]string
//	@Router			/public/portal/me [get]
func (h *MemberPortalHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"member": member,
		"eeg":    eeg,
	})
}

// GetEnergy handles GET /api/v1/public/portal/energy
//
//	@Summary		Get member's monthly energy data
//	@Description	Returns monthly energy readings aggregated for the authenticated member. Requires a valid X-Portal-Session header.
//	@Tags			Mitgliederportal
//	@Produce		json
//	@Param			X-Portal-Session	header		string	true	"Portal session token"
//	@Success		200					{array}		interface{}
//	@Failure		401					{object}	map[string]string
//	@Failure		500					{object}	map[string]string
//	@Router			/public/portal/energy [get]
var validPortalGranularities = map[string]bool{
	"year": true, "month": true, "day": true, "15min": true,
}

func (h *MemberPortalHandler) GetEnergy(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	granularity := r.URL.Query().Get("granularity")

	// No params → legacy monthly default (server-side initial render)
	if fromStr == "" || toStr == "" || granularity == "" {
		rows, err := h.readingRepo.GetMemberMonthlyEnergy(r.Context(), memberID)
		if err != nil {
			jsonError(w, "query error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
		return
	}

	if !validPortalGranularities[granularity] {
		jsonError(w, "invalid granularity: must be year, month, day or 15min", http.StatusBadRequest)
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		jsonError(w, "invalid from date (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		jsonError(w, "invalid to date (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Row-cap: prevent accidental huge queries
	diff := to.Sub(from)
	if granularity == "15min" && diff > 7*24*time.Hour {
		jsonError(w, "15min granularity: max range is 7 days", http.StatusBadRequest)
		return
	}
	if granularity == "day" && diff > 366*24*time.Hour {
		jsonError(w, "day granularity: max range is 366 days", http.StatusBadRequest)
		return
	}

	rows, err := h.readingRepo.GetMemberEnergy(r.Context(), memberID, from, to, granularity)
	if err != nil {
		jsonError(w, "query error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

// GetInvoices handles GET /api/v1/public/portal/invoices
//
//	@Summary		Get member's invoices
//	@Description	Returns all invoices for the authenticated member. Requires a valid X-Portal-Session header.
//	@Tags			Mitgliederportal
//	@Produce		json
//	@Param			X-Portal-Session	header		string	true	"Portal session token"
//	@Success		200					{array}		domain.Invoice
//	@Failure		401					{object}	map[string]string
//	@Failure		500					{object}	map[string]string
//	@Router			/public/portal/invoices [get]
func (h *MemberPortalHandler) GetInvoices(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	invs, err := h.invoiceRepo.ListByMember(r.Context(), eegID, memberID)
	if err != nil {
		jsonError(w, "query error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if invs == nil {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(invs)
}

// GetInvoicePDF handles GET /api/v1/public/portal/invoices/{invoiceID}/pdf
//
//	@Summary		Download invoice PDF
//	@Description	Streams the PDF file for the given invoice. The invoice must belong to the member identified by the X-Portal-Session header. Returns 404 if the PDF has not yet been generated.
//	@Tags			Mitgliederportal
//	@Produce		application/pdf
//	@Param			X-Portal-Session	header		string	true	"Portal session token"
//	@Param			invoiceID			path		string	true	"Invoice UUID"
//	@Success		200					{file}		binary
//	@Failure		400					{object}	map[string]string
//	@Failure		401					{object}	map[string]string
//	@Failure		404					{object}	map[string]string
//	@Router			/public/portal/invoices/{invoiceID}/pdf [get]
func (h *MemberPortalHandler) GetInvoicePDF(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	invoiceIDStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		jsonError(w, "invalid invoice ID", http.StatusBadRequest)
		return
	}

	// Load invoice and verify ownership
	inv, err := h.invoiceRepo.GetByID(r.Context(), invoiceID)
	if err != nil || inv.MemberID != memberID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if inv.PdfPath == "" {
		jsonError(w, "PDF not available", http.StatusNotFound)
		return
	}

	f, err := os.Open(inv.PdfPath)
	if err != nil {
		jsonError(w, "PDF not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Rechnung_%s.pdf"`, invoiceIDStr[:8]))
	io.Copy(w, f)
}

// portalMeterPoint is the meter point view returned to the member portal.
type portalMeterPoint struct {
	ID                  uuid.UUID `json:"id"`
	Zaehlpunkt          string    `json:"zaehlpunkt"`
	Direction           string    `json:"direction"`
	Status              string    `json:"status"`
	ParticipationFactor float64   `json:"participation_factor"`
}

// GetMeterPoints handles GET /api/v1/public/portal/meter-points
func (h *MemberPortalHandler) GetMeterPoints(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	mps, err := h.meterPointRepo.ListByMember(r.Context(), memberID)
	if err != nil {
		jsonError(w, "query error", http.StatusInternalServerError)
		return
	}

	result := make([]portalMeterPoint, 0, len(mps))
	today := time.Now().UTC()
	for _, mp := range mps {
		factor := 100.0
		if p, err := h.participRepo.GetActiveForPeriod(r.Context(), eegID, mp.ID, today); err == nil {
			factor = p.ParticipationFactor
		}
		result = append(result, portalMeterPoint{
			ID:                  mp.ID,
			Zaehlpunkt:          mp.Zaehlpunkt,
			Direction:           mp.Energierichtung,
			Status:              mp.Status,
			ParticipationFactor: factor,
		})
	}

	jsonOK(w, result)
}

// ChangeParticipationFactor handles POST /api/v1/public/portal/change-factor
// Body: {"zaehlpunkt": "...", "participation_factor": 75.0}
// Creates an EC_PRTFACT_CHG EDA process.
func (h *MemberPortalHandler) ChangeParticipationFactor(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Time-of-day restriction: 09:00–17:00 Vienna time (per EDA protocol).
	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	now := time.Now().In(viennaLoc)
	if now.Hour() < 9 || now.Hour() >= 17 {
		jsonError(w, "Änderungen des Teilnahmefaktors sind nur zwischen 09:00 und 17:00 Uhr (Wien) möglich", http.StatusUnprocessableEntity)
		return
	}

	var req struct {
		Zaehlpunkt          string  `json:"zaehlpunkt"`
		ParticipationFactor float64 `json:"participation_factor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Zaehlpunkt == "" {
		jsonError(w, "zaehlpunkt is required", http.StatusBadRequest)
		return
	}
	if req.ParticipationFactor <= 0 || req.ParticipationFactor > 100 {
		jsonError(w, "participation_factor muss zwischen 0 und 100 liegen", http.StatusBadRequest)
		return
	}

	// Verify the meter point belongs to this member.
	mps, err := h.meterPointRepo.ListByMember(r.Context(), memberID)
	if err != nil {
		jsonError(w, "query error", http.StatusInternalServerError)
		return
	}
	belongs := false
	for _, mp := range mps {
		if mp.Zaehlpunkt == req.Zaehlpunkt {
			belongs = true
			break
		}
	}
	if !belongs {
		jsonError(w, "Zählpunkt nicht gefunden", http.StatusNotFound)
		return
	}

	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	if eeg.EdaMarktpartnerID == "" || eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA ist für diese Energiegemeinschaft nicht konfiguriert", http.StatusBadRequest)
		return
	}

	// Duplicate check: only one change per Zählpunkt per day.
	dup, err := h.edaProcRepo.HasPendingFactorChangeToday(r.Context(), eegID, req.Zaehlpunkt)
	if err != nil {
		jsonError(w, "failed to check duplicate", http.StatusInternalServerError)
		return
	}
	if dup {
		jsonError(w, "Es gibt bereits eine Teilnahmefaktor-Änderung für diesen Zählpunkt heute", http.StatusConflict)
		return
	}

	// Valid from tomorrow.
	tomorrow := now.AddDate(0, 0, 1)
	validFrom := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)

	convID := uuid.NewString()
	xmlBody, err := edaxml.BuildCPRequest(edaxml.CPRequestParams{
		From:                eeg.EdaMarktpartnerID,
		To:                  eeg.EdaNetzbetreiberID,
		GemeinschaftID:      eeg.GemeinschaftID,
		Process:             "EC_PRTFACT_CHG",
		MessageID:           uuid.NewString(),
		ConversationID:      convID,
		Zaehlpunkt:          req.Zaehlpunkt,
		ValidFrom:           validFrom,
		ParticipationFactor: req.ParticipationFactor,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
		return
	}

	var mpID *uuid.UUID
	if mp, err := h.meterPointRepo.GetByZaehlpunkt(r.Context(), req.Zaehlpunkt); err == nil {
		id := mp.ID
		mpID = &id
	}

	factor := req.ParticipationFactor
	proc := &domain.EDAProcess{
		EegID:               eegID,
		MeterPointID:        mpID,
		ProcessType:         "EC_PRTFACT_CHG",
		Status:              "pending",
		ConversationID:      convID,
		Zaehlpunkt:          req.Zaehlpunkt,
		ValidFrom:           &validFrom,
		ParticipationFactor: &factor,
		InitiatedAt:         time.Now().UTC(),
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		slog.Error("failed to create EDA process for portal factor change", "error", err)
		jsonError(w, "failed to create EDA process", http.StatusInternalServerError)
		return
	}
	if err := h.jobRepo.EnqueueEDA(r.Context(), "EC_PRTFACT_CHG", eeg.EdaMarktpartnerID, eeg.EdaNetzbetreiberID,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		slog.Error("failed to enqueue EDA job for portal factor change", "error", err)
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}
