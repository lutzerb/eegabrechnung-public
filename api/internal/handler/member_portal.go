package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/invoice"
	"github.com/lutzerb/eegabrechnung/internal/mailutil"
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
	orgRepo        *repository.OrganizationRepository
	edaProcRepo    *repository.EDAProcessRepository
	jobRepo        *repository.JobRepository
	emailLogRepo   *repository.EmailLogRepository
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
	orgRepo *repository.OrganizationRepository,
	edaProcRepo *repository.EDAProcessRepository,
	jobRepo *repository.JobRepository,
	emailLogRepo *repository.EmailLogRepository,
) *MemberPortalHandler {
	return &MemberPortalHandler{
		portalRepo:     portalRepo,
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		participRepo:   participRepo,
		readingRepo:    readingRepo,
		invoiceRepo:    invoiceRepo,
		eegRepo:        eegRepo,
		orgRepo:        orgRepo,
		edaProcRepo:    edaProcRepo,
		jobRepo:        jobRepo,
		emailLogRepo:   emailLogRepo,
	}
}

// resolveTenantOrg resolves the organization for the domain the browser used, from
// the X-Tenant-Host header set by the Next.js proxy (the Go API's own inbound Host
// header always reflects the internal Docker service name, not the original public
// domain). Unknown/missing host is a hard failure — never fall back to searching
// across all organizations, or member-email lookups leak across tenants.
func (h *MemberPortalHandler) resolveTenantOrg(r *http.Request) (uuid.UUID, error) {
	host := r.Header.Get("X-Tenant-Host")
	if host == "" {
		return uuid.Nil, fmt.Errorf("missing X-Tenant-Host header")
	}
	return h.orgRepo.GetOrgIDByHost(r.Context(), host)
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

	orgID, err := h.resolveTenantOrg(r)
	if err != nil {
		jsonError(w, "unknown domain", http.StatusBadRequest)
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
		member, err := h.portalRepo.FindMemberByEmailAndEEG(r.Context(), req.Email, eegID, orgID)
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
		if eeg != nil && !eeg.IsDemo {
			portalLink := fmt.Sprintf("%s/portal/%s", eeg.PortalBaseURL, token)
			go h.sendPortalLinkEmail(eegID, member.Email, member.Name1+" "+member.Name2, portalLink)
		}
		w.Write([]byte(`{"ok":true}`))
		return
	}

	// --- Case: no EEG specified — look up all matching members ---
	choices, err := h.portalRepo.FindMembersByEmail(r.Context(), req.Email, orgID)
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
			portalLink := fmt.Sprintf("%s/portal/%s", c.PortalBaseURL, token)
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

	msg := []byte(mailutil.Headers(eeg.SMTPFrom, toEmail, subject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body)

	cfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}
	if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "portal_link", toEmail, subject, nil, nil, msg); err != nil {
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

// LoginPassword handles POST /api/v1/public/portal/login-password
// Body: {"email": "...", "password": "...", "eeg_id": "..."} — eeg_id is optional.
// Password is checked against member_portal_credentials, which is keyed by email
// (not member_id), so one password unlocks every EEG membership sharing that email —
// same identity model as the magic-link flow's EEG-choice step. If the email has
// more than one active membership and no eeg_id was given, returns a choices list
// (same shape as RequestLink) for the frontend to disambiguate; the client then
// resubmits with the chosen eeg_id and the same password. On success, returns the
// same shape as ExchangeToken.
func (h *MemberPortalHandler) LoginPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		EegID    string `json:"eeg_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	const invalidMsg = "E-Mail oder Passwort ungültig."

	orgID, err := h.resolveTenantOrg(r)
	if err != nil {
		jsonError(w, "unknown domain", http.StatusBadRequest)
		return
	}

	hash, err := h.portalRepo.FindPasswordHash(r.Context(), req.Email)
	if err != nil || hash == "" || !auth.CheckPassword(hash, req.Password) {
		jsonError(w, invalidMsg, http.StatusUnauthorized)
		return
	}

	choices, err := h.portalRepo.FindMembersByEmail(r.Context(), req.Email, orgID)
	if err != nil || len(choices) == 0 {
		jsonError(w, invalidMsg, http.StatusUnauthorized)
		return
	}

	var chosen *repository.PortalMemberChoice
	if req.EegID != "" {
		eegID, err := uuid.Parse(req.EegID)
		if err != nil {
			jsonError(w, invalidMsg, http.StatusUnauthorized)
			return
		}
		for i := range choices {
			if choices[i].EegID == eegID {
				chosen = &choices[i]
				break
			}
		}
		if chosen == nil {
			jsonError(w, invalidMsg, http.StatusUnauthorized)
			return
		}
	} else if len(choices) == 1 {
		chosen = &choices[0]
	} else {
		type choiceItem struct {
			EegID   string `json:"eeg_id"`
			EegName string `json:"eeg_name"`
		}
		items := make([]choiceItem, len(choices))
		for i, c := range choices {
			items[i] = choiceItem{EegID: c.EegID.String(), EegName: c.EegName}
		}
		jsonOK(w, struct {
			OK      bool         `json:"ok"`
			Choices []choiceItem `json:"choices"`
		}{OK: true, Choices: items})
		return
	}

	sessionToken, err := h.portalRepo.CreateSessionForMember(r.Context(), chosen.MemberID, chosen.EegID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_token": sessionToken,
		"member_id":     chosen.MemberID.String(),
		"eeg_id":        chosen.EegID.String(),
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
	hasPassword, err := h.portalRepo.HasPassword(r.Context(), member.Email)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"member":       member,
		"eeg":          eeg,
		"has_password": hasPassword,
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

// GetReferral handles GET /api/v1/public/portal/referral
//
// Returns the member's personal "Mitglieder werben Mitglieder" invite link
// (lazily generating a referral code on first use) plus the EEG's current
// referral bonus amount, for display in the portal's "Mitglieder werben" section.
func (h *MemberPortalHandler) GetReferral(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	code, err := h.memberRepo.GetOrCreateReferralCode(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get/create referral code", "error", err, "member_id", memberID)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]any{
		"code":      code,
		"share_url": fmt.Sprintf("%s/onboarding/%s?ref=%s", eeg.PortalBaseURL, eegID, code),
		"bonus_eur": eeg.ReferralBonusEur,
	})
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

	// Scoped to THIS EEG — an unscoped lookup could resolve to a different tenant's
	// active row for the same Zählpunkt string.
	var mpID *uuid.UUID
	if mp, err := h.meterPointRepo.GetLatestByZaehlpunktInEEG(r.Context(), eegID, req.Zaehlpunkt); err == nil {
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

// ChangeSepaMandate handles POST /api/v1/public/portal/sepa-mandate
// Body: {"iban": "AT...", "confirm": true}
// Archives the member's current SEPA mandate snapshot (old IBAN + prior signature) into
// member_sepa_mandate_history, then stores a freshly signed mandate for the new IBAN —
// the confirming click is the member's own re-consent, captured with IP + timestamp
// just like the onboarding mandate.
func (h *MemberPortalHandler) ChangeSepaMandate(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		IBAN    string `json:"iban"`
		Confirm bool   `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !req.Confirm {
		jsonError(w, "Bestätigung des SEPA-Mandats ist erforderlich", http.StatusBadRequest)
		return
	}
	if !validIBAN(req.IBAN) {
		jsonError(w, "invalid IBAN (only AT/DE IBANs accepted)", http.StatusBadRequest)
		return
	}
	newIBAN := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(req.IBAN), " ", ""))

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}
	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	signedAt := time.Now().UTC()

	mandateText := eeg.OnboardingContractText
	mandateText = strings.ReplaceAll(mandateText, "{iban}", newIBAN)
	mandateText = strings.ReplaceAll(mandateText, "{datum}", signedAt.Format("02.01.2006"))

	if err := h.memberRepo.SetMandate(r.Context(), member, newIBAN, signedAt, ip, mandateText); err != nil {
		slog.Error("failed to set new SEPA mandate", "error", err, "member_id", memberID)
		jsonError(w, "failed to update mandate", http.StatusInternalServerError)
		return
	}

	go h.sendMandateChangedEmail(eegID, member, member.IBAN, newIBAN)

	jsonOK(w, map[string]string{"iban": newIBAN, "signed_at": signedAt.Format(time.RFC3339)})
}

// SetPassword handles POST /api/v1/public/portal/set-password
// Requires a valid portal session — which can only have been obtained via a prior
// magic-link login — so no separate "current password" check is needed; the session
// itself proves identity. Sets or replaces the member's portal password. Keyed by
// email (not member_id): applies to every EEG membership sharing that email, not just
// the one used to authenticate this request.
func (h *MemberPortalHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "Das Passwort muss mindestens 8 Zeichen lang sein.", http.StatusBadRequest)
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}
	if err := h.portalRepo.SetPassword(r.Context(), member.Email, hash); err != nil {
		jsonError(w, "failed to set password", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]bool{"ok": true})
}

// RequestEmailChange handles POST /api/v1/public/portal/email-change
// Body: {"email": "new@example.com"}
// Sends a verification link to the NEW address; the change is only applied once the
// member clicks that link (ConfirmEmailChange). The old email is not touched or archived
// until then, so a mistyped address can never lock the member out.
func (h *MemberPortalHandler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	memberID, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(req.Email))
	if err != nil {
		jsonError(w, "Bitte geben Sie eine gültige E-Mail-Adresse ein.", http.StatusBadRequest)
		return
	}
	newEmail := addr.Address

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}
	if strings.EqualFold(newEmail, member.Email) {
		jsonError(w, "Das ist bereits Ihre aktuelle E-Mail-Adresse.", http.StatusBadRequest)
		return
	}
	exists, err := h.memberRepo.ExistsByEmailInEEG(r.Context(), eegID, newEmail)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if exists {
		jsonError(w, "Diese E-Mail-Adresse wird bereits verwendet.", http.StatusConflict)
		return
	}

	token, err := h.portalRepo.CreateEmailChangeVerification(r.Context(), memberID, eegID, newEmail)
	if err != nil {
		slog.Error("failed to create email change verification", "error", err, "member_id", memberID)
		jsonError(w, "failed to start email change", http.StatusInternalServerError)
		return
	}

	if eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID); err == nil && eeg != nil && !eeg.IsDemo {
		confirmLink := fmt.Sprintf("%s/portal/email-change/confirm?token=%s", eeg.PortalBaseURL, token)
		go h.sendEmailChangeVerificationLink(eegID, newEmail, member, confirmLink)
	}

	jsonOK(w, map[string]string{"new_email": newEmail})
}

// ConfirmEmailChange handles POST /api/v1/public/portal/email-change/confirm/{token}
// Public — no portal session required, since the member may click the link on a
// different device than the one they requested the change from. Idempotent: confirming
// an already-verified token returns success again instead of erroring.
func (h *MemberPortalHandler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}

	v, err := h.portalRepo.FindEmailChangeVerificationByToken(r.Context(), token)
	if err != nil {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}
	if time.Now().After(v.ExpiresAt) {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}
	if v.VerifiedAt != nil {
		jsonOK(w, map[string]any{"ok": true, "email": v.NewEmail})
		return
	}

	exists, err := h.memberRepo.ExistsByEmailInEEG(r.Context(), v.EegID, v.NewEmail)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if exists {
		jsonError(w, "Diese E-Mail-Adresse wird bereits verwendet.", http.StatusConflict)
		return
	}

	oldEmail, err := h.portalRepo.ConfirmEmailChange(r.Context(), v)
	if err != nil {
		slog.Error("failed to confirm email change", "error", err, "member_id", v.MemberID)
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}

	if member, err := h.memberRepo.GetByID(r.Context(), v.MemberID); err == nil && member != nil {
		go h.sendEmailChangedNotifications(v.EegID, member, oldEmail)
	}

	jsonOK(w, map[string]any{"ok": true, "email": v.NewEmail})
}

// sendMandateChangedEmail notifies the member that their SEPA mandate was updated, and
// separately informs the EEG admin (eeg.SMTPFrom) so an IBAN change doesn't go unnoticed —
// same "admin gets a copy" convention used elsewhere for EDA notifications.
func (h *MemberPortalHandler) sendMandateChangedEmail(eegID uuid.UUID, member *domain.Member, oldIBAN, newIBAN string) {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), eegID)
	if err != nil || eeg == nil || eeg.SMTPHost == "" || eeg.IsDemo {
		return
	}
	cfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}

	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	if fullName == "" {
		fullName = "Mitglied"
	}

	// --- Notify the member ---
	memberSubject := "Ihr SEPA-Lastschriftmandat wurde aktualisiert"
	memberBody := fmt.Sprintf(`Sehr geehrte/r %s,

Sie haben soeben über Ihr Mitglieder-Portal ein neues SEPA-Lastschriftmandat mit folgender IBAN bestätigt:

%s

Das bisherige Mandat wurde archiviert. Falls Sie diese Änderung nicht selbst vorgenommen haben, kontaktieren Sie uns bitte umgehend.

Mit freundlichen Grüßen
Ihr EEG-Team`, fullName, newIBAN)

	memberMsg := []byte(mailutil.Headers(eeg.SMTPFrom, member.Email, memberSubject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		memberBody)

	if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "sepa_mandate_changed", member.Email, memberSubject, &member.ID, nil, memberMsg); err != nil {
		slog.Error("sendMandateChangedEmail: failed to send to member", "error", err, "to", member.Email)
	}

	// --- Notify the admin (info only) ---
	if eeg.SMTPFrom == "" || eeg.SMTPFrom == member.Email {
		return
	}
	adminSubject := fmt.Sprintf("IBAN-Änderung: %s hat sein SEPA-Mandat aktualisiert", fullName)
	adminBody := fmt.Sprintf(`Info: Ein Mitglied hat über das Mitglieder-Portal eine neue IBAN bestätigt.

Mitglied: %s (Nr. %s)
Bisherige IBAN: %s
Neue IBAN: %s

Das bisherige Mandat wurde archiviert und ist über die Mandats-Historie auf der Mitgliederseite einsehbar.`,
		fullName, member.MitgliedsNr, orDash(oldIBAN), newIBAN)

	adminMsg := []byte(mailutil.Headers(eeg.SMTPFrom, eeg.SMTPFrom, adminSubject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		adminBody)

	if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "sepa_mandate_changed_admin", eeg.SMTPFrom, adminSubject, &member.ID, nil, adminMsg); err != nil {
		slog.Error("sendMandateChangedEmail: failed to send to admin", "error", err, "to", eeg.SMTPFrom)
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// sendEmailChangeVerificationLink sends the confirmation link to the NEW address only.
// The change is not applied to members.email until this link is clicked.
func (h *MemberPortalHandler) sendEmailChangeVerificationLink(eegID uuid.UUID, newEmail string, member *domain.Member, confirmLink string) {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), eegID)
	if err != nil || eeg == nil || eeg.SMTPHost == "" || eeg.IsDemo {
		return
	}
	cfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}

	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	if fullName == "" {
		fullName = "Mitglied"
	}

	subject := "Bitte bestätigen Sie Ihre neue E-Mail-Adresse"
	body := fmt.Sprintf(`Sehr geehrte/r %s,

Sie haben über Ihr Mitglieder-Portal beantragt, Ihre E-Mail-Adresse auf diese Adresse zu ändern.

Bitte bestätigen Sie die Änderung über folgenden Link:

%s

Der Link ist 30 Minuten gültig. Erst nach Bestätigung wird die neue Adresse übernommen.

Falls Sie diese Änderung nicht selbst veranlasst haben, ignorieren Sie diese E-Mail einfach — es wird nichts geändert.

Mit freundlichen Grüßen
Ihr EEG-Team`, fullName, confirmLink)

	msg := []byte(mailutil.Headers(eeg.SMTPFrom, newEmail, subject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body)

	if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "portal_email_change_verify", newEmail, subject, &member.ID, nil, msg); err != nil {
		slog.Error("sendEmailChangeVerificationLink: failed to send", "error", err, "to", newEmail)
	}
}

// sendEmailChangedNotifications fires after the new email has been applied: confirms to
// the new address, sends a security notice to the OLD address (in case of account
// takeover), and informs the admin — same "admin gets a copy" convention used for IBAN
// changes.
func (h *MemberPortalHandler) sendEmailChangedNotifications(eegID uuid.UUID, member *domain.Member, oldEmail string) {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), eegID)
	if err != nil || eeg == nil || eeg.SMTPHost == "" || eeg.IsDemo {
		return
	}
	cfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}

	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	if fullName == "" {
		fullName = "Mitglied"
	}

	// --- Confirmation to the NEW address ---
	newSubject := "Ihre E-Mail-Adresse wurde geändert"
	newBody := fmt.Sprintf(`Sehr geehrte/r %s,

Ihre E-Mail-Adresse für das Mitglieder-Portal wurde erfolgreich auf diese Adresse geändert.

Mit freundlichen Grüßen
Ihr EEG-Team`, fullName)
	newMsg := []byte(mailutil.Headers(eeg.SMTPFrom, member.Email, newSubject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		newBody)
	if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "portal_email_change_done", member.Email, newSubject, &member.ID, nil, newMsg); err != nil {
		slog.Error("sendEmailChangedNotifications: failed to send to new address", "error", err, "to", member.Email)
	}

	// --- Security notice to the OLD address ---
	if oldEmail != "" && !strings.EqualFold(oldEmail, member.Email) {
		oldSubject := "Ihre E-Mail-Adresse wurde geändert"
		oldBody := fmt.Sprintf(`Sehr geehrte/r %s,

die für Ihr Mitglieder-Portal hinterlegte E-Mail-Adresse wurde soeben von dieser Adresse auf %s geändert.

Falls Sie diese Änderung nicht selbst vorgenommen haben, kontaktieren Sie uns bitte umgehend, damit wir Ihr Konto absichern können.

Mit freundlichen Grüßen
Ihr EEG-Team`, fullName, member.Email)
		oldMsg := []byte(mailutil.Headers(eeg.SMTPFrom, oldEmail, oldSubject) +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			oldBody)
		if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "portal_email_change_security", oldEmail, oldSubject, &member.ID, nil, oldMsg); err != nil {
			slog.Error("sendEmailChangedNotifications: failed to send to old address", "error", err, "to", oldEmail)
		}
	}

	// --- Admin copy (info only) ---
	if eeg.SMTPFrom == "" || strings.EqualFold(eeg.SMTPFrom, member.Email) {
		return
	}
	adminSubject := fmt.Sprintf("E-Mail-Änderung: %s hat seine E-Mail-Adresse geändert", fullName)
	adminBody := fmt.Sprintf(`Info: Ein Mitglied hat über das Mitglieder-Portal seine E-Mail-Adresse geändert.

Mitglied: %s (Nr. %s)
Bisherige E-Mail: %s
Neue E-Mail: %s`, fullName, member.MitgliedsNr, orDash(oldEmail), member.Email)
	adminMsg := []byte(mailutil.Headers(eeg.SMTPFrom, eeg.SMTPFrom, adminSubject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		adminBody)
	if err := invoice.SendLogged(context.Background(), h.emailLogRepo, cfg, eeg.ID, "portal_email_change_admin", eeg.SMTPFrom, adminSubject, &member.ID, nil, adminMsg); err != nil {
		slog.Error("sendEmailChangedNotifications: failed to send to admin", "error", err, "to", eeg.SMTPFrom)
	}
}
