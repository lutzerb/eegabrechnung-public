package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	_ "time/tzdata" // embed IANA timezone database (required in Alpine containers)

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// EDAHandler handles EDA process management (Anmeldung, Abmeldung, Teilnahmefaktor).
type EDAHandler struct {
	eegRepo          *repository.EEGRepository
	mpRepo           *repository.MeterPointRepository
	edaProcRepo      *repository.EDAProcessRepository
	jobRepo          *repository.JobRepository
	edaErrorRepo     *repository.EDAErrorRepository
	workerStatusRepo *repository.EDAWorkerStatusRepository
	edaWorkerURL     string
}

func NewEDAHandler(
	eegRepo *repository.EEGRepository,
	mpRepo *repository.MeterPointRepository,
	edaProcRepo *repository.EDAProcessRepository,
	jobRepo *repository.JobRepository,
	edaErrorRepo *repository.EDAErrorRepository,
	workerStatusRepo *repository.EDAWorkerStatusRepository,
	edaWorkerURL string,
) *EDAHandler {
	return &EDAHandler{
		eegRepo:          eegRepo,
		mpRepo:           mpRepo,
		edaProcRepo:      edaProcRepo,
		jobRepo:          jobRepo,
		edaErrorRepo:     edaErrorRepo,
		workerStatusRepo: workerStatusRepo,
		edaWorkerURL:     edaWorkerURL,
	}
}

// ListProcesses godoc
//
//	@Summary		List EDA processes
//	@Description	Returns all EDA processes (Anmeldung, Abmeldung, Teilnahmefaktor) for an EEG.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG ID (UUID)"
//	@Success		200		{array}		domain.EDAProcess
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/processes [get]
func (h *EDAHandler) ListProcesses(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	ps, err := h.edaProcRepo.ListByEEG(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to list EDA processes", http.StatusInternalServerError)
		return
	}
	if ps == nil {
		ps = []domain.EDAProcess{}
	}
	jsonOK(w, ps)
}

// anmeldungRequest is the body for POST /eda/anmeldung.
type anmeldungRequest struct {
	Zaehlpunkt          string  `json:"zaehlpunkt"`
	ValidFrom           string  `json:"valid_from"`           // YYYY-MM-DD
	ShareType           string  `json:"share_type"`           // GC, RC_R, RC_L, CC …
	ParticipationFactor float64 `json:"participation_factor"` // 0..100
	EnergyDirection     string  `json:"energy_direction"`     // CONSUMPTION or GENERATION
}

// Anmeldung godoc
//
//	@Summary		Register meter point (EC_EINZEL_ANM)
//	@Description	Creates an EC_EINZEL_ANM process to register a single meter point with the energy community (Netzbetreiber). Queues an outbound XML job for the EDA worker.
//	@Tags			EDA
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG ID (UUID)"
//	@Param			body	body		anmeldungRequest	true	"Anmeldung request"
//	@Success		201		{object}	domain.EDAProcess
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/anmeldung [post]
func (h *EDAHandler) Anmeldung(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	if eeg.IsDemo {
		jsonError(w, "EDA-Nachrichten sind im Demo-Modus deaktiviert", http.StatusForbidden)
		return
	}
	if eeg.EdaMarktpartnerID == "" || eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Marktpartner-ID und Netzbetreiber-ID müssen in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}

	var req anmeldungRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Zaehlpunkt == "" {
		jsonError(w, "zaehlpunkt is required", http.StatusBadRequest)
		return
	}
	if len(req.Zaehlpunkt) >= 8 && req.Zaehlpunkt[:8] != eeg.EdaNetzbetreiberID {
		jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (Präfix: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, req.Zaehlpunkt[:8]), http.StatusBadRequest)
		return
	}

	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	tomorrowVienna := time.Now().In(viennaLoc).AddDate(0, 0, 1)
	tomorrow := time.Date(tomorrowVienna.Year(), tomorrowVienna.Month(), tomorrowVienna.Day(), 0, 0, 0, 0, time.UTC)
	validFrom := tomorrow
	if req.ValidFrom != "" {
		t, parseErr := time.Parse("2006-01-02", req.ValidFrom)
		if parseErr != nil {
			jsonError(w, "valid_from must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		validFrom = t
	}
	maxDate := tomorrow.AddDate(0, 0, 30)
	if validFrom.Before(tomorrow) || validFrom.After(maxDate) {
		jsonError(w, "valid_from muss frühestens morgen und höchstens 30 Tage in der Zukunft liegen", http.StatusBadRequest)
		return
	}

	// Derive Netzbetreiber-ID from Zählpunkt prefix; more reliable than the EEG-wide setting.
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		netzbetreiberTo = req.Zaehlpunkt[:8]
	}

	msgID := uuid.NewString()
	convID := uuid.NewString()

	energyDirection := req.EnergyDirection
	if energyDirection == "" {
		energyDirection = "CONSUMPTION"
	}

	xmlBody, err := edaxml.BuildCMRequest(edaxml.CMRequestParams{
		From:            eeg.EdaMarktpartnerID,
		To:              netzbetreiberTo,
		MessageID:       msgID,
		ConversationID:  convID,
		CMRequestID:     uuid.NewString(),
		MeteringPoint:   req.Zaehlpunkt,
		ECID:            eeg.GemeinschaftID,
		DateFrom:        validFrom,
		ECPartFact:      req.ParticipationFactor,
		EnergyDirection: energyDirection,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
		return
	}

	// Resolve meter_point_id if available (best effort).
	var mpID *uuid.UUID
	if mp, err := h.mpRepo.GetByZaehlpunkt(r.Context(), req.Zaehlpunkt); err == nil {
		id := mp.ID
		mpID = &id
	}

	now := time.Now()
	deadline := now.AddDate(0, 2, 0) // 2 months (EAG §16e Abs. 1)

	proc := &domain.EDAProcess{
		EegID:               eegID,
		MeterPointID:        mpID,
		ProcessType:         "EC_REQ_ONL",
		Status:              "pending",
		ConversationID:      convID,
		Zaehlpunkt:          req.Zaehlpunkt,
		ShareType:           req.ShareType,
		InitiatedAt:         now,
		DeadlineAt:          &deadline,
	}
	if !validFrom.IsZero() {
		proc.ValidFrom = &validFrom
	}
	if req.ParticipationFactor > 0 {
		proc.ParticipationFactor = &req.ParticipationFactor
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		jsonError(w, "failed to create EDA process record", http.StatusInternalServerError)
		return
	}

	// Queue outbound job for the worker.
	if err := h.jobRepo.EnqueueEDA(r.Context(), "EC_REQ_ONL", eeg.EdaMarktpartnerID, netzbetreiberTo,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}

// teilnahmefaktorRequest is the body for POST /eda/teilnahmefaktor.
type teilnahmefaktorRequest struct {
	Zaehlpunkt          string   `json:"zaehlpunkt"`
	ParticipationFactor float64  `json:"participation_factor"` // 0..100
	ShareType           string   `json:"share_type"`           // GC, RC_R, RC_L, CC → ECType
	ECDisModel          string   `json:"ec_dis_model"`         // S or D (default "S")
	DateTo              string   `json:"date_to"`              // YYYY-MM-DD (default empty = 9999-12-31)
	EnergyDirection     string   `json:"energy_direction"`     // CONSUMPTION or GENERATION (default "CONSUMPTION")
	ECShare             *float64 `json:"ec_share,omitempty"`
	ValidFrom           string   `json:"valid_from"` // YYYY-MM-DD; defaults to tomorrow
}

// TeilnahmefaktorAendern godoc
//
//	@Summary		Change participation factor (EC_PRTFACT_CHG)
//	@Description	Creates an EC_PRTFACT_CHG process to change the participation factor for a meter point. Restricted to 09:00–17:00 Vienna time; effective from the next calendar day. Only one change per Zählpunkt per day is allowed.
//	@Tags			EDA
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string						true	"EEG ID (UUID)"
//	@Param			body	body		teilnahmefaktorRequest		true	"Teilnahmefaktor change request"
//	@Success		201		{object}	domain.EDAProcess
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string	"Duplicate change for today"
//	@Failure		422		{object}	map[string]string	"Outside allowed time window (09:00–17:00 Vienna)"
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/teilnahmefaktor [post]
func (h *EDAHandler) TeilnahmefaktorAendern(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Time-of-day restriction: 09:00–17:00 Vienna time (per EDA protocol).
	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	now := time.Now().In(viennaLoc)
	if now.Hour() < 9 || now.Hour() >= 17 {
		jsonError(w, "EC_PRTFACT_CHG ist nur zwischen 09:00 und 17:00 Uhr (Wien) erlaubt", http.StatusUnprocessableEntity)
		return
	}

	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	if eeg.IsDemo {
		jsonError(w, "EDA-Nachrichten sind im Demo-Modus deaktiviert", http.StatusForbidden)
		return
	}
	if eeg.EdaMarktpartnerID == "" || eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Marktpartner-ID und Netzbetreiber-ID müssen konfiguriert sein", http.StatusBadRequest)
		return
	}

	var req teilnahmefaktorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Zaehlpunkt == "" {
		jsonError(w, "zaehlpunkt is required", http.StatusBadRequest)
		return
	}
	if len(req.Zaehlpunkt) >= 8 && req.Zaehlpunkt[:8] != eeg.EdaNetzbetreiberID {
		jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (Präfix: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, req.Zaehlpunkt[:8]), http.StatusBadRequest)
		return
	}
	if req.ParticipationFactor <= 0 || req.ParticipationFactor > 100 {
		jsonError(w, "participation_factor must be between 0 and 100", http.StatusBadRequest)
		return
	}

	// Check: only one change per day per Zählpunkt.
	dup, err := h.edaProcRepo.HasPendingFactorChangeToday(r.Context(), eegID, req.Zaehlpunkt)
	if err != nil {
		jsonError(w, "failed to check duplicate", http.StatusInternalServerError)
		return
	}
	if dup {
		jsonError(w, "Es gibt bereits eine Teilnahmefaktor-Änderung für diesen Zählpunkt heute", http.StatusConflict)
		return
	}

	// ValidFrom defaults to tomorrow (change takes effect next calendar day).
	tomorrow := time.Now().In(viennaLoc).AddDate(0, 0, 1)
	validFrom := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
	if req.ValidFrom != "" {
		if t, err := time.Parse("2006-01-02", req.ValidFrom); err == nil {
			validFrom = t
		}
	}

	// Apply defaults for optional fields.
	ecDisModel := req.ECDisModel
	if ecDisModel == "" {
		ecDisModel = "S"
	}
	energyDirection := req.EnergyDirection
	if energyDirection == "" {
		energyDirection = "CONSUMPTION"
	}
	shareType := req.ShareType
	if shareType == "" {
		shareType = "GC"
	}

	// DateTo: parse optional date_to field.
	var dateTo time.Time
	if req.DateTo != "" {
		if t, parseErr := time.Parse("2006-01-02", req.DateTo); parseErr == nil {
			dateTo = t
		}
	}

	// Derive Netzbetreiber-ID from Zählpunkt prefix; more reliable than the EEG-wide setting.
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		netzbetreiberTo = req.Zaehlpunkt[:8]
	}

	convID := uuid.NewString()
	xmlBody, err := edaxml.BuildECMPList(edaxml.ECMPListParams{
		From:            eeg.EdaMarktpartnerID,
		To:              netzbetreiberTo,
		MessageID:       uuid.NewString(),
		ConversationID:  convID,
		ECID:            eeg.GemeinschaftID,
		ECType:          shareType,
		ECDisModel:      ecDisModel,
		MessageCode:     "ANFORDERUNG_CPF",
		MeteringPoint:   req.Zaehlpunkt,
		DateFrom:        validFrom,
		DateTo:          dateTo,
		DateActivate:    validFrom,
		EnergyDirection: energyDirection,
		ECPartFact:      req.ParticipationFactor,
		ECShare:         req.ECShare,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
		return
	}

	var mpID *uuid.UUID
	if mp, err := h.mpRepo.GetByZaehlpunkt(r.Context(), req.Zaehlpunkt); err == nil {
		id := mp.ID
		mpID = &id
	}

	factor := req.ParticipationFactor
	nowUTC := time.Now().UTC()
	proc := &domain.EDAProcess{
		EegID:               eegID,
		MeterPointID:        mpID,
		ProcessType:         "EC_PRTFACT_CHG",
		Status:              "pending",
		ConversationID:      convID,
		Zaehlpunkt:          req.Zaehlpunkt,
		ValidFrom:           &validFrom,
		ParticipationFactor: &factor,
		ShareType:           shareType,
		ECDisModel:          ecDisModel,
		EnergyDirection:     energyDirection,
		ECShare:             req.ECShare,
		InitiatedAt:         nowUTC,
	}
	if !dateTo.IsZero() {
		proc.DateTo = &dateTo
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		jsonError(w, "failed to create EDA process record", http.StatusInternalServerError)
		return
	}
	if err := h.jobRepo.EnqueueEDA(r.Context(), "EC_PRTFACT_CHG", eeg.EdaMarktpartnerID, netzbetreiberTo,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}

// zaehlerstandsgangRequest is the body for POST /eda/zaehlerstandsgang.
type zaehlerstandsgangRequest struct {
	Zaehlpunkt string `json:"zaehlpunkt"`
	DateFrom   string `json:"date_from"` // YYYY-MM-DD
	DateTo     string `json:"date_to"`   // YYYY-MM-DD
}

// ZaehlerstandsgangAnfordern godoc
//
//	@Summary		Request historical meter data (EC_REQ_PT)
//	@Description	Sends an EC_REQ_PT request for historical Zählpunktdaten (meter readings) over a given date range. Queues an outbound XML job for the EDA worker.
//	@Tags			EDA
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string						true	"EEG ID (UUID)"
//	@Param			body	body		zaehlerstandsgangRequest	true	"Zählerstandsgang request"
//	@Success		201		{object}	domain.EDAProcess
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/zaehlerstandsgang [post]
func (h *EDAHandler) ZaehlerstandsgangAnfordern(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	if eeg.IsDemo {
		jsonError(w, "EDA-Nachrichten sind im Demo-Modus deaktiviert", http.StatusForbidden)
		return
	}
	if eeg.EdaMarktpartnerID == "" || eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Marktpartner-ID und Netzbetreiber-ID müssen konfiguriert sein", http.StatusBadRequest)
		return
	}

	var req zaehlerstandsgangRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Zaehlpunkt == "" {
		jsonError(w, "zaehlpunkt is required", http.StatusBadRequest)
		return
	}
	if len(req.Zaehlpunkt) >= 8 && req.Zaehlpunkt[:8] != eeg.EdaNetzbetreiberID {
		jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (Präfix: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, req.Zaehlpunkt[:8]), http.StatusBadRequest)
		return
	}
	if req.DateFrom == "" || req.DateTo == "" {
		jsonError(w, "date_from and date_to are required", http.StatusBadRequest)
		return
	}

	// Parse dates in Vienna local time so DateTimeFrom = midnight Vienna, not midnight UTC.
	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	dateFrom, err := time.ParseInLocation("2006-01-02", req.DateFrom, viennaLoc)
	if err != nil {
		jsonError(w, "date_from must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	dateTo, err := time.ParseInLocation("2006-01-02", req.DateTo, viennaLoc)
	if err != nil {
		jsonError(w, "date_to must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if !dateTo.After(dateFrom) {
		jsonError(w, "date_to must be after date_from", http.StatusBadRequest)
		return
	}
	// EVN treats DateTimeTo as exclusive (data ends at the slot *before* this timestamp).
	// Add one day so that date_to="2026-01-31" sends DateTimeTo=2026-02-01T00:00:00+01:00
	// and the response includes all QH slots of January 31.
	dateTo = dateTo.AddDate(0, 0, 1)

	// Derive Netzbetreiber-ID from Zählpunkt prefix; more reliable than the EEG-wide setting.
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		netzbetreiberTo = req.Zaehlpunkt[:8]
	}

	convID := uuid.NewString()
	xmlBody, err := edaxml.BuildAnforderungPT(edaxml.AnforderungPTParams{
		From:           eeg.EdaMarktpartnerID,
		To:             netzbetreiberTo,
		MessageID:      uuid.NewString(),
		ConversationID: convID,
		Zaehlpunkt:     req.Zaehlpunkt,
		DateFrom:       dateFrom,
		DateTo:         dateTo,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
		return
	}

	var mpID *uuid.UUID
	if mp, err := h.mpRepo.GetByZaehlpunkt(r.Context(), req.Zaehlpunkt); err == nil {
		id := mp.ID
		mpID = &id
	}

	now := time.Now()
	proc := &domain.EDAProcess{
		EegID:          eegID,
		MeterPointID:   mpID,
		ProcessType:    "EC_REQ_PT",
		Status:         "pending",
		ConversationID: convID,
		Zaehlpunkt:     req.Zaehlpunkt,
		InitiatedAt:    now,
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		jsonError(w, "failed to create EDA process record", http.StatusInternalServerError)
		return
	}
	if err := h.jobRepo.EnqueueEDA(r.Context(), "EC_REQ_PT", eeg.EdaMarktpartnerID, netzbetreiberTo,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}

// podListRequest is the body for POST /eda/podlist.
// No fields are required — the EEG's ECID and Netzbetreiber are used automatically.
type podListRequest struct{}

// PODList godoc
//
//	@Summary		Request Zählpunktliste (EC_PODLIST)
//	@Description	Sends an ANFORDERUNG_ECP (CPRequest 01.12) to the Netzbetreiber to request the current list of registered meter points (Zählpunktliste) for the energy community. Queues an outbound XML job for the EDA worker.
//	@Tags			EDA
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Success		201		{object}	domain.EDAProcess
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/podlist [post]
func (h *EDAHandler) PODList(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	if eeg.IsDemo {
		jsonError(w, "EDA-Nachrichten sind im Demo-Modus deaktiviert", http.StatusForbidden)
		return
	}
	if eeg.EdaMarktpartnerID == "" || eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Marktpartner-ID und Netzbetreiber-ID müssen konfiguriert sein", http.StatusBadRequest)
		return
	}
	if eeg.GemeinschaftID == "" {
		jsonError(w, "Gemeinschafts-ID (ECID) muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}

	convID := uuid.NewString()
	xmlBody, err := edaxml.BuildPODList(edaxml.PODListParams{
		From:           eeg.EdaMarktpartnerID,
		To:             eeg.EdaNetzbetreiberID,
		MessageID:      uuid.NewString(),
		ConversationID: convID,
		ECID:           eeg.GemeinschaftID,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	proc := &domain.EDAProcess{
		EegID:          eegID,
		ProcessType:    "EC_PODLIST",
		Status:         "pending",
		ConversationID: convID,
		InitiatedAt:    now,
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		jsonError(w, "failed to create EDA process record", http.StatusInternalServerError)
		return
	}
	if err := h.jobRepo.EnqueueEDA(r.Context(), "EC_PODLIST", eeg.EdaMarktpartnerID, eeg.EdaNetzbetreiberID,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}

// widerrufRequest is the body for POST /eda/widerruf.
type widerrufRequest struct {
	Zaehlpunkt string `json:"zaehlpunkt"`
	ConsentEnd string `json:"consent_end"` // YYYY-MM-DD; min today, max +30 Austrian working days
	ReasonKey  int    `json:"reason_key,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// WiderrufEEG godoc
//
//	@Summary		Revoke customer consent (CM_REV_SP)
//	@Description	Sends a CMRevoke 01.10 (AUFHEBUNG_CCMS) to the Netzbetreiber to revoke a previously granted customer consent. Used when a member leaves the EEG.
//	@Tags			EDA
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string			true	"EEG ID (UUID)"
//	@Param			body	body		widerrufRequest	true	"Widerruf request"
//	@Success		201		{object}	domain.EDAProcess
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/widerruf [post]
func (h *EDAHandler) WiderrufEEG(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	if eeg.IsDemo {
		jsonError(w, "EDA-Nachrichten sind im Demo-Modus deaktiviert", http.StatusForbidden)
		return
	}
	if eeg.EdaMarktpartnerID == "" || eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Marktpartner-ID und Netzbetreiber-ID müssen konfiguriert sein", http.StatusBadRequest)
		return
	}

	var req widerrufRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Zaehlpunkt == "" {
		jsonError(w, "zaehlpunkt is required", http.StatusBadRequest)
		return
	}
	if len(req.Zaehlpunkt) >= 8 && req.Zaehlpunkt[:8] != eeg.EdaNetzbetreiberID {
		jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (Präfix: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, req.Zaehlpunkt[:8]), http.StatusBadRequest)
		return
	}
	if req.ConsentEnd == "" {
		jsonError(w, "consent_end is required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	consentEnd, err := time.Parse("2006-01-02", req.ConsentEnd)
	if err != nil {
		jsonError(w, "consent_end must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	// Validate: frühestens Tagesdatum, spätestens +30 Arbeitstage (EDA CM_REV_SP Regelwerk)
	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	nowVienna := time.Now().In(viennaLoc)
	todayUTC := time.Date(nowVienna.Year(), nowVienna.Month(), nowVienna.Day(), 0, 0, 0, 0, time.UTC)
	maxDate := addAustrianWorkingDays(todayUTC, 30)
	if consentEnd.Before(todayUTC) {
		jsonError(w, "consent_end darf nicht in der Vergangenheit liegen", http.StatusBadRequest)
		return
	}
	if consentEnd.After(maxDate) {
		jsonError(w, fmt.Sprintf("consent_end darf höchstens 30 Arbeitstage in der Zukunft liegen (max %s)", maxDate.Format("2006-01-02")), http.StatusBadRequest)
		return
	}

	// Look up meter point to get the stored consent_id.
	mp, err := h.mpRepo.GetByZaehlpunkt(r.Context(), req.Zaehlpunkt)
	if err != nil {
		jsonError(w, fmt.Sprintf("Zählpunkt %s nicht gefunden", req.Zaehlpunkt), http.StatusNotFound)
		return
	}
	if mp.ConsentID == "" {
		jsonError(w, fmt.Sprintf("Zählpunkt %s hat keine gespeicherte Consent-ID — Anmeldung wurde möglicherweise über einen anderen Prozess durchgeführt", req.Zaehlpunkt), http.StatusUnprocessableEntity)
		return
	}

	// Derive Netzbetreiber-ID from Zählpunkt prefix; more reliable than the EEG-wide setting.
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		netzbetreiberTo = req.Zaehlpunkt[:8]
	}

	msgID := uuid.NewString()
	convID := uuid.NewString()

	xmlBody, err := edaxml.BuildCMRevoke(edaxml.CMRevokeParams{
		From:           eeg.EdaMarktpartnerID,
		To:             netzbetreiberTo,
		MessageID:      msgID,
		ConversationID: convID,
		MeteringPoint:  req.Zaehlpunkt,
		ConsentID:      mp.ConsentID,
		ConsentEnd:     consentEnd,
		ReasonKey:      req.ReasonKey,
		Reason:         req.Reason,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
		return
	}

	mpID := mp.ID

	now := time.Now()
	proc := &domain.EDAProcess{
		EegID:          eegID,
		MeterPointID:   &mpID,
		ProcessType:    "CM_REV_SP",
		Status:         "pending",
		ConversationID: convID,
		Zaehlpunkt:     req.Zaehlpunkt,
		InitiatedAt:    now,
		ValidFrom:      &consentEnd,
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		jsonError(w, "failed to create EDA process record", http.StatusInternalServerError)
		return
	}

	if err := h.jobRepo.EnqueueEDA(r.Context(), "CM_REV_SP", eeg.EdaMarktpartnerID, netzbetreiberTo,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}

// ListErrors godoc
//
//	@Summary		List EDA dead-letter errors
//	@Description	Returns EDA messages that failed processing and were moved to the dead-letter (eda_errors) table for manual review.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			limit	query		int		false	"Maximum number of entries to return (default 50, max 500)"
//	@Success		200		{array}		domain.EDAError
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/errors [get]
func (h *EDAHandler) ListErrors(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	errs, err := h.edaErrorRepo.ListByEEG(r.Context(), eegID, limit)
	if err != nil {
		jsonError(w, "failed to list EDA errors", http.StatusInternalServerError)
		return
	}
	if errs == nil {
		errs = []domain.EDAError{}
	}
	jsonOK(w, errs)
}

// GetWorkerStatus godoc
//
//	@Summary		Get EDA worker status
//	@Description	Returns the last-known status of the EDA worker (transport mode, last poll time, last error). Returns an empty status object if the worker has never run.
//	@Tags			EDA
//	@Produce		json
//	@Success		200	{object}	domain.EDAWorkerStatus
//	@Failure		500	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eda/worker-status [get]
func (h *EDAHandler) GetWorkerStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.workerStatusRepo.Get(r.Context())
	if err != nil {
		// Row may not exist yet if worker has never run — return empty status.
		jsonOK(w, &domain.EDAWorkerStatus{})
		return
	}
	jsonOK(w, status)
}

// PollNow godoc
//
//	@Summary		Trigger immediate EDA worker poll
//	@Description	Proxies a poll-now trigger to the EDA worker HTTP server. Requires EDA_WORKER_URL to be configured. Returns 503 if the worker is not configured or unreachable.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Success		200		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Failure		503		{object}	map[string]string	"EDA worker not configured or unreachable"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/poll-now [post]
func (h *EDAHandler) PollNow(w http.ResponseWriter, r *http.Request) {
	if h.edaWorkerURL == "" {
		jsonError(w, "EDA worker not configured (EDA_WORKER_URL not set)", http.StatusServiceUnavailable)
		return
	}
	url := h.edaWorkerURL + "/eda/poll-now"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, nil)
	if err != nil {
		jsonError(w, "failed to build request to EDA worker", http.StatusInternalServerError)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, "EDA worker unreachable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
