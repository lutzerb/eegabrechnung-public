package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/smtp"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/invoice"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/netzbetreiber"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// OnboardingHandler handles public and admin onboarding endpoints.
type OnboardingHandler struct {
	onboardingRepo *repository.OnboardingRepository
	eegRepo        *repository.EEGRepository
	memberRepo     *repository.MemberRepository
	meterPointRepo *repository.MeterPointRepository
	docRepo        *repository.EEGDocumentRepository
	db             *pgxpool.Pool
	webBaseURL     string
}

// NewOnboardingHandler creates an OnboardingHandler.
func NewOnboardingHandler(
	onboardingRepo *repository.OnboardingRepository,
	eegRepo *repository.EEGRepository,
	memberRepo *repository.MemberRepository,
	meterPointRepo *repository.MeterPointRepository,
	docRepo *repository.EEGDocumentRepository,
	db *pgxpool.Pool,
	webBaseURL string,
) *OnboardingHandler {
	return &OnboardingHandler{
		onboardingRepo: onboardingRepo,
		eegRepo:        eegRepo,
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		docRepo:        docRepo,
		db:             db,
		webBaseURL:     webBaseURL,
	}
}

// publicDocumentItem is a document reference returned to the public onboarding page.
type publicDocumentItem struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Filename string    `json:"filename"`
	MimeType string    `json:"mime_type"`
}

// publicEEGInfo is the limited EEG view returned for the public onboarding page.
type publicEEGInfo struct {
	ID                     uuid.UUID            `json:"id"`
	Name                   string               `json:"name"`
	BillingPeriod          string               `json:"billing_period"`
	OnboardingContractText string               `json:"onboarding_contract_text"`
	Documents              []publicDocumentItem `json:"documents"`
}

// GetPublicEEGInfo handles GET /api/v1/public/eegs/{eegID}/info
//
//	@Summary		Get public EEG info
//	@Description	Returns limited EEG details for display on the public onboarding page (name, billing period, contract text).
//	@Tags			Onboarding
//	@Produce		json
//	@Param			eegID	path		string			true	"EEG UUID"
//	@Success		200		{object}	publicEEGInfo
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/public/eegs/{eegID}/info [get]
func (h *OnboardingHandler) GetPublicEEGInfo(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	docs, err := h.docRepo.ListForOnboarding(r.Context(), eeg.ID)
	if err != nil {
		docs = nil
	}
	pubDocs := make([]publicDocumentItem, 0, len(docs))
	for _, d := range docs {
		pubDocs = append(pubDocs, publicDocumentItem{
			ID:       d.ID,
			Title:    d.Title,
			Filename: d.Filename,
			MimeType: d.MimeType,
		})
	}

	jsonOK(w, publicEEGInfo{
		ID:                     eeg.ID,
		Name:                   eeg.Name,
		BillingPeriod:          eeg.BillingPeriod,
		OnboardingContractText: eeg.OnboardingContractText,
		Documents:              pubDocs,
	})
}

// onboardingSubmitRequest is the body for POST /api/v1/public/eegs/{eegID}/onboarding.
type onboardingSubmitRequest struct {
	Name1            string                        `json:"name1"`
	Name2            string                        `json:"name2"`
	Email            string                        `json:"email"`
	Phone            string                        `json:"phone"`
	Strasse          string                        `json:"strasse"`
	PLZ              string                        `json:"plz"`
	Ort              string                        `json:"ort"`
	IBAN             string                        `json:"iban"`
	BIC              string                        `json:"bic"`
	MemberType       string                        `json:"member_type"`
	BusinessRole     string                        `json:"business_role"`  // privat | kleinunternehmer | ...
	UidNummer        string                        `json:"uid_nummer"`
	UseVat           bool                          `json:"use_vat"`
	MeterPoints      []domain.OnboardingMeterPoint `json:"meter_points"`
	BeitrittsDatum   string                        `json:"beitritts_datum"` // optional YYYY-MM-DD
	ContractAccepted bool                          `json:"contract_accepted"`
}

// SubmitOnboarding handles POST /api/v1/public/eegs/{eegID}/onboarding
//
//	@Summary		Submit membership application
//	@Description	Creates a new onboarding request (pending status). Sends a magic-token status link to the applicant and notifies EEG administrators. Always requires contract_accepted=true and at least one meter point.
//	@Tags			Onboarding
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string					true	"EEG UUID"
//	@Param			body	body		onboardingSubmitRequest	true	"Membership application"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/public/eegs/{eegID}/onboarding [post]
func (h *OnboardingHandler) SubmitOnboarding(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	var body onboardingSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(body.Name1) == "" {
		jsonError(w, "name1 is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Email) == "" {
		jsonError(w, "email is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Strasse) == "" {
		jsonError(w, "strasse is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.PLZ) == "" {
		jsonError(w, "plz is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Ort) == "" {
		jsonError(w, "ort is required", http.StatusBadRequest)
		return
	}
	if !validIBAN(body.IBAN) {
		jsonError(w, "invalid IBAN (only AT/DE IBANs accepted)", http.StatusBadRequest)
		return
	}
	hasMP := false
	for _, mp := range body.MeterPoints {
		if strings.TrimSpace(mp.Zaehlpunkt) != "" {
			hasMP = true
			break
		}
	}
	if !hasMP {
		jsonError(w, "at least one meter point is required", http.StatusBadRequest)
		return
	}
	if !body.ContractAccepted {
		jsonError(w, "contract must be accepted", http.StatusBadRequest)
		return
	}

	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	// Determine client IP
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	memberType := body.MemberType
	if memberType == "" {
		memberType = "CONSUMER"
	}

	// Parse optional beitritts_datum
	var beitrittsDatum *time.Time
	if body.BeitrittsDatum != "" {
		t, err := time.Parse("2006-01-02", body.BeitrittsDatum)
		if err != nil {
			jsonError(w, "invalid beitritts_datum: use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		beitrittsDatum = &t
	}

	businessRole := strings.TrimSpace(body.BusinessRole)
	if businessRole == "" {
		businessRole = "privat"
	}

	now := time.Now()
	req := &domain.OnboardingRequest{
		EegID:              eegID,
		Status:             "pending",
		Name1:              strings.TrimSpace(body.Name1),
		Name2:              strings.TrimSpace(body.Name2),
		Email:              strings.TrimSpace(body.Email),
		Phone:              strings.TrimSpace(body.Phone),
		Strasse:            strings.TrimSpace(body.Strasse),
		PLZ:                strings.TrimSpace(body.PLZ),
		Ort:                strings.TrimSpace(body.Ort),
		IBAN:               strings.TrimSpace(body.IBAN),
		BIC:                strings.TrimSpace(body.BIC),
		MemberType:         memberType,
		BusinessRole:       businessRole,
		UidNummer:          strings.TrimSpace(body.UidNummer),
		UseVat:             body.UseVat,
		MeterPoints:        body.MeterPoints,
		BeitrittsDatum:     beitrittsDatum,
		ContractAcceptedAt: &now,
		ContractIP:         ip,
	}

	if req.MeterPoints == nil {
		req.MeterPoints = []domain.OnboardingMeterPoint{}
	}

	// Check: email already registered as active member
	if emailExists, err := h.memberRepo.ExistsByEmailInEEG(r.Context(), eegID, req.Email); err == nil && emailExists {
		jsonError(w, "Diese E-Mail-Adresse ist bereits als Mitglied registriert.", http.StatusConflict)
		return
	}

	// Check: open onboarding request with same email already exists
	if pending, err := h.onboardingRepo.FindPendingByEmailAndEEG(r.Context(), req.Email, eegID); err == nil && pending != nil {
		jsonError(w, "Es liegt bereits ein offener Beitrittsantrag für diese E-Mail-Adresse vor. Bitte prüfen Sie Ihre E-Mails.", http.StatusConflict)
		return
	}

	// Check: meter points already registered in this EEG
	for _, mp := range req.MeterPoints {
		if strings.TrimSpace(mp.Zaehlpunkt) == "" {
			continue
		}
		if exists, err := h.meterPointRepo.ExistsByZaehlpunktInEEG(r.Context(), eegID, mp.Zaehlpunkt); err == nil && exists {
			jsonError(w, "Der Zählpunkt "+mp.Zaehlpunkt+" ist bereits in dieser Energiegemeinschaft registriert.", http.StatusConflict)
			return
		}
	}

	if err := h.onboardingRepo.Create(r.Context(), req); err != nil {
		slog.Error("failed to create onboarding request", "error", err)
		jsonError(w, "failed to submit onboarding request", http.StatusInternalServerError)
		return
	}

	if !eeg.IsDemo {
		// Send magic token email to applicant (non-fatal on error)
		if err := h.sendMagicTokenEmail(req, eeg.Name); err != nil {
			slog.Warn("failed to send onboarding magic token email", "error", err, "request_id", req.ID)
		}

		// Notify all EEG admins/assigned users (non-fatal on error)
		if err := h.sendAdminNotificationEmail(r.Context(), eegID, req, eeg.Name); err != nil {
			slog.Warn("failed to send admin notification email", "error", err, "request_id", req.ID)
		}
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]any{
		"id":      req.ID,
		"status":  req.Status,
		"message": "Antrag eingereicht. Bitte prüfen Sie Ihre E-Mail für den Status-Link.",
	})
}

// GetOnboardingStatus handles GET /api/v1/public/onboarding/status/{token}
//
//	@Summary		Get onboarding request status by magic token
//	@Description	Returns the onboarding request corresponding to the given magic token. Returns 410 Gone if the token has expired.
//	@Tags			Onboarding
//	@Produce		json
//	@Param			token	path		string	true	"Magic token from status email"
//	@Success		200		{object}	domain.OnboardingRequest
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		410		{object}	map[string]string	"Token expired"
//	@Failure		500		{object}	map[string]string
//	@Router			/public/onboarding/status/{token} [get]
func (h *OnboardingHandler) GetOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		jsonError(w, "token is required", http.StatusBadRequest)
		return
	}

	req, err := h.onboardingRepo.GetByToken(r.Context(), token)
	if err != nil {
		if strings.Contains(err.Error(), "token expired") {
			jsonError(w, "token expired", http.StatusGone)
			return
		}
		slog.Error("failed to get onboarding by token", "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if req == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	jsonOK(w, req)
}

// resendTokenRequest is the body for POST /api/v1/public/onboarding/resend-token.
type resendTokenRequest struct {
	Email string `json:"email"`
	EegID string `json:"eeg_id"`
}

// ResendToken handles POST /api/v1/public/onboarding/resend-token
//
//	@Summary		Resend onboarding status magic token
//	@Description	Looks up a pending onboarding request by email + EEG ID, issues a fresh magic token, and resends the status link email. Always returns HTTP 200 to avoid email enumeration.
//	@Tags			Onboarding
//	@Accept			json
//	@Produce		json
//	@Param			body	body		resendTokenRequest	true	"Email and EEG ID"
//	@Success		200		{object}	map[string]string
//	@Router			/public/onboarding/resend-token [post]
func (h *OnboardingHandler) ResendToken(w http.ResponseWriter, r *http.Request) {
	var body resendTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Always return 200 to avoid email enumeration
		jsonOK(w, map[string]string{"message": "ok"})
		return
	}

	eegID, err := uuid.Parse(body.EegID)
	if err != nil {
		jsonOK(w, map[string]string{"message": "ok"})
		return
	}

	req, err := h.onboardingRepo.FindPendingByEmailAndEEG(r.Context(), strings.TrimSpace(body.Email), eegID)
	if err != nil || req == nil {
		// Don't leak whether the email exists
		jsonOK(w, map[string]string{"message": "ok"})
		return
	}

	newToken, err := h.onboardingRepo.UpdateToken(r.Context(), req.ID)
	if err != nil {
		slog.Error("failed to update token for resend", "error", err)
		jsonOK(w, map[string]string{"message": "ok"})
		return
	}
	req.MagicToken = newToken

	eeg, err := h.eegRepo.GetByIDInternal(r.Context(), eegID)
	if err != nil {
		jsonOK(w, map[string]string{"message": "ok"})
		return
	}

	if err := h.sendMagicTokenEmail(req, eeg.Name); err != nil {
		slog.Warn("failed to resend magic token email", "error", err, "request_id", req.ID)
	}

	jsonOK(w, map[string]string{"message": "ok"})
}

// ListOnboarding handles GET /api/v1/eegs/{eegID}/onboarding (auth required)
//
//	@Summary		List onboarding requests for an EEG
//	@Description	Returns all onboarding requests (pending, approved, rejected, converted) for the given EEG.
//	@Tags			Onboarding
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Success		200		{array}		domain.OnboardingRequest
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/onboarding [get]
func (h *OnboardingHandler) ListOnboarding(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	reqs, err := h.onboardingRepo.ListByEEG(r.Context(), eegID)
	if err != nil {
		slog.Error("failed to list onboarding requests", "error", err)
		jsonError(w, "failed to list onboarding requests", http.StatusInternalServerError)
		return
	}

	if reqs == nil {
		reqs = []domain.OnboardingRequest{}
	}
	jsonOK(w, reqs)
}

// updateStatusRequest is the body for PATCH /api/v1/eegs/{eegID}/onboarding/{id}.
// When Status is empty, all other fields are treated as a data-field update (no status change).
type updateStatusRequest struct {
	Status          string `json:"status"`
	AdminNotes      string `json:"admin_notes"`
	NetzbetreiberID string `json:"netzbetreiber_id"` // optional override for conversion email
	CustomMessage   string `json:"custom_message"`   // optional custom email body paragraph
	// Field-update fields (used when Status is "")
	Name1          string                        `json:"name1"`
	Name2          string                        `json:"name2"`
	Email          string                        `json:"email"`
	Phone          string                        `json:"phone"`
	Strasse        string                        `json:"strasse"`
	PLZ            string                        `json:"plz"`
	Ort            string                        `json:"ort"`
	IBAN           string                        `json:"iban"`
	BIC            string                        `json:"bic"`
	MemberType     string                        `json:"member_type"`
	BusinessRole   string                        `json:"business_role"`
	UidNummer      string                        `json:"uid_nummer"`
	UseVat         bool                          `json:"use_vat"`
	MeterPoints    []domain.OnboardingMeterPoint `json:"meter_points"`
	BeitrittsDatum string                        `json:"beitritts_datum"` // YYYY-MM-DD or ""
}

// UpdateOnboardingStatus handles PATCH /api/v1/eegs/{eegID}/onboarding/{id} (auth required).
// When status='approved', it triggers the full member creation + meter point creation + EDA Anmeldung,
// then sets the onboarding status to 'converted'.
// When status='rejected', it simply records the rejection with notes.
//
//	@Summary		Approve or reject an onboarding request
//	@Description	Updates the status of an onboarding request. When status is 'approved', atomically creates the member, meter points, and EDA Anmeldung processes, then marks the request as 'converted'. When status is 'rejected', records the rejection with optional admin notes.
//	@Tags			Onboarding
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG UUID"
//	@Param			id		path		string				true	"Onboarding request UUID"
//	@Param			body	body		updateStatusRequest	true	"Status update (approved/rejected/pending) with optional admin_notes"
//	@Success		200		{object}	domain.OnboardingRequest
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string	"Already converted"
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/onboarding/{id} [patch]
func (h *OnboardingHandler) UpdateOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid request ID", http.StatusBadRequest)
		return
	}

	var body updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Field-only update (no status change): when Status is empty.
	if body.Status == "" {
		req, err := h.onboardingRepo.GetByID(ctx, id)
		if err != nil || req == nil {
			jsonError(w, "onboarding request not found", http.StatusNotFound)
			return
		}
		if req.Status == "converted" || req.Status == "active" {
			jsonError(w, "cannot edit a converted or active onboarding request", http.StatusConflict)
			return
		}
		req.Name1 = strings.TrimSpace(body.Name1)
		req.Name2 = strings.TrimSpace(body.Name2)
		req.Email = strings.TrimSpace(body.Email)
		req.Phone = strings.TrimSpace(body.Phone)
		req.Strasse = strings.TrimSpace(body.Strasse)
		req.PLZ = strings.TrimSpace(body.PLZ)
		req.Ort = strings.TrimSpace(body.Ort)
		req.IBAN = strings.TrimSpace(body.IBAN)
		req.BIC = strings.TrimSpace(body.BIC)
		if body.MemberType != "" {
			req.MemberType = body.MemberType
		}
		if body.BusinessRole != "" {
			req.BusinessRole = body.BusinessRole
		}
		req.UidNummer = strings.TrimSpace(body.UidNummer)
		req.UseVat = body.UseVat
		if body.MeterPoints != nil {
			req.MeterPoints = body.MeterPoints
		}
		req.AdminNotes = body.AdminNotes
		if body.BeitrittsDatum != "" {
			t, err := time.Parse("2006-01-02", body.BeitrittsDatum)
			if err != nil {
				jsonError(w, "invalid beitritts_datum: use YYYY-MM-DD", http.StatusBadRequest)
				return
			}
			req.BeitrittsDatum = &t
		} else {
			req.BeitrittsDatum = nil
		}
		if err := h.onboardingRepo.UpdateFields(ctx, req); err != nil {
			slog.Error("failed to update onboarding fields", "error", err)
			jsonError(w, "failed to update onboarding request", http.StatusInternalServerError)
			return
		}
		updated, _ := h.onboardingRepo.GetByID(ctx, id)
		jsonOK(w, updated)
		return
	}

	validStatuses := map[string]bool{"pending": true, "approved": true, "rejected": true, "eda_sent": true, "active": true}
	if !validStatuses[body.Status] {
		jsonError(w, "invalid status; must be pending, approved, rejected, eda_sent or active", http.StatusBadRequest)
		return
	}

	// For 'approved': do full member creation atomically, then set status to 'eda_sent'
	if body.Status == "approved" {
		req, err := h.onboardingRepo.GetByID(ctx, id)
		if err != nil || req == nil {
			jsonError(w, "onboarding request not found", http.StatusNotFound)
			return
		}
		if req.Status == "converted" || req.Status == "eda_sent" || req.Status == "active" {
			jsonError(w, "already converted", http.StatusConflict)
			return
		}

		// Determine beitritts_datum: frühestens morgen, höchstens 30 Tage in der Zukunft.
		// Falls kein Datum gespeichert oder Datum in der Vergangenheit/heute → morgen.
		// Falls Datum mehr als 30 Tage in der Zukunft → auf max kürzen.
		viennaLoc, _ := time.LoadLocation("Europe/Vienna")
		tomorrowVienna := time.Now().In(viennaLoc).AddDate(0, 0, 1)
		tomorrow := time.Date(tomorrowVienna.Year(), tomorrowVienna.Month(), tomorrowVienna.Day(), 0, 0, 0, 0, time.UTC)
		maxDate := tomorrow.AddDate(0, 0, 30)
		var beitrittsDatum *time.Time
		if req.BeitrittsDatum == nil || req.BeitrittsDatum.Before(tomorrow) {
			d := tomorrow
			beitrittsDatum = &d
		} else if req.BeitrittsDatum.After(maxDate) {
			d := maxDate
			beitrittsDatum = &d
		} else {
			beitrittsDatum = req.BeitrittsDatum
		}

		// Generate member number (outside transaction for simplicity)
		mitgliedsNr, err := h.memberRepo.NextMemberNumber(ctx, eegID)
		if err != nil {
			slog.Error("failed to get next member number", "error", err)
			jsonError(w, "failed to generate member number", http.StatusInternalServerError)
			return
		}

		// Get EEG to check EDA configuration
		eeg, err := h.eegRepo.GetByIDInternal(ctx, eegID)
		if err != nil {
			slog.Error("failed to get EEG", "error", err)
			jsonError(w, "EEG not found", http.StatusNotFound)
			return
		}

		// Begin transaction
		tx, err := h.db.Begin(ctx)
		if err != nil {
			slog.Error("failed to begin transaction", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		// Insert member
		var memberID uuid.UUID
		var memberCreatedAt time.Time
		businessRole := req.BusinessRole
		if businessRole == "" {
			businessRole = "privat"
		}
		var useVatPtr *bool
		if req.UseVat {
			t := true
			useVatPtr = &t
		}
		// Build mandate text: fill {iban} and {datum} placeholders in the EEG contract template
		mandateText := eeg.OnboardingContractText
		if beitrittsDatum != nil {
			mandateText = strings.ReplaceAll(mandateText, "{datum}", beitrittsDatum.Format("02.01.2006"))
		}
		mandateText = strings.ReplaceAll(mandateText, "{iban}", req.IBAN)

		memberQ := `INSERT INTO members (eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort,
		                                business_role, uid_nummer, use_vat, vat_pct, status, beitritt_datum,
		                                sepa_mandate_signed_at, sepa_mandate_signed_ip, sepa_mandate_text)
		            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		            RETURNING id, created_at`
		err = tx.QueryRow(ctx, memberQ,
			eegID, mitgliedsNr, req.Name1, req.Name2, req.Email, req.IBAN,
			req.Strasse, req.PLZ, req.Ort, businessRole,
			req.UidNummer, useVatPtr, nil, "REGISTERED", beitrittsDatum,
			req.ContractAcceptedAt, req.ContractIP, mandateText,
		).Scan(&memberID, &memberCreatedAt)
		if err != nil {
			slog.Error("failed to insert member", "error", err)
			jsonError(w, "failed to create member", http.StatusInternalServerError)
			return
		}

		// Insert meter points and collect IDs for EDA
		type meterPointCreated struct {
			id                  uuid.UUID
			zaehlpunkt          string
			participationFactor float64
		}
		var createdMPs []meterPointCreated

		for _, mp := range req.MeterPoints {
			if strings.TrimSpace(mp.Zaehlpunkt) == "" {
				continue
			}
			direction := mp.Direction
			if direction == "" {
				direction = "CONSUMPTION"
			}
			var mpID uuid.UUID
			var genType *string
			if direction == "GENERATION" && mp.GenerationType != "" {
				gt := mp.GenerationType
				genType = &gt
			}
			mpQ := `INSERT INTO meter_points (member_id, eeg_id, zaehlpunkt, energierichtung, verteilungsmodell,
			                                  zugeteilte_menge_pct, status, registriert_seit, generation_type)
			        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			        RETURNING id`
			err := tx.QueryRow(ctx, mpQ,
				memberID, eegID, mp.Zaehlpunkt, direction, "DYNAMIC", float64(100), "ACTIVE", beitrittsDatum, genType,
			).Scan(&mpID)
			if err != nil {
				slog.Error("failed to insert meter point", "error", err, "zaehlpunkt", mp.Zaehlpunkt)
				jsonError(w, fmt.Sprintf("failed to create meter point %s: %v", mp.Zaehlpunkt, err), http.StatusInternalServerError)
				return
			}
			factor := mp.ParticipationFactor
			if factor <= 0 || factor > 100 {
				factor = 100.0
			}
			createdMPs = append(createdMPs, meterPointCreated{id: mpID, zaehlpunkt: mp.Zaehlpunkt, participationFactor: factor})
		}

		// Mark onboarding as eda_sent
		convQ := `UPDATE onboarding_requests
		          SET status = 'eda_sent', converted_member_id = $1, admin_notes = $2, updated_at = now()
		          WHERE id = $3`
		if _, err := tx.Exec(ctx, convQ, memberID, body.AdminNotes, id); err != nil {
			slog.Error("failed to set eda_sent", "error", err)
			jsonError(w, "failed to mark request as eda_sent", http.StatusInternalServerError)
			return
		}

		// Trigger EDA Anmeldung for each meter point if EEG has EDA configured
		if eeg.EdaMarktpartnerID != "" {
			for _, cmp := range createdMPs {
				conversationID := uuid.New().String()
				msgID := uuid.NewString()

				direction := "CONSUMPTION"
				for _, mp := range req.MeterPoints {
					if mp.Zaehlpunkt == cmp.zaehlpunkt && mp.Direction == "GENERATION" {
						direction = "GENERATION"
					}
				}

				// Derive the Netzbetreiber-ID from the Zählpunkt prefix (first 8 chars = AT + 6-digit code).
				// This is more reliable than the EEG-wide setting which may belong to a different service area.
				netzbetreiberTo := eeg.EdaNetzbetreiberID
				if len(cmp.zaehlpunkt) >= 8 {
					netzbetreiberTo = cmp.zaehlpunkt[:8]
				}

				xmlBody, xmlErr := edaxml.BuildCMRequest(edaxml.CMRequestParams{
					From:            eeg.EdaMarktpartnerID,
					To:              netzbetreiberTo,
					MessageID:       msgID,
					ConversationID:  conversationID,
					CMRequestID:     uuid.NewString(),
					MeteringPoint:   cmp.zaehlpunkt,
					ECID:            eeg.GemeinschaftID,
					DateFrom:        func() time.Time { if beitrittsDatum != nil { return *beitrittsDatum }; return time.Now() }(),
					ECPartFact:      cmp.participationFactor,
					EnergyDirection: direction,
				})
				if xmlErr != nil {
					slog.Error("failed to build CMRequest XML", "error", xmlErr, "zaehlpunkt", cmp.zaehlpunkt)
					continue
				}

				var processID uuid.UUID
				edaQ := `INSERT INTO eda_processes (eeg_id, meter_point_id, process_type, status, conversation_id,
				                                    zaehlpunkt, valid_from, participation_factor, share_type,
				                                    initiated_at, deadline_at)
				          VALUES ($1, $2, 'EC_REQ_ONL', 'pending', $3, $4, $5, $6, $7, now(), now() + interval '60 days')
				          RETURNING id`
				err := tx.QueryRow(ctx, edaQ,
					eegID, cmp.id, conversationID, cmp.zaehlpunkt, beitrittsDatum, cmp.participationFactor, "GC",
				).Scan(&processID)
				if err != nil {
					slog.Error("failed to insert EDA process", "error", err, "zaehlpunkt", cmp.zaehlpunkt)
					continue
				}

				type fullJobPayload struct {
					Process        string    `json:"process"`
					From           string    `json:"from"`
					To             string    `json:"to"`
					GemeinschaftID string    `json:"gemeinschaft_id"`
					ConversationID string    `json:"conversation_id"`
					XMLPayload     string    `json:"xml_payload"`
					EDAProcessID   uuid.UUID `json:"eda_process_id"`
					EegID          uuid.UUID `json:"eeg_id"`
				}
				jobPayload, _ := json.Marshal(fullJobPayload{
					Process:        "EC_REQ_ONL",
					From:           eeg.EdaMarktpartnerID,
					To:             netzbetreiberTo,
					GemeinschaftID: eeg.GemeinschaftID,
					ConversationID: conversationID,
					XMLPayload:     xmlBody,
					EDAProcessID:   processID,
					EegID:          eegID,
				})
				if _, err := tx.Exec(ctx,
					`INSERT INTO jobs (type, payload, status) VALUES ('eda.EC_REQ_ONL', $1, 'pending')`,
					jobPayload,
				); err != nil {
					slog.Error("failed to insert EDA job", "error", err, "process_id", processID)
				}
			}
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("failed to commit approval transaction", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Send conversion email to member (async, non-fatal)
		go func() {
			nbID := body.NetzbetreiberID
			if nbID == "" {
				nbID = eeg.EdaNetzbetreiberID
			}
			nb, _ := netzbetreiber.ByMarktpartnerID(nbID)
			// Use first meter point's zaehlpunkt as fallback lookup
			if nb.Name == "" && len(req.MeterPoints) > 0 {
				nb, _ = netzbetreiber.ByZaehlpunkt(req.MeterPoints[0].Zaehlpunkt)
			}
			if err := h.sendConversionEmail(req, eeg.Name, nb, body.CustomMessage, memberID); err != nil {
				slog.Warn("failed to send conversion email", "request_id", req.ID, "error", err)
			}
		}()

		result, err := h.onboardingRepo.GetByID(ctx, id)
		if err != nil || result == nil {
			jsonOK(w, map[string]any{"id": id, "status": "eda_sent", "converted_member_id": memberID})
			return
		}
		jsonOK(w, result)
		return
	}

	// For 'rejected' or 'pending': simple status update
	if err := h.onboardingRepo.UpdateStatus(ctx, id, body.Status, body.AdminNotes); err != nil {
		slog.Error("failed to update onboarding status", "error", err)
		jsonError(w, "failed to update status", http.StatusInternalServerError)
		return
	}

	req, err := h.onboardingRepo.GetByID(ctx, id)
	if err != nil || req == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, req)
}


// GetOnboardingByID handles GET /api/v1/eegs/{eegID}/onboarding/{id} (auth required).
//
//	@Summary		Get a single onboarding request
//	@Description	Returns a single onboarding request by its UUID, scoped to the given EEG.
//	@Tags			Onboarding
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			id		path		string	true	"Onboarding request UUID"
//	@Success		200		{object}	domain.OnboardingRequest
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/onboarding/{id} [get]
func (h *OnboardingHandler) GetOnboardingByID(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid request ID", http.StatusBadRequest)
		return
	}
	req, err := h.onboardingRepo.GetByID(r.Context(), id)
	if err != nil || req == nil || req.EegID != eegID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, req)
}

// DeleteOnboarding handles DELETE /eegs/{eegID}/onboarding/{id}
func (h *OnboardingHandler) DeleteOnboarding(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid request ID", http.StatusBadRequest)
		return
	}
	if err := h.onboardingRepo.Delete(r.Context(), id, eegID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "not found", http.StatusNotFound)
		} else {
			jsonError(w, "failed to delete onboarding request", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sendAdminNotificationEmail notifies all users with access to the EEG about a new onboarding request.
func (h *OnboardingHandler) sendAdminNotificationEmail(ctx context.Context, eegID uuid.UUID, req *domain.OnboardingRequest, eegName string) error {
	eeg, err := h.eegRepo.GetByIDInternal(ctx, eegID)
	if err != nil || eeg.SMTPHost == "" {
		return nil
	}
	smtpCfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}

	// Query admins (all users in org) + explicitly assigned users for this EEG
	rows, err := h.db.Query(ctx, `
		SELECT DISTINCT u.email, u.name
		FROM users u
		WHERE u.organization_id = (SELECT organization_id FROM eegs WHERE id = $1)
		  AND (
		    u.role = 'admin'
		    OR EXISTS (SELECT 1 FROM user_eeg_assignments a WHERE a.user_id = u.id AND a.eeg_id = $1)
		  )
	`, eegID)
	if err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	type recipient struct{ email, name string }
	var recipients []recipient
	for rows.Next() {
		var r recipient
		if err := rows.Scan(&r.email, &r.name); err != nil {
			continue
		}
		recipients = append(recipients, r)
	}
	if len(recipients) == 0 {
		return nil
	}

	adminURL := fmt.Sprintf("%s/eegs/%s/onboarding/%s", h.webBaseURL, eegID, req.ID)
	subject := fmt.Sprintf("Neuer Beitrittsantrag – %s", eegName)
	fullName := strings.TrimSpace(req.Name1 + " " + req.Name2)

	var auth smtp.Auth
	if smtpCfg.Username != "" {
		host := smtpCfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, host)
	}

	htmlTemplate := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #1e40af;">Neuer Beitrittsantrag eingegangen</h2>
<p>Hallo %s,</p>
<p>für die Energiegemeinschaft <strong>%s</strong> ist ein neuer Beitrittsantrag eingegangen:</p>
<table style="border-collapse: collapse; width: 100%%; margin: 16px 0;">
  <tr><td style="padding: 6px 12px 6px 0; color: #64748b; white-space: nowrap;">Name</td><td style="padding: 6px 0; font-weight: bold;">%s</td></tr>
  <tr><td style="padding: 6px 12px 6px 0; color: #64748b;">E-Mail</td><td style="padding: 6px 0;">%s</td></tr>
  <tr><td style="padding: 6px 12px 6px 0; color: #64748b;">Typ</td><td style="padding: 6px 0;">%s</td></tr>
  <tr><td style="padding: 6px 12px 6px 0; color: #64748b;">Zählpunkte</td><td style="padding: 6px 0;">%d</td></tr>
</table>
<p style="margin: 24px 0;">
  <a href="%s" style="background-color: #1e40af; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">
    Antrag ansehen &amp; bearbeiten
  </a>
</p>
<p style="color: #64748b; font-size: 14px;">Oder öffnen Sie: <a href="%s">%s</a></p>
</body>
</html>`

	memberTypeLabel := map[string]string{
		"CONSUMER": "Verbraucher",
		"PRODUCER": "Erzeuger",
		"PROSUMER": "Prosumer",
	}[req.MemberType]
	if memberTypeLabel == "" {
		memberTypeLabel = req.MemberType
	}

	for _, rcpt := range recipients {
		htmlBody := fmt.Sprintf(htmlTemplate,
			rcpt.name, eegName, fullName, req.Email, memberTypeLabel, len(req.MeterPoints),
			adminURL, adminURL, adminURL,
		)
		var msg strings.Builder
		msg.WriteString("From: " + smtpCfg.From + "\r\n")
		msg.WriteString("To: " + rcpt.email + "\r\n")
		msg.WriteString("Subject: " + subject + "\r\n")
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
		if err := smtp.SendMail(smtpCfg.Host, auth, smtpCfg.From, []string{rcpt.email}, []byte(msg.String())); err != nil {
			slog.Warn("failed to send admin notification to user", "email", rcpt.email, "error", err)
		}
	}
	return nil
}

// VerifyEmail handles POST /api/v1/public/eegs/{eegID}/onboarding/verify-email
// Body: {"email": "...", "name1": "...", "name2": "..."}
// Creates a UUID token in onboarding_email_verifications (expires 30 min) and sends a link.
// Always returns 200 {"ok":true} to avoid email enumeration.
//
//	@Summary		Initiate email address verification
//	@Description	Creates a short-lived (30 min) email-verification token and sends a confirmation link to the applicant. Always returns HTTP 200 to avoid email enumeration.
//	@Tags			Onboarding
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			body	body		object{email=string,name1=string,name2=string}	true	"Applicant details"
//	@Success		200		{object}	map[string]bool
//	@Failure		400		{object}	map[string]string
//	@Router			/public/eegs/{eegID}/onboarding/verify-email [post]
func (h *OnboardingHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	var body struct {
		Email string `json:"email"`
		Name1 string `json:"name1"`
		Name2 string `json:"name2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}

	token := uuid.New().String()
	expiresAt := time.Now().Add(30 * time.Minute)

	_, err = h.db.Exec(r.Context(), `
		INSERT INTO onboarding_email_verifications (eeg_id, email, name1, name2, token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		eegID, email, strings.TrimSpace(body.Name1), strings.TrimSpace(body.Name2), token, expiresAt,
	)
	if err != nil {
		slog.Error("failed to create email verification", "error", err)
		// Still return ok to avoid leaking DB errors
		jsonOK(w, map[string]bool{"ok": true})
		return
	}

	verifyLink := fmt.Sprintf("%s/onboarding/%s?ev=%s", h.webBaseURL, eegID, token)
	if eeg, _ := h.eegRepo.GetByIDInternal(r.Context(), eegID); eeg == nil || !eeg.IsDemo {
		go h.sendEmailVerificationLink(eegID, email, strings.TrimSpace(body.Name1), verifyLink)
	}

	jsonOK(w, map[string]bool{"ok": true})
}

// ConfirmEmailVerification handles POST /api/v1/public/eegs/{eegID}/onboarding/verify/{token}
// Marks the token as verified if valid and not expired.
//
//	@Summary		Confirm email verification token
//	@Description	Validates the email-verification token and marks it as verified. Returns the verified email address and name fields on success. Returns 400 if the token is invalid or expired.
//	@Tags			Onboarding
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			token	path		string	true	"Email verification token"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string	"Invalid or expired token"
//	@Router			/public/eegs/{eegID}/onboarding/verify/{token} [post]
func (h *OnboardingHandler) ConfirmEmailVerification(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	token := chi.URLParam(r, "token")
	if token == "" {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}

	var ev struct {
		ID         uuid.UUID
		Email      string
		Name1      string
		Name2      string
		ExpiresAt  time.Time
		VerifiedAt *time.Time
	}
	err = h.db.QueryRow(r.Context(), `
		SELECT id, email, name1, name2, expires_at, verified_at
		FROM onboarding_email_verifications
		WHERE token = $1 AND eeg_id = $2`,
		token, eegID,
	).Scan(&ev.ID, &ev.Email, &ev.Name1, &ev.Name2, &ev.ExpiresAt, &ev.VerifiedAt)
	if err != nil {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}

	if time.Now().After(ev.ExpiresAt) {
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}
	if ev.VerifiedAt != nil {
		// Already verified — still return success with the data
		jsonOK(w, map[string]any{
			"ok":    true,
			"email": ev.Email,
			"name1": ev.Name1,
			"name2": ev.Name2,
		})
		return
	}

	_, err = h.db.Exec(r.Context(), `
		UPDATE onboarding_email_verifications SET verified_at = NOW() WHERE id = $1`, ev.ID)
	if err != nil {
		slog.Error("failed to mark email verification", "error", err)
		jsonError(w, "Link ungültig oder abgelaufen.", http.StatusBadRequest)
		return
	}

	jsonOK(w, map[string]any{
		"ok":    true,
		"email": ev.Email,
		"name1": ev.Name1,
		"name2": ev.Name2,
	})
}

// sendEmailVerificationLink sends the email address verification link to the applicant.
func (h *OnboardingHandler) sendEmailVerificationLink(eegID uuid.UUID, toEmail, name1, verifyLink string) {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), eegID)
	if err != nil || eeg.SMTPHost == "" {
		slog.Info("SMTP not configured, skipping email verification", "email", toEmail)
		return
	}
	smtpCfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}

	greeting := name1
	if greeting == "" {
		greeting = "Antragsteller/in"
	}
	subject := "E-Mail-Adresse bestätigen"
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #1e40af;">E-Mail-Adresse bestätigen</h2>
<p>Hallo %s,</p>
<p>bitte bestätigen Sie Ihre E-Mail-Adresse, indem Sie auf den folgenden Link klicken:</p>
<p style="margin: 24px 0;">
  <a href="%s" style="background-color: #1e40af; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">
    E-Mail bestätigen
  </a>
</p>
<p style="color: #64748b; font-size: 14px;">Oder kopieren Sie diesen Link: <a href="%s">%s</a></p>
<p style="color: #64748b; font-size: 14px;">Dieser Link ist 30 Minuten gültig.</p>
<p style="color: #64748b; font-size: 14px;">Falls Sie keinen Beitrittsantrag gestellt haben, können Sie diese E-Mail ignorieren.</p>
</body>
</html>`, greeting, verifyLink, verifyLink, verifyLink)

	var msgBuilder strings.Builder
	msgBuilder.WriteString("From: " + smtpCfg.From + "\r\n")
	msgBuilder.WriteString("To: " + toEmail + "\r\n")
	msgBuilder.WriteString("Subject: " + subject + "\r\n")
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	var smtpAuth smtp.Auth
	if smtpCfg.Username != "" {
		host := smtpCfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		smtpAuth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, host)
	}
	if err := smtp.SendMail(smtpCfg.Host, smtpAuth, smtpCfg.From, []string{toEmail}, []byte(msgBuilder.String())); err != nil {
		slog.Warn("failed to send email verification link", "email", toEmail, "error", err)
	}
}

// sendMagicTokenEmail sends the onboarding status link to the applicant.
func (h *OnboardingHandler) sendMagicTokenEmail(req *domain.OnboardingRequest, eegName string) error {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), req.EegID)
	if err != nil || eeg.SMTPHost == "" {
		slog.Info("SMTP not configured, skipping onboarding email", "request_id", req.ID)
		return nil
	}
	smtpCfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}

	// Fetch onboarding documents (AGB etc.) for this EEG to include in the email.
	docs, _ := h.docRepo.ListForOnboarding(context.Background(), req.EegID)

	statusURL := fmt.Sprintf("%s/onboarding/status?token=%s", h.webBaseURL, req.MagicToken)
	subject := fmt.Sprintf("Ihr Beitrittsantrag – %s", eegName)

	// Build optional document/AGB section
	var docSection string
	if len(docs) > 0 {
		var sb strings.Builder
		sb.WriteString(`<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">`)
		sb.WriteString(`<h3 style="color: #1e293b;">Dokumente</h3>`)
		sb.WriteString(`<ul style="color: #475569; line-height: 1.8;">`)
		for _, d := range docs {
			docURL := fmt.Sprintf("%s/api/public/eegs/%s/documents/%s", h.webBaseURL, req.EegID, d.ID)
			sb.WriteString(fmt.Sprintf(`<li><a href="%s" style="color: #1e40af;">%s</a></li>`, docURL, d.Title))
		}
		sb.WriteString(`</ul>`)
		docSection = sb.String()
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #1e40af;">Ihr Beitrittsantrag wurde eingereicht</h2>
<p>Liebe(r) %s,</p>
<p>vielen Dank für Ihren Beitrittsantrag zur Energiegemeinschaft <strong>%s</strong>.</p>
<p>Sie können den Status Ihres Antrags jederzeit unter folgendem Link einsehen:</p>
<p style="margin: 24px 0;">
  <a href="%s" style="background-color: #1e40af; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">
    Antragsstatus prüfen
  </a>
</p>
<p style="color: #64748b; font-size: 14px;">Oder kopieren Sie diesen Link: <a href="%s">%s</a></p>
<p style="color: #64748b; font-size: 14px;">Dieser Link ist 30 Tage gültig.</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<h3 style="color: #1e293b;">Was passiert als nächstes?</h3>
<ol style="color: #475569; line-height: 1.8;">
<li>Wir prüfen Ihren Antrag und melden uns bei Ihnen</li>
<li>Nach Genehmigung legen wir Ihr Mitgliedskonto an und stellen die EDA-Anmeldung beim Netzbetreiber</li>
<li>Sie erhalten eine E-Mail mit dem Link zum Portal Ihres Netzbetreibers</li>
<li>Dort aktivieren Sie den 15-Minuten-Takt und erteilen die Datenfreigabe für die Energiegemeinschaft</li>
<li>Sobald der Netzbetreiber die Anmeldung verarbeitet hat, beginnt die gemeinschaftliche Energieverrechnung</li>
</ol>
%s
<p style="color: #64748b; font-size: 14px;">Bei Fragen wenden Sie sich bitte direkt an die Energiegemeinschaft.</p>
</body>
</html>`,
		req.Name1, eegName, statusURL, statusURL, statusURL, docSection,
	)

	// Build simple MIME message
	var msgBuilder strings.Builder
	msgBuilder.WriteString("From: " + smtpCfg.From + "\r\n")
	msgBuilder.WriteString("To: " + req.Email + "\r\n")
	msgBuilder.WriteString("Subject: " + subject + "\r\n")
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	msgBytes := []byte(msgBuilder.String())

	var auth smtp.Auth
	if smtpCfg.Username != "" {
		host := smtpCfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, host)
	}

	return smtp.SendMail(smtpCfg.Host, auth, smtpCfg.From, []string{req.Email}, msgBytes)
}

// sendConversionEmail sends the approval + activation instructions email to the member.
// Called after successful conversion (member created + EDA Anmeldung sent).
func (h *OnboardingHandler) sendConversionEmail(req *domain.OnboardingRequest, eegName string, nb netzbetreiber.Info, customMessage string, memberID uuid.UUID) error {
	eeg2, err := h.eegRepo.GetByIDInternal(context.Background(), req.EegID)
	if err != nil || eeg2.SMTPHost == "" {
		return nil
	}
	statusURL := fmt.Sprintf("%s/onboarding/status?token=%s", h.webBaseURL, req.MagicToken)
	subject := fmt.Sprintf("Willkommen in der %s – Nächste Schritte", eegName)

	// Build Netzbetreiber portal block
	var nbBlock string
	if nb.PortalURL != "" {
		nbBlock = fmt.Sprintf(`
<div style="background: #f0f9ff; border: 1px solid #bae6fd; border-radius: 8px; padding: 16px; margin: 16px 0;">
  <strong style="color: #0369a1;">%s – %s</strong><br>
  <a href="%s" style="color: #1e40af;">%s</a><br>
  %s
</div>`, nb.Name, nb.PortalName, nb.PortalURL, nb.PortalURL,
			func() string {
				if nb.Hinweis != "" {
					return `<span style="color: #64748b; font-size: 13px;">` + nb.Hinweis + `</span>`
				}
				return ""
			}())
	} else {
		nbBlock = `<p style="color: #475569;">Bitte wenden Sie sich an Ihren Netzbetreiber für den Zugang zum Kundenportal.</p>`
	}

	// Custom message block
	customBlock := ""
	if customMessage != "" {
		customBlock = `<p style="color: #475569; background: #f8fafc; border-left: 3px solid #1e40af; padding: 12px 16px; margin: 16px 0;">` + customMessage + `</p>`
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #1e40af;">Ihr Beitritt wurde genehmigt!</h2>
<p>Liebe(r) %s,</p>
<p>Ihr Antrag zur Energiegemeinschaft <strong>%s</strong> wurde genehmigt. Wir haben die EDA-Anmeldung Ihres Zählpunkts beim Netzbetreiber eingereicht.</p>
%s
<h3 style="color: #1e293b; margin-top: 24px;">Was Sie jetzt tun müssen:</h3>
<ol style="color: #475569; line-height: 2;">
  <li><strong>Melden Sie sich im Portal Ihres Netzbetreibers an</strong> (Link oben)</li>
  <li><strong>Aktivieren Sie den 15-Minuten-Takt</strong> für Ihren Smartmeter</li>
  <li><strong>Erteilen Sie die Datenfreigabe</strong> für die Energiegemeinschaft %s</li>
</ol>
<p style="color: #475569; font-size: 14px;"><strong>Bitte beachten Sie:</strong> Es kann einige Stunden dauern, bis die Anfrage auf Datenfreigabe im Portal Ihres Netzbetreibers aufscheint – in der Regel innerhalb von 24 Stunden.</p>
<p style="color: #475569; font-size: 14px;">Sobald Ihr Netzbetreiber die Anmeldung verarbeitet hat, beginnt die gemeinschaftliche Energieverrechnung automatisch.</p>
%s
<p style="margin: 24px 0;">
  <a href="%s" style="background-color: #1e40af; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">
    Antragsstatus ansehen
  </a>
</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Bei Fragen wenden Sie sich bitte direkt an die Energiegemeinschaft.</p>
</body>
</html>`,
		req.Name1, eegName, nbBlock, eegName, customBlock, statusURL,
	)

	smtpCfg2 := invoice.SMTPConfig{Host: eeg2.SMTPHost, From: eeg2.SMTPFrom, Username: eeg2.SMTPUser, Password: eeg2.SMTPPassword}
	var msgBuilder strings.Builder
	msgBuilder.WriteString("From: " + smtpCfg2.From + "\r\n")
	msgBuilder.WriteString("To: " + req.Email + "\r\n")
	msgBuilder.WriteString("Subject: " + subject + "\r\n")
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	var auth smtp.Auth
	if smtpCfg2.Username != "" {
		host := smtpCfg2.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", smtpCfg2.Username, smtpCfg2.Password, host)
	}
	return smtp.SendMail(smtpCfg2.Host, auth, smtpCfg2.From, []string{req.Email}, []byte(msgBuilder.String()))
}

// RunReminderCheck queries for eda_sent onboarding requests older than 72 h
// that haven't received a reminder yet and sends a single follow-up email.
// Intended to be called from a background goroutine every hour.
func (h *OnboardingHandler) RunReminderCheck(ctx context.Context) {
	reqs, err := h.onboardingRepo.FindNeedingReminder(ctx, 72*time.Hour)
	if err != nil {
		slog.Error("reminder check: query failed", "error", err)
		return
	}
	for _, req := range reqs {
		req := req // capture
		eeg, err := h.eegRepo.GetByIDInternal(ctx, req.EegID)
		if err != nil || eeg.SMTPHost == "" {
			continue
		}
		nb, _ := netzbetreiber.ByMarktpartnerID(eeg.EdaNetzbetreiberID)
		if nb.Name == "" && len(req.MeterPoints) > 0 {
			nb, _ = netzbetreiber.ByZaehlpunkt(req.MeterPoints[0].Zaehlpunkt)
		}
		if err := h.sendReminderEmail(&req, eeg.Name, nb); err != nil {
			slog.Warn("reminder check: send failed", "request_id", req.ID, "error", err)
			continue
		}
		if err := h.onboardingRepo.SetReminderSent(ctx, req.ID); err != nil {
			slog.Error("reminder check: SetReminderSent failed", "request_id", req.ID, "error", err)
		} else {
			slog.Info("reminder sent", "request_id", req.ID, "email", req.Email)
		}
	}
}

// sendReminderEmail sends the 72-hour follow-up reminder to a member who has
// not yet confirmed data access in the grid operator portal.
func (h *OnboardingHandler) sendReminderEmail(req *domain.OnboardingRequest, eegName string, nb netzbetreiber.Info) error {
	eeg, err := h.eegRepo.GetByIDInternal(context.Background(), req.EegID)
	if err != nil || eeg.SMTPHost == "" {
		return nil
	}

	statusURL := fmt.Sprintf("%s/onboarding/status?token=%s", h.webBaseURL, req.MagicToken)
	subject := fmt.Sprintf("Erinnerung: Datenfreigabe für %s noch ausstehend", eegName)

	var nbBlock string
	if nb.PortalURL != "" {
		nbBlock = fmt.Sprintf(`
<div style="background: #f0f9ff; border: 1px solid #bae6fd; border-radius: 8px; padding: 16px; margin: 16px 0;">
  <strong style="color: #0369a1;">%s – %s</strong><br>
  <a href="%s" style="color: #1e40af;">%s</a>
</div>`, nb.Name, nb.PortalName, nb.PortalURL, nb.PortalURL)
	} else {
		nbBlock = `<p style="color: #475569;">Bitte wenden Sie sich an Ihren Netzbetreiber für den Zugang zum Kundenportal.</p>`
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #1e293b;">
<h2 style="color: #1e40af;">Erinnerung: Datenfreigabe noch ausstehend</h2>
<p>Liebe(r) %s,</p>
<p>Ihr Beitritt zur Energiegemeinschaft <strong>%s</strong> wurde genehmigt und die EDA-Anmeldung beim Netzbetreiber eingereicht.</p>
<p>Wir haben jedoch noch keine Rückmeldung erhalten, dass die Datenfreigabe in Ihrem Netzbetreiber-Portal bestätigt wurde.</p>
<p><strong>Bitte prüfen Sie Ihr Netzbetreiber-Portal und bestätigen Sie die Anfrage auf Datenfreigabe:</strong></p>
%s
<ol style="color: #475569; line-height: 2;">
  <li><strong>Melden Sie sich im Portal Ihres Netzbetreibers an</strong> (Link oben)</li>
  <li><strong>Aktivieren Sie den 15-Minuten-Takt</strong> für Ihren Smartmeter (falls noch nicht geschehen)</li>
  <li><strong>Erteilen Sie die Datenfreigabe</strong> für die Energiegemeinschaft %s</li>
</ol>
<p style="color: #475569; font-size: 14px;">Falls die Anfrage noch nicht im Portal aufscheint: Es kann in Ausnahmefällen bis zu 24 Stunden dauern. Bitte prüfen Sie nochmals zu einem späteren Zeitpunkt.</p>
<p style="color: #475569; font-size: 14px;">Bei Fragen wenden Sie sich bitte direkt an die Energiegemeinschaft.</p>
<p style="margin: 24px 0;">
  <a href="%s" style="background-color: #1e40af; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">
    Antragsstatus ansehen
  </a>
</p>
<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
<p style="color: #94a3b8; font-size: 12px;">Diese Erinnerung wurde automatisch verschickt, da seit Ihrer Aufnahme noch keine Bestätigung der Datenfreigabe vorliegt.</p>
</body>
</html>`, req.Name1, eegName, nbBlock, eegName, statusURL)

	smtpCfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}
	var msgBuilder strings.Builder
	msgBuilder.WriteString("From: " + smtpCfg.From + "\r\n")
	msgBuilder.WriteString("To: " + req.Email + "\r\n")
	msgBuilder.WriteString("Subject: " + subject + "\r\n")
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	var smtpAuth smtp.Auth
	if smtpCfg.Username != "" {
		host := smtpCfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		smtpAuth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, host)
	}
	return smtp.SendMail(smtpCfg.Host, smtpAuth, smtpCfg.From, []string{req.Email}, []byte(msgBuilder.String()))
}

// validIBAN validates an IBAN using the ISO 13616 checksum algorithm.
// Accepts AT and DE IBANs; strips spaces before validation.
func validIBAN(raw string) bool {
	iban := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
	if len(iban) < 5 {
		return false
	}
	country := iban[:2]
	switch country {
	case "AT":
		if len(iban) != 20 {
			return false
		}
	case "DE":
		if len(iban) != 22 {
			return false
		}
	default:
		return false
	}
	// Move first 4 chars to end, then replace letters with digits (A=10 … Z=35)
	rearranged := iban[4:] + iban[:4]
	var numeric strings.Builder
	for _, ch := range rearranged {
		if unicode.IsDigit(ch) {
			numeric.WriteRune(ch)
		} else if ch >= 'A' && ch <= 'Z' {
			fmt.Fprintf(&numeric, "%d", int(ch-'A')+10)
		} else {
			return false
		}
	}
	n := new(big.Int)
	n.SetString(numeric.String(), 10)
	mod := new(big.Int).Mod(n, big.NewInt(97))
	return mod.Int64() == 1
}
