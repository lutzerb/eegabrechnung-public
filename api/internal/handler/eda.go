package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed IANA timezone database (required in Alpine containers)

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/netzbetreiber"
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
	readingRepo      *repository.ReadingRepository
	edaWorkerURL     string
}

func NewEDAHandler(
	eegRepo *repository.EEGRepository,
	mpRepo *repository.MeterPointRepository,
	edaProcRepo *repository.EDAProcessRepository,
	jobRepo *repository.JobRepository,
	edaErrorRepo *repository.EDAErrorRepository,
	workerStatusRepo *repository.EDAWorkerStatusRepository,
	readingRepo *repository.ReadingRepository,
	edaWorkerURL string,
) *EDAHandler {
	return &EDAHandler{
		eegRepo:          eegRepo,
		mpRepo:           mpRepo,
		edaProcRepo:      edaProcRepo,
		jobRepo:          jobRepo,
		edaErrorRepo:     edaErrorRepo,
		workerStatusRepo: workerStatusRepo,
		readingRepo:      readingRepo,
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

// activeNetzbetreiberResponse is one entry of GET /eegs/{eegID}/eda/netzbetreiber.
type activeNetzbetreiberResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Unresolved bool   `json:"unresolved"`
}

// ListActiveNetzbetreiber godoc
//
//	@Summary		List currently active Netzbetreiber
//	@Description	Derives the distinct set of Netzbetreiber from the EEG's currently active meter points (Zählpunkt prefix). Primarily useful for BEG communities, which can span multiple grid operators.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Success		200		{array}		activeNetzbetreiberResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/netzbetreiber [get]
func (h *EDAHandler) ListActiveNetzbetreiber(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	mps, err := h.mpRepo.ListByEeg(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to list meter points", http.StatusInternalServerError)
		return
	}
	infos := netzbetreiber.ActiveFromMeterPoints(mps)
	result := make([]activeNetzbetreiberResponse, len(infos))
	for i, info := range infos {
		result[i] = activeNetzbetreiberResponse{ID: info.ID, Name: info.Name, Unresolved: info.Unresolved}
	}
	jsonOK(w, result)
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
	if eeg.EdaMarktpartnerID == "" {
		jsonError(w, "EDA Marktpartner-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}
	if eeg.GemeinschaftTyp != "BEG" && eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Netzbetreiber-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
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
	if req.ParticipationFactor <= 0 || req.ParticipationFactor > 100 {
		jsonError(w, "participation_factor must be between 0 and 100", http.StatusBadRequest)
		return
	}

	if active, err := h.edaProcRepo.HasActiveAnmeldung(r.Context(), eegID, req.Zaehlpunkt); err == nil && active {
		jsonError(w, "Für diesen Zählpunkt läuft bereits eine Anmeldung oder sie ist bereits bestätigt — bitte Status abwarten statt erneut anzumelden", http.StatusConflict)
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

	// Resolve the actual Netzbetreiber-ID from the Zählpunkt: the raw 8-char
	// prefix is not always the correct routing target (some regional
	// operators reuse historical sub-area codes in Zählpunkten that were
	// never registered as their own Marktpartner-ID — see prefixOverrides).
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		resolved, ok := netzbetreiber.ResolveRoutingID(req.Zaehlpunkt)
		if !ok {
			jsonError(w, fmt.Sprintf("Unbekannter Netzbetreiber-Code für Zählpunkt %s (Präfix %s) — bitte in der Netzbetreiber-Registry oder den Ausnahme-Zuordnungen ergänzen (api/internal/netzbetreiber/lookup.go)", req.Zaehlpunkt, req.Zaehlpunkt[:8]), http.StatusUnprocessableEntity)
			return
		}
		if eeg.GemeinschaftTyp != "BEG" && resolved != eeg.EdaNetzbetreiberID {
			jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (aufgelöste ID: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, resolved), http.StatusBadRequest)
			return
		}
		netzbetreiberTo = resolved
	}

	msgID := uuid.NewString()
	convID := uuid.NewString()

	// Resolve meter_point_id if available (best effort), scoped to THIS EEG — a
	// Zählpunkt string can be simultaneously active in a different EEG (Mehrfachteilnahme
	// modeled as separate meter_points rows), so an unscoped lookup could resolve to
	// another tenant's row. GetLatestByZaehlpunktInEEG covers both the active case and
	// re-registration after a deregistration (e.g. Zählpunkt-Lieferantenwechsel).
	var mpID *uuid.UUID
	var storedDirection string
	if mp, err := h.mpRepo.GetLatestByZaehlpunktInEEG(r.Context(), eegID, req.Zaehlpunkt); err == nil {
		id := mp.ID
		mpID = &id
		storedDirection = mp.Energierichtung
	}

	// The meter point's own Energierichtung is authoritative — it must never
	// silently diverge from what was registered with the Netzbetreiber for this
	// Zählpunkt. Only fall back to the client-supplied value (and finally to
	// CONSUMPTION) when no meter point row exists yet to read it from.
	energyDirection := storedDirection
	if energyDirection == "" {
		energyDirection = req.EnergyDirection
	}
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
	proc.ParticipationFactor = &req.ParticipationFactor
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
	ECDisModel          string   `json:"ec_dis_model"`         // deprecated/ignored — the EEG-wide eda_dis_model setting is used
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
	if eeg.EdaMarktpartnerID == "" {
		jsonError(w, "EDA Marktpartner-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}
	if eeg.GemeinschaftTyp != "BEG" && eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Netzbetreiber-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
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

	// ECDisModel is declared once for the whole community (eegs.eda_dis_model,
	// migration 081) — always take it from the EEG settings. The request field
	// is deliberately ignored so different UI paths cannot report contradicting
	// Verteilungsmodelle to the Netzbetreiber.
	ecDisModel := eeg.EdaDisModel
	if ecDisModel == "" {
		ecDisModel = "D"
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

	// Resolve the actual Netzbetreiber-ID from the Zählpunkt: the raw 8-char
	// prefix is not always the correct routing target (some regional
	// operators reuse historical sub-area codes in Zählpunkten that were
	// never registered as their own Marktpartner-ID — see prefixOverrides).
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		resolved, ok := netzbetreiber.ResolveRoutingID(req.Zaehlpunkt)
		if !ok {
			jsonError(w, fmt.Sprintf("Unbekannter Netzbetreiber-Code für Zählpunkt %s (Präfix %s) — bitte in der Netzbetreiber-Registry oder den Ausnahme-Zuordnungen ergänzen (api/internal/netzbetreiber/lookup.go)", req.Zaehlpunkt, req.Zaehlpunkt[:8]), http.StatusUnprocessableEntity)
			return
		}
		if eeg.GemeinschaftTyp != "BEG" && resolved != eeg.EdaNetzbetreiberID {
			jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (aufgelöste ID: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, resolved), http.StatusBadRequest)
			return
		}
		netzbetreiberTo = resolved
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

	// Scoped to THIS EEG — see comment in Anmeldung above on why an unscoped lookup
	// is unsafe when the same Zählpunkt is active in another EEG.
	var mpID *uuid.UUID
	if mp, err := h.mpRepo.GetLatestByZaehlpunktInEEG(r.Context(), eegID, req.Zaehlpunkt); err == nil {
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
//	@Summary		Request historical meter data (CR_REQ_PT)
//	@Description	Sends an CR_REQ_PT request for historical Zählpunktdaten (meter readings) over a given date range. Queues an outbound XML job for the EDA worker.
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
	if eeg.EdaMarktpartnerID == "" {
		jsonError(w, "EDA Marktpartner-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}
	if eeg.GemeinschaftTyp != "BEG" && eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Netzbetreiber-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
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
	if dateTo.Before(dateFrom) {
		jsonError(w, "date_to must not be before date_from", http.StatusBadRequest)
		return
	}
	// EVN treats DateTimeTo as exclusive (data ends at the slot *before* this timestamp).
	// Add one day so that date_to="2026-01-31" sends DateTimeTo=2026-02-01T00:00:00+01:00
	// and the response includes all QH slots of January 31.
	dateTo = dateTo.AddDate(0, 0, 1)

	// Resolve the actual Netzbetreiber-ID from the Zählpunkt: the raw 8-char
	// prefix is not always the correct routing target (some regional
	// operators reuse historical sub-area codes in Zählpunkten that were
	// never registered as their own Marktpartner-ID — see prefixOverrides).
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		resolved, ok := netzbetreiber.ResolveRoutingID(req.Zaehlpunkt)
		if !ok {
			jsonError(w, fmt.Sprintf("Unbekannter Netzbetreiber-Code für Zählpunkt %s (Präfix %s) — bitte in der Netzbetreiber-Registry oder den Ausnahme-Zuordnungen ergänzen (api/internal/netzbetreiber/lookup.go)", req.Zaehlpunkt, req.Zaehlpunkt[:8]), http.StatusUnprocessableEntity)
			return
		}
		if eeg.GemeinschaftTyp != "BEG" && resolved != eeg.EdaNetzbetreiberID {
			jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (aufgelöste ID: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, resolved), http.StatusBadRequest)
			return
		}
		netzbetreiberTo = resolved
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

	// Scoped to THIS EEG — see comment in Anmeldung above on why an unscoped lookup
	// is unsafe when the same Zählpunkt is active in another EEG.
	var mpID *uuid.UUID
	if mp, err := h.mpRepo.GetLatestByZaehlpunktInEEG(r.Context(), eegID, req.Zaehlpunkt); err == nil {
		id := mp.ID
		mpID = &id
	}

	now := time.Now()
	proc := &domain.EDAProcess{
		EegID:          eegID,
		MeterPointID:   mpID,
		ProcessType:    "CR_REQ_PT",
		Status:         "pending",
		ConversationID: convID,
		Zaehlpunkt:     req.Zaehlpunkt,
		InitiatedAt:    now,
	}
	if err := h.edaProcRepo.Create(r.Context(), proc); err != nil {
		jsonError(w, "failed to create EDA process record", http.StatusInternalServerError)
		return
	}
	if err := h.jobRepo.EnqueueEDA(r.Context(), "CR_REQ_PT", eeg.EdaMarktpartnerID, netzbetreiberTo,
		eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
		jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, proc)
}

// ZaehlerstandsgangPerioden godoc
//
//	@Summary		List registration periods for a Zählpunkt
//	@Description	Returns the full Anmeldung/Abmeldung history (meter_point_registration_periods) for a raw Zählpunkt string, oldest first. Used by the "Nur aktive Perioden anfordern" option on CR_REQ_PT — works for any Zählpunkt (free-typed or from the picker), not just ones with a local meter_points row.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID		path	string	true	"EEG ID (UUID)"
//	@Param			zaehlpunkt	query	string	true	"Zählpunkt ID"
//	@Success		200	{array}		domain.MeterPointRegistrationPeriod
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/zaehlerstandsgang/perioden [get]
func (h *EDAHandler) ZaehlerstandsgangPerioden(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	zp := strings.TrimSpace(r.URL.Query().Get("zaehlpunkt"))
	if zp == "" {
		jsonError(w, "zaehlpunkt query parameter is required", http.StatusBadRequest)
		return
	}
	periods, err := h.mpRepo.ListRegistrationHistory(r.Context(), eeg.ID, zp)
	if err != nil {
		jsonError(w, "failed to load registration periods", http.StatusInternalServerError)
		return
	}
	if periods == nil {
		periods = []domain.MeterPointRegistrationPeriod{}
	}
	jsonOK(w, periods)
}

// fehlendeDatenPreviewItem is one Zählpunkt/period entry of the missing-data preview.
type fehlendeDatenPreviewItem struct {
	Zaehlpunkt      string                        `json:"zaehlpunkt"`
	MemberName      string                        `json:"member_name"`
	PeriodID        uuid.UUID                     `json:"period_id"`
	RegistriertSeit string                        `json:"registriert_seit"`
	AbgemeldetAm    *string                       `json:"abgemeldet_am,omitempty"`
	MissingRanges   []repository.CategorizedRange `json:"missing_ranges"`
	InFlight        bool                          `json:"in_flight"`
}

// fehlendeDatenPreviewResponse is the body of GET /eda/zaehlerstandsgang/fehlende-daten.
type fehlendeDatenPreviewResponse struct {
	Items       []fehlendeDatenPreviewItem `json:"items"`
	TotalRanges int                        `json:"total_ranges"`
}

// FehlendeDatenPreview godoc
//
//	@Summary		Preview missing CR_REQ_PT data across all Zählpunkte/periods
//	@Description	For every registration period of every Zählpunkt ever registered in this EEG (meter_point_registration_periods, regardless of current member status), finds contiguous date ranges lacking a full day of L1/L2 energy_readings, and flags whether the Zählpunkt has an in-flight (pending/sent) CR_REQ_PT process. Read-only — does not send anything.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID	path	string	true	"EEG ID (UUID)"
//	@Success		200	{object}	fehlendeDatenPreviewResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/zaehlerstandsgang/fehlende-daten [get]
func (h *EDAHandler) FehlendeDatenPreview(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	now := time.Now().In(viennaLoc)
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, viennaLoc)

	gaps, err := h.readingRepo.MissingDataByRegistrationPeriod(r.Context(), eeg.ID, todayMidnight)
	if err != nil {
		jsonError(w, "failed to compute missing data", http.StatusInternalServerError)
		return
	}
	inFlight, err := h.edaProcRepo.ListZaehlpunkteWithInFlightCRReqPT(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to check in-flight requests", http.StatusInternalServerError)
		return
	}

	items := make([]fehlendeDatenPreviewItem, 0, len(gaps))
	total := 0
	for _, g := range gaps {
		items = append(items, fehlendeDatenPreviewItem{
			Zaehlpunkt: g.Zaehlpunkt, MemberName: g.MemberName, PeriodID: g.PeriodID, RegistriertSeit: g.RegistriertSeit,
			AbgemeldetAm: g.AbgemeldetAm, MissingRanges: g.MissingRanges,
			InFlight: inFlight[g.Zaehlpunkt],
		})
		total += len(g.MissingRanges)
	}
	jsonOK(w, fehlendeDatenPreviewResponse{Items: items, TotalRanges: total})
}

// podListRequest is the body for POST /eda/podlist.
// No fields are required — the EEG's ECID and Netzbetreiber are used automatically.
// date_from/date_to are optional (YYYY-MM-DD): per the ebutilities CPRequest schema
// they carry the desired period for the Zählpunktliste snapshot. Some Netzbetreiber
// (e.g. Netz NÖ) tolerate omitting them and default to "current status", others
// (e.g. Energienetze Steiermark) appear to require them — omitted, they reject with
// response code 181 ("Gemeinschafts-ID nicht vorhanden").
type podListRequest struct {
	DateFrom string `json:"date_from,omitempty"`
	DateTo   string `json:"date_to,omitempty"`
}

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
	if eeg.EdaMarktpartnerID == "" {
		jsonError(w, "EDA Marktpartner-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}
	if eeg.GemeinschaftID == "" {
		jsonError(w, "Gemeinschafts-ID (ECID) muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}

	var req podListRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional; ignore decode errors on empty body

	var dateFrom, dateTo time.Time
	if req.DateFrom != "" || req.DateTo != "" {
		if req.DateFrom == "" || req.DateTo == "" {
			jsonError(w, "date_from and date_to must both be set (or both omitted)", http.StatusBadRequest)
			return
		}
		viennaLoc, _ := time.LoadLocation("Europe/Vienna")
		var err error
		dateFrom, err = time.ParseInLocation("2006-01-02", req.DateFrom, viennaLoc)
		if err != nil {
			jsonError(w, "date_from must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		dateTo, err = time.ParseInLocation("2006-01-02", req.DateTo, viennaLoc)
		if err != nil {
			jsonError(w, "date_to must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		if dateTo.Before(dateFrom) {
			jsonError(w, "date_to must not be before date_from", http.StatusBadRequest)
			return
		}
		// Exclusive upper bound, same convention as CR_REQ_PT.
		dateTo = dateTo.AddDate(0, 0, 1)
	}

	// For BEG: send one PODLIST per distinct Netzbetreiber derived from active meter point prefixes.
	// For EEG/GEA: send one PODLIST to the configured Netzbetreiber.
	var nbTargets []string
	if eeg.GemeinschaftTyp == "BEG" {
		mps, err := h.mpRepo.ListByEeg(r.Context(), eegID)
		if err != nil {
			jsonError(w, "failed to list meter points", http.StatusInternalServerError)
			return
		}
		var unresolved []string
		for _, info := range netzbetreiber.ActiveFromMeterPoints(mps) {
			if info.Unresolved {
				unresolved = append(unresolved, info.ID)
				continue
			}
			nbTargets = append(nbTargets, info.ID)
		}
		if len(unresolved) > 0 {
			jsonError(w, fmt.Sprintf("Unbekannter Netzbetreiber-Code für Zählpunkt-Präfix(e) %s — bitte in der Netzbetreiber-Registry oder den Ausnahme-Zuordnungen ergänzen (api/internal/netzbetreiber/lookup.go)", strings.Join(unresolved, ", ")), http.StatusUnprocessableEntity)
			return
		}
		if len(nbTargets) == 0 && eeg.EdaNetzbetreiberID != "" {
			nbTargets = []string{eeg.EdaNetzbetreiberID}
		}
	} else {
		if eeg.EdaNetzbetreiberID == "" {
			jsonError(w, "EDA Netzbetreiber-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
			return
		}
		nbTargets = []string{eeg.EdaNetzbetreiberID}
	}
	if len(nbTargets) == 0 {
		jsonError(w, "Kein Netzbetreiber konfiguriert und keine aktiven Zählpunkte vorhanden", http.StatusBadRequest)
		return
	}

	now := time.Now()
	var lastProc *domain.EDAProcess
	for _, nb := range nbTargets {
		convID := uuid.NewString()
		xmlBody, err := edaxml.BuildPODList(edaxml.PODListParams{
			From:           eeg.EdaMarktpartnerID,
			To:             nb,
			MessageID:      uuid.NewString(),
			ConversationID: convID,
			ECID:           eeg.GemeinschaftID,
			DateFrom:       dateFrom,
			DateTo:         dateTo,
		})
		if err != nil {
			jsonError(w, fmt.Sprintf("build XML: %v", err), http.StatusInternalServerError)
			return
		}
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
		if err := h.jobRepo.EnqueueEDA(r.Context(), "EC_PODLIST", eeg.EdaMarktpartnerID, nb,
			eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
			jsonError(w, "failed to queue EDA job", http.StatusInternalServerError)
			return
		}
		lastProc = proc
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, lastProc)
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
	if eeg.EdaMarktpartnerID == "" {
		jsonError(w, "EDA Marktpartner-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
		return
	}
	if eeg.GemeinschaftTyp != "BEG" && eeg.EdaNetzbetreiberID == "" {
		jsonError(w, "EDA Netzbetreiber-ID muss in den EEG-Einstellungen konfiguriert sein", http.StatusBadRequest)
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

	// Look up meter point to get the stored consent_id — scoped to THIS EEG, since an
	// unscoped lookup could resolve to a different tenant's active row for the same
	// Zählpunkt string and revoke consent that isn't ours to revoke.
	mp, err := h.mpRepo.GetLatestByZaehlpunktInEEG(r.Context(), eegID, req.Zaehlpunkt)
	if err != nil {
		jsonError(w, fmt.Sprintf("Zählpunkt %s nicht gefunden", req.Zaehlpunkt), http.StatusNotFound)
		return
	}
	if mp.ConsentID == "" {
		jsonError(w, fmt.Sprintf("Zählpunkt %s hat keine gespeicherte Consent-ID — Anmeldung wurde möglicherweise über einen anderen Prozess durchgeführt", req.Zaehlpunkt), http.StatusUnprocessableEntity)
		return
	}

	// Resolve the actual Netzbetreiber-ID from the Zählpunkt: the raw 8-char
	// prefix is not always the correct routing target (some regional
	// operators reuse historical sub-area codes in Zählpunkten that were
	// never registered as their own Marktpartner-ID — see prefixOverrides).
	netzbetreiberTo := eeg.EdaNetzbetreiberID
	if len(req.Zaehlpunkt) >= 8 {
		resolved, ok := netzbetreiber.ResolveRoutingID(req.Zaehlpunkt)
		if !ok {
			jsonError(w, fmt.Sprintf("Unbekannter Netzbetreiber-Code für Zählpunkt %s (Präfix %s) — bitte in der Netzbetreiber-Registry oder den Ausnahme-Zuordnungen ergänzen (api/internal/netzbetreiber/lookup.go)", req.Zaehlpunkt, req.Zaehlpunkt[:8]), http.StatusUnprocessableEntity)
			return
		}
		if eeg.GemeinschaftTyp != "BEG" && resolved != eeg.EdaNetzbetreiberID {
			jsonError(w, fmt.Sprintf("Zählpunkt %s passt nicht zum konfigurierten Netzbetreiber %s (aufgelöste ID: %s)", req.Zaehlpunkt, eeg.EdaNetzbetreiberID, resolved), http.StatusBadRequest)
			return
		}
		netzbetreiberTo = resolved
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
