package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	"github.com/xuri/excelize/v2"
)

type EEGHandler struct {
	eegRepo         *repository.EEGRepository
	memberRepo      *repository.MemberRepository
	meterPointRepo  *repository.MeterPointRepository
	participRepo    *repository.ParticipationRepository
}

func NewEEGHandler(eegRepo *repository.EEGRepository, memberRepo *repository.MemberRepository, meterPointRepo *repository.MeterPointRepository, participRepo *repository.ParticipationRepository) *EEGHandler {
	return &EEGHandler{eegRepo: eegRepo, memberRepo: memberRepo, meterPointRepo: meterPointRepo, participRepo: participRepo}
}

func publicLogoURL(eeg domain.EEG) string {
	if eeg.LogoPath == "" {
		return ""
	}
	return fmt.Sprintf("/api/v1/eegs/%s/logo", eeg.ID)
}

func presentEEG(eeg domain.EEG) domain.EEG {
	eeg.LogoPath = publicLogoURL(eeg)
	return eeg
}

func presentEEGs(eegs []domain.EEG) []domain.EEG {
	out := make([]domain.EEG, len(eegs))
	for i, eeg := range eegs {
		out[i] = presentEEG(eeg)
	}
	return out
}

// eegRequest is shared by CreateEEG and UpdateEEG.
type eegRequest struct {
	GemeinschaftID         string     `json:"gemeinschaft_id"`
	Netzbetreiber          string     `json:"netzbetreiber"`
	Name                   string     `json:"name"`
	EnergyPrice            float64    `json:"energy_price"`
	ProducerPrice          float64    `json:"producer_price"`
	UseVat                 bool       `json:"use_vat"`
	VatPct                 float64    `json:"vat_pct"`
	MeterFeeEur            float64    `json:"meter_fee_eur"`
	FreeKwh                float64    `json:"free_kwh"`
	DiscountPct            float64    `json:"discount_pct"`
	ParticipationFeeEur    float64    `json:"participation_fee_eur"`
	BillingPeriod          string     `json:"billing_period"`
	InvoiceNumberPrefix    string     `json:"invoice_number_prefix"`
	InvoiceNumberDigits    int        `json:"invoice_number_digits"`
	InvoiceNumberStart     int        `json:"invoice_number_start"`
	InvoicePreText         string     `json:"invoice_pre_text"`
	InvoicePostText        string     `json:"invoice_post_text"`
	InvoiceFooterText      string     `json:"invoice_footer_text"`
	GenerateCreditNotes    bool       `json:"generate_credit_notes"`
	CreditNoteNumberPrefix string     `json:"credit_note_number_prefix"`
	CreditNoteNumberDigits int        `json:"credit_note_number_digits"`
	IBAN                   string     `json:"iban"`
	BIC                    string     `json:"bic"`
	SepaCreditorID         string     `json:"sepa_creditor_id"`
	EdaTransitionDate      string     `json:"eda_transition_date,omitempty"`
	EdaMarktpartnerID      string     `json:"eda_marktpartner_id"`
	EdaNetzbetreiberID     string     `json:"eda_netzbetreiber_id"`
	// Accounting / DATEV
	AccountingRevenueAccount int    `json:"accounting_revenue_account"`
	AccountingExpenseAccount int    `json:"accounting_expense_account"`
	AccountingDebitorPrefix  int    `json:"accounting_debitor_prefix"`
	DatevConsultantNr        string `json:"datev_consultant_nr"`
	DatevClientNr            string `json:"datev_client_nr"`
	// Rechnungssteller address (§11 UStG)
	Strasse   string `json:"strasse"`
	Plz       string `json:"plz"`
	Ort       string `json:"ort"`
	UidNummer string `json:"uid_nummer"`
	// Gründungsdatum (date-only, YYYY-MM-DD)
	Gruendungsdatum string `json:"gruendungsdatum"`
	// Onboarding
	OnboardingContractText string `json:"onboarding_contract_text"`
	// Per-EEG EDA credentials (IMAP + SMTP for MaKo)
	EDAIMAPHost     string `json:"eda_imap_host"`
	EDAIMAPUser     string `json:"eda_imap_user"`
	EDAIMAPPassword string `json:"eda_imap_password"`
	EDASmtpHost     string `json:"eda_smtp_host"`
	EDASmtpUser     string `json:"eda_smtp_user"`
	EDASmtpPassword string `json:"eda_smtp_password"`
	EDASmtpFrom     string `json:"eda_smtp_from"`
	// Per-EEG invoice SMTP credentials
	SMTPHost     string `json:"smtp_host"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`
	// Auto-billing
	AutoBillingEnabled    bool   `json:"auto_billing_enabled"`
	AutoBillingDayOfMonth int    `json:"auto_billing_day_of_month"`
	AutoBillingPeriod     string `json:"auto_billing_period"`
	// Gap alert
	GapAlertEnabled       bool `json:"gap_alert_enabled"`
	GapAlertThresholdDays int  `json:"gap_alert_threshold_days"`
	// Member portal
	PortalShowFullEnergy bool `json:"portal_show_full_energy"`
}

// memberWithMeterPoints is the response shape for ListMembers and GetMember.
type memberWithMeterPoints struct {
	ID             uuid.UUID         `json:"id"`
	EegID          uuid.UUID         `json:"eeg_id"`
	MitgliedsNr    string            `json:"mitglieds_nr"`
	Name           string            `json:"name"`
	Name1          string            `json:"name1"`
	Name2          string            `json:"name2"`
	Email          string            `json:"email"`
	IBAN           string            `json:"iban"`
	Strasse        string            `json:"strasse"`
	Plz            string            `json:"plz"`
	Ort            string            `json:"ort"`
	BusinessRole   string            `json:"business_role"`
	UidNummer      string            `json:"uid_nummer"`
	UseVat         *bool             `json:"use_vat"`
	VatPct         *float64          `json:"vat_pct"`
	Status         string            `json:"status"`
	BeitrittsDatum *time.Time        `json:"beitritts_datum,omitempty"`
	AustrittsDatum *time.Time        `json:"austritts_datum,omitempty"`
	MeterPoints    []meterPointShort `json:"meter_points"`
}

type meterPointShort struct {
	ID                  uuid.UUID  `json:"id"`
	MeterID             string     `json:"meter_id"`
	Direction           string     `json:"direction"`
	GenerationType      *string    `json:"generation_type,omitempty"`
	ParticipationFactor float64    `json:"participation_factor"`
	FactorValidFrom     *time.Time `json:"factor_valid_from,omitempty"` // valid_from of the EDA process that set the factor
	RegistriertSeit     *time.Time `json:"registriert_seit,omitempty"`
	AbgemeldetAm        *time.Time `json:"abgemeldet_am,omitempty"`
	AnmeldungStatus     *string    `json:"anmeldung_status,omitempty"` // latest EC_REQ_ONL status
	AbmeldungStatus     *string    `json:"abmeldung_status,omitempty"` // latest CM_REV_SP status
}

func toMemberWithMPs(m domain.Member, mps []domain.MeterPoint, factors map[uuid.UUID]float64, edaStatus map[uuid.UUID]repository.MeterPointEDAStatus) memberWithMeterPoints {
	name := m.Name1
	if m.Name2 != "" {
		name += " " + m.Name2
	}
	shorts := make([]meterPointShort, 0, len(mps))
	for _, mp := range mps {
		factor := 100.0
		if f, ok := factors[mp.ID]; ok {
			factor = f
		} else if s := edaStatus[mp.ID]; s.ParticipationFactor != nil {
			factor = *s.ParticipationFactor
		}
		s := edaStatus[mp.ID]
		shorts = append(shorts, meterPointShort{
			ID:                  mp.ID,
			MeterID:             mp.Zaehlpunkt,
			Direction:           mp.Energierichtung,
			GenerationType:      mp.GenerationType,
			ParticipationFactor: factor,
			FactorValidFrom:     s.FactorValidFrom,
			RegistriertSeit:     mp.RegistriertSeit,
			AbgemeldetAm:        mp.AbgemeldetAm,
			AnmeldungStatus:     s.AnmeldungStatus,
			AbmeldungStatus:     s.AbmeldungStatus,
		})
	}
	return memberWithMeterPoints{
		ID:             m.ID,
		EegID:          m.EegID,
		MitgliedsNr:    m.MitgliedsNr,
		Name:           name,
		Name1:          m.Name1,
		Name2:          m.Name2,
		Email:          m.Email,
		IBAN:           m.IBAN,
		Strasse:        m.Strasse,
		Plz:            m.Plz,
		Ort:            m.Ort,
		BusinessRole:   m.BusinessRole,
		UidNummer:      m.UidNummer,
		UseVat:         m.UseVat,
		VatPct:         m.VatPct,
		Status:         m.Status,
		BeitrittsDatum: m.BeitrittsDatum,
		AustrittsDatum: m.AustrittsDatum,
		MeterPoints:    shorts,
	}
}

// ListEEGs godoc
// @Summary     List EEGs
// @Description Returns all EEGs accessible to the authenticated user. Admins see all EEGs in the organization; other roles see only their assigned EEGs.
// @Tags        EEGs
// @Produce     json
// @Success     200  {array}   domain.EEG  "List of EEGs"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs [get]
func (h *EEGHandler) ListEEGs(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var eegs []domain.EEG
	var err error
	if claims.Role == "admin" {
		eegs, err = h.eegRepo.List(r.Context(), claims.OrganizationID)
	} else {
		userID, _ := uuid.Parse(claims.RegisteredClaims.Subject)
		eegs, err = h.eegRepo.ListForUser(r.Context(), userID)
	}
	if err != nil {
		jsonError(w, "failed to list EEGs", http.StatusInternalServerError)
		return
	}
	if eegs == nil {
		eegs = []domain.EEG{}
	}
	jsonOK(w, presentEEGs(eegs))
}

// CreateEEG godoc
// @Summary     Create EEG
// @Description Creates a new Energiegemeinschaft (EEG) within the authenticated user's organization. Fields name and netzbetreiber are required.
// @Tags        EEGs
// @Accept      json
// @Produce     json
// @Param       eeg  body      eegRequest  true  "EEG data (name and netzbetreiber required)"
// @Success     201  {object}  domain.EEG  "Created EEG"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs [post]
func (h *EEGHandler) CreateEEG(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req eegRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Netzbetreiber == "" || req.Name == "" {
		jsonError(w, "netzbetreiber and name are required", http.StatusBadRequest)
		return
	}

	eeg := &domain.EEG{
		OrganizationID:      claims.OrganizationID,
		GemeinschaftID:      req.GemeinschaftID,
		Netzbetreiber:       req.Netzbetreiber,
		Name:                req.Name,
		EnergyPrice:         req.EnergyPrice,
		ProducerPrice:       req.ProducerPrice,
		UseVat:              req.UseVat,
		VatPct:              req.VatPct,
		MeterFeeEur:         req.MeterFeeEur,
		FreeKwh:             req.FreeKwh,
		DiscountPct:         req.DiscountPct,
		ParticipationFeeEur: req.ParticipationFeeEur,
		BillingPeriod:       req.BillingPeriod,
		InvoiceNumberPrefix: req.InvoiceNumberPrefix,
		InvoiceNumberDigits: req.InvoiceNumberDigits,
		InvoicePreText:      req.InvoicePreText,
		InvoicePostText:     req.InvoicePostText,
		InvoiceFooterText:   req.InvoiceFooterText,
		IBAN:               req.IBAN,
		BIC:                req.BIC,
		SepaCreditorID:     req.SepaCreditorID,
		EdaMarktpartnerID:  req.EdaMarktpartnerID,
		EdaNetzbetreiberID: req.EdaNetzbetreiberID,
	}
	if req.EdaTransitionDate != "" {
		t, err := time.Parse("2006-01-02", req.EdaTransitionDate)
		if err != nil {
			jsonError(w, "ungültiges EDA-Umstellungsdatum (Format: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		eeg.EdaTransitionDate = &t
	}
	if err := h.eegRepo.Create(r.Context(), eeg); err != nil {
		jsonError(w, "failed to create EEG", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, presentEEG(*eeg))
}

// UpdateEEG godoc
// @Summary     Update EEG
// @Description Updates settings of an existing EEG. Only fields provided in the body are applied; name and netzbetreiber are kept if omitted.
// @Tags        EEGs
// @Accept      json
// @Produce     json
// @Param       eegID  path      string      true  "EEG UUID"
// @Param       eeg    body      eegRequest  true  "EEG update data"
// @Success     200  {object}  domain.EEG  "Updated EEG"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID} [put]
func (h *EEGHandler) UpdateEEG(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	existing, err := h.eegRepo.GetByID(r.Context(), id, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	var req eegRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Netzbetreiber != "" {
		existing.Netzbetreiber = req.Netzbetreiber
	}
	existing.EnergyPrice = req.EnergyPrice
	existing.ProducerPrice = req.ProducerPrice
	existing.UseVat = req.UseVat
	existing.VatPct = req.VatPct
	existing.MeterFeeEur = req.MeterFeeEur
	existing.FreeKwh = req.FreeKwh
	existing.DiscountPct = req.DiscountPct
	existing.ParticipationFeeEur = req.ParticipationFeeEur
	if req.BillingPeriod != "" {
		existing.BillingPeriod = req.BillingPeriod
	}
	if req.InvoiceNumberPrefix != "" {
		existing.InvoiceNumberPrefix = req.InvoiceNumberPrefix
	}
	if req.InvoiceNumberDigits > 0 {
		existing.InvoiceNumberDigits = req.InvoiceNumberDigits
	}
	if req.InvoiceNumberStart > 0 {
		existing.InvoiceNumberStart = req.InvoiceNumberStart
	}
	existing.InvoicePreText = req.InvoicePreText
	existing.InvoicePostText = req.InvoicePostText
	existing.InvoiceFooterText = req.InvoiceFooterText
	existing.GenerateCreditNotes = req.GenerateCreditNotes
	if req.CreditNoteNumberPrefix != "" {
		existing.CreditNoteNumberPrefix = req.CreditNoteNumberPrefix
	}
	if req.CreditNoteNumberDigits > 0 {
		existing.CreditNoteNumberDigits = req.CreditNoteNumberDigits
	}
	existing.IBAN = req.IBAN
	existing.BIC = req.BIC
	existing.SepaCreditorID = req.SepaCreditorID
	if req.EdaTransitionDate != "" {
		t, err := time.Parse("2006-01-02", req.EdaTransitionDate)
		if err != nil {
			jsonError(w, "ungültiges EDA-Umstellungsdatum (Format: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		existing.EdaTransitionDate = &t
	} else {
		existing.EdaTransitionDate = nil
	}
	existing.EdaMarktpartnerID = req.EdaMarktpartnerID
	existing.EdaNetzbetreiberID = req.EdaNetzbetreiberID
	if req.AccountingRevenueAccount > 0 {
		existing.AccountingRevenueAccount = req.AccountingRevenueAccount
	}
	if req.AccountingExpenseAccount > 0 {
		existing.AccountingExpenseAccount = req.AccountingExpenseAccount
	}
	if req.AccountingDebitorPrefix > 0 {
		existing.AccountingDebitorPrefix = req.AccountingDebitorPrefix
	}
	existing.DatevConsultantNr = req.DatevConsultantNr
	existing.DatevClientNr = req.DatevClientNr
	existing.Strasse = req.Strasse
	existing.Plz = req.Plz
	existing.Ort = req.Ort
	existing.UidNummer = req.UidNummer
	existing.OnboardingContractText = req.OnboardingContractText
	// Credentials — only overwrite if non-empty in request (empty = keep existing)
	if req.EDAIMAPHost != "" {
		existing.EDAIMAPHost = req.EDAIMAPHost
	}
	if req.EDAIMAPUser != "" {
		existing.EDAIMAPUser = req.EDAIMAPUser
	}
	if req.EDAIMAPPassword != "" {
		existing.EDAIMAPPassword = req.EDAIMAPPassword
	}
	if req.EDASmtpHost != "" {
		existing.EDASmtpHost = req.EDASmtpHost
	}
	if req.EDASmtpUser != "" {
		existing.EDASmtpUser = req.EDASmtpUser
	}
	if req.EDASmtpPassword != "" {
		existing.EDASmtpPassword = req.EDASmtpPassword
	}
	if req.EDASmtpFrom != "" {
		existing.EDASmtpFrom = req.EDASmtpFrom
	}
	if req.SMTPHost != "" {
		existing.SMTPHost = req.SMTPHost
	}
	if req.SMTPUser != "" {
		existing.SMTPUser = req.SMTPUser
	}
	if req.SMTPPassword != "" {
		existing.SMTPPassword = req.SMTPPassword
	}
	if req.SMTPFrom != "" {
		existing.SMTPFrom = req.SMTPFrom
	}
	if req.Gruendungsdatum != "" {
		t, err := time.Parse("2006-01-02", req.Gruendungsdatum)
		if err != nil {
			jsonError(w, "ungültiges Gründungsdatum (Format: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		if !t.Before(time.Now()) {
			jsonError(w, "Gründungsdatum muss in der Vergangenheit liegen", http.StatusBadRequest)
			return
		}
		existing.Gruendungsdatum = &t
	} else {
		existing.Gruendungsdatum = nil
	}

	// Auto-billing settings
	existing.AutoBillingEnabled = req.AutoBillingEnabled
	if req.AutoBillingDayOfMonth < 1 || req.AutoBillingDayOfMonth > 28 {
		existing.AutoBillingDayOfMonth = 0
	} else {
		existing.AutoBillingDayOfMonth = req.AutoBillingDayOfMonth
	}
	if req.AutoBillingPeriod == "monthly" || req.AutoBillingPeriod == "quarterly" {
		existing.AutoBillingPeriod = req.AutoBillingPeriod
	} else {
		existing.AutoBillingPeriod = "monthly"
	}

	// Gap alert settings
	existing.GapAlertEnabled = req.GapAlertEnabled
	if req.GapAlertThresholdDays <= 0 {
		existing.GapAlertThresholdDays = 5
	} else {
		existing.GapAlertThresholdDays = req.GapAlertThresholdDays
	}

	// Member portal
	existing.PortalShowFullEnergy = req.PortalShowFullEnergy

	if err := h.eegRepo.Update(r.Context(), existing); err != nil {
		jsonError(w, "failed to update EEG", http.StatusInternalServerError)
		return
	}
	jsonOK(w, presentEEG(*existing))
}

// GetEEG godoc
// @Summary     Get EEG
// @Description Returns a single EEG by its UUID, scoped to the authenticated user's organization.
// @Tags        EEGs
// @Produce     json
// @Param       eegID  path      string      true  "EEG UUID"
// @Success     200  {object}  domain.EEG  "EEG object"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID} [get]
func (h *EEGHandler) GetEEG(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), id, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	jsonOK(w, presentEEG(*eeg))
}

// ListGapAlerts godoc
// @Summary     List current gap alerts for an EEG
// @Description Returns meter points with gap_alert_sent_at set (i.e. an alert was sent and data is still missing).
// @Tags        EEGs
// @Produce     json
// @Param       eegID  path      string  true  "EEG UUID"
// @Success     200  {array}   repository.MeterPointGap  "List of gap alerts"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/gap-alerts [get]
func (h *EEGHandler) ListGapAlerts(w http.ResponseWriter, r *http.Request) {
	_, _, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	gaps, err := h.meterPointRepo.ListCurrentGapAlerts(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to list gap alerts", http.StatusInternalServerError)
		return
	}
	if gaps == nil {
		gaps = []repository.MeterPointGap{}
	}
	jsonOK(w, gaps)
}

// UploadLogo godoc
// @Summary     Upload EEG logo
// @Description Accepts a JPEG or PNG image (max 4 MB) via multipart/form-data field "logo", stores it on disk and updates the EEG logo.
// @Tags        EEGs
// @Accept      multipart/form-data
// @Produce     json
// @Param       eegID  path      string  true  "EEG UUID"
// @Param       logo   formData  file    true  "Logo image (JPEG or PNG, max 4 MB)"
// @Success     200  {object}  map[string]bool  "Upload status"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/logo [post]
func (h *EEGHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(4 << 20); err != nil { // 4 MB
		jsonError(w, "failed to parse form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("logo")
	if err != nil {
		jsonError(w, "logo file missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")
	var ext string
	switch ct {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	default:
		jsonError(w, "only JPEG and PNG are accepted", http.StatusBadRequest)
		return
	}

	logoDir := os.Getenv("INVOICE_DIR")
	if logoDir == "" {
		logoDir = "/data/invoices"
	}
	logoDir = filepath.Join(logoDir, "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		jsonError(w, "failed to create logo directory", http.StatusInternalServerError)
		return
	}

	logoPath := filepath.Join(logoDir, eeg.ID.String()+ext)
	out, err := os.Create(logoPath)
	if err != nil {
		jsonError(w, "failed to write logo", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		jsonError(w, "failed to write logo", http.StatusInternalServerError)
		return
	}

	if err := h.eegRepo.UpdateLogo(r.Context(), eeg.ID, logoPath); err != nil {
		jsonError(w, "failed to update logo path", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]bool{"uploaded": true})
}

// GetLogo godoc
// @Summary     Get EEG logo
// @Description Serves the stored logo image for an EEG. Returns image/jpeg or image/png depending on the stored file extension.
// @Tags        EEGs
// @Produce     image/jpeg
// @Produce     image/png
// @Param       eegID  path  string  true  "EEG UUID"
// @Success     200  {file}    binary  "Logo image"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/logo [get]
// GetLogo handles GET /eegs/{eegID}/logo — serves the logo image file.
func (h *EEGHandler) GetLogo(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	if eeg.LogoPath == "" {
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(eeg.LogoPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(eeg.LogoPath))
	ct := "image/png"
	if ext == ".jpg" || ext == ".jpeg" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-cache")
	io.Copy(w, f) //nolint:errcheck
}

// ExportStammdaten godoc
// @Summary     Export master data as XLSX
// @Description Generates and downloads an XLSX workbook with two sheets: Mitglieder (members) and Zählpunkte (meter points) for the given EEG.
// @Tags        EEGs
// @Produce     application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param       eegID  path  string  true  "EEG UUID"
// @Success     200  {file}    binary  "XLSX file attachment"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/export/stammdaten [get]
// ExportStammdaten handles GET /eegs/{eegID}/export/stammdaten
// Returns an XLSX with two sheets: Mitglieder and Zählpunkte.
func (h *EEGHandler) ExportStammdaten(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	members, err := h.memberRepo.ListByEeg(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to list members", http.StatusInternalServerError)
		return
	}
	mps, err := h.meterPointRepo.ListByEeg(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to list meter points", http.StatusInternalServerError)
		return
	}

	f := excelize.NewFile()

	// Sheet 1: Mitglieder
	sheet1 := "Mitglieder"
	f.SetSheetName("Sheet1", sheet1)
	headers1 := []string{"Mitglieds-Nr", "Name1", "Name2", "E-Mail", "IBAN", "Straße", "PLZ", "Ort", "Rolle", "UID-Nr", "Status", "Erstellt"}
	for i, h := range headers1 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}
	for row, m := range members {
		r := row + 2
		vals := []any{
			m.MitgliedsNr, m.Name1, m.Name2, m.Email, m.IBAN,
			m.Strasse, m.Plz, m.Ort, m.BusinessRole, m.UidNummer, m.Status,
			m.CreatedAt.Format("2006-01-02"),
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, r)
			f.SetCellValue(sheet1, cell, v)
		}
	}

	// Sheet 2: Zählpunkte
	sheet2 := "Zählpunkte"
	f.NewSheet(sheet2)
	headers2 := []string{"Zählpunkt", "Energierichtung", "Verteilungsmodell", "Zuteilung %", "Status", "Registriert seit", "Mitglieds-ID"}
	for i, h := range headers2 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet2, cell, h)
	}
	for row, mp := range mps {
		r := row + 2
		since := ""
		if mp.RegistriertSeit != nil {
			since = mp.RegistriertSeit.Format("2006-01-02")
		}
		vals := []any{
			mp.Zaehlpunkt, mp.Energierichtung, mp.Verteilungsmodell,
			mp.ZugeteilteMenugePct, mp.Status, since, mp.MemberID.String(),
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, r)
			f.SetCellValue(sheet2, cell, v)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		jsonError(w, "failed to generate XLSX", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Stammdaten_%s.xlsx\"", id.String()[:8]))
	w.Write(buf.Bytes())
}

// ListMembers godoc
// @Summary     List members of an EEG
// @Description Returns all members belonging to the given EEG, each with their assigned meter points. Supports optional full-text search (q) and status filter.
// @Tags        EEGs
// @Produce     json
// @Param       eegID   path   string  true   "EEG UUID"
// @Param       q       query  string  false  "Full-text search string (name, email, member number)"
// @Param       status  query  string  false  "Filter by member status (ACTIVE or INACTIVE)"
// @Success     200  {array}   memberWithMeterPoints  "List of members with meter points"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/members [get]
func (h *EEGHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	stichtagStr := r.URL.Query().Get("stichtag")
	var stichtag *time.Time
	if stichtagStr != "" {
		if t, parseErr := time.Parse("2006-01-02", stichtagStr); parseErr == nil {
			stichtag = &t
		}
	}
	var members []domain.Member
	if q != "" || status != "" || stichtag != nil {
		members, err = h.memberRepo.SearchByEeg(r.Context(), id, q, status, stichtag)
	} else {
		members, err = h.memberRepo.ListByEeg(r.Context(), id)
	}
	if err != nil {
		jsonError(w, "failed to list members", http.StatusInternalServerError)
		return
	}

	allMPs, err := h.meterPointRepo.ListByEeg(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to list meter points", http.StatusInternalServerError)
		return
	}
	mpsByMember := make(map[uuid.UUID][]domain.MeterPoint)
	for _, mp := range allMPs {
		mpsByMember[mp.MemberID] = append(mpsByMember[mp.MemberID], mp)
	}

	factors, err := h.participRepo.GetCurrentFactorsByEEG(r.Context(), id)
	if err != nil {
		factors = map[uuid.UUID]float64{} // non-fatal, fall back to 100%
	}

	mpIDs := make([]uuid.UUID, 0, len(allMPs))
	for _, mp := range allMPs {
		mpIDs = append(mpIDs, mp.ID)
	}
	edaStatus, err := h.meterPointRepo.GetEDAStatusByMeterPoints(r.Context(), mpIDs)
	if err != nil {
		edaStatus = map[uuid.UUID]repository.MeterPointEDAStatus{}
	}

	result := make([]memberWithMeterPoints, 0, len(members))
	for _, m := range members {
		result = append(result, toMemberWithMPs(m, mpsByMember[m.ID], factors, edaStatus))
	}
	jsonOK(w, result)
}

// DeleteEEG godoc
// @Summary     Delete EEG
// @Description Permanently deletes an EEG and all its associated data. This action is irreversible.
// @Tags        EEGs
// @Param       eegID  path  string  true  "EEG UUID"
// @Success     204  "No content"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID} [delete]
// DeleteEEG handles DELETE /api/v1/eegs/{eegID}
func (h *EEGHandler) DeleteEEG(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if err := h.eegRepo.Delete(r.Context(), eegID, claims.OrganizationID); err != nil {
		if err.Error() == "not found" {
			jsonError(w, "EEG not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to delete EEG: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
