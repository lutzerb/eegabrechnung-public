package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type MeterPointHandler struct {
	meterPointRepo *repository.MeterPointRepository
	memberRepo     *repository.MemberRepository
	eegRepo        *repository.EEGRepository
	edaProcRepo    *repository.EDAProcessRepository
	readingRepo    *repository.ReadingRepository
	edaMsgRepo     *repository.EDAMessageRepository
	obisCache      *obisIndexCache
}

func NewMeterPointHandler(meterPointRepo *repository.MeterPointRepository, memberRepo *repository.MemberRepository, eegRepo *repository.EEGRepository, edaProcRepo *repository.EDAProcessRepository, readingRepo *repository.ReadingRepository, edaMsgRepo *repository.EDAMessageRepository) *MeterPointHandler {
	return &MeterPointHandler{
		meterPointRepo: meterPointRepo,
		memberRepo:     memberRepo,
		eegRepo:        eegRepo,
		edaProcRepo:    edaProcRepo,
		readingRepo:    readingRepo,
		edaMsgRepo:     edaMsgRepo,
		obisCache:      newOBISIndexCache(),
	}
}

type meterPointRequest struct {
	Zaehlpunkt          string  `json:"zaehlpunkt"`
	Energierichtung     string  `json:"energierichtung"`
	Verteilungsmodell   string  `json:"verteilungsmodell"`
	ZugeteilteMenugePct float64 `json:"zugeteilte_menge_pct"`
	Status              string  `json:"status"`
	RegistriertSeit     string  `json:"registriert_seit"`
	AbgemeldetAm        string  `json:"abgemeldet_am"` // manually settable for non-EDA EEGs; "" = keep existing, "clear" = set to NULL
	GenerationType      string  `json:"generation_type"` // PV | Windkraft | Wasserkraft
	Notes               string  `json:"notes"`
}

// CreateMeterPoint godoc
// @Summary     Create meter point
// @Description Adds a new meter point (Zählpunkt) to a member. zaehlpunkt and energierichtung are required. verteilungsmodell defaults to DYNAMIC; status defaults to ACTIVATED.
// @Tags        Zählpunkte
// @Accept      json
// @Produce     json
// @Param       eegID     path      string             true  "EEG UUID"
// @Param       memberID  path      string             true  "Member UUID"
// @Param       mp        body      meterPointRequest  true  "Meter point data (zaehlpunkt and energierichtung required)"
// @Success     201  {object}  domain.MeterPoint  "Created meter point"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/members/{memberID}/meter-points [post]
// CreateMeterPoint handles POST /eegs/{eegID}/members/{memberID}/meter-points
func (h *MeterPointHandler) CreateMeterPoint(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	eegID := eeg.ID
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	// Ensure the target member actually belongs to this EEG before attaching a meter point.
	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil || member.EegID != eegID {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}

	var req meterPointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Zaehlpunkt == "" {
		jsonError(w, "zaehlpunkt is required", http.StatusBadRequest)
		return
	}
	if req.Energierichtung == "" {
		jsonError(w, "energierichtung is required", http.StatusBadRequest)
		return
	}
	if req.Verteilungsmodell == "" {
		req.Verteilungsmodell = "DYNAMIC"
	}
	if req.Status == "" {
		req.Status = "ACTIVATED"
	}

	mp := &domain.MeterPoint{
		MemberID:            memberID,
		EegID:               eegID,
		Zaehlpunkt:          req.Zaehlpunkt,
		Energierichtung:     req.Energierichtung,
		Verteilungsmodell:   req.Verteilungsmodell,
		ZugeteilteMenugePct: req.ZugeteilteMenugePct,
		Status:              req.Status,
	}
	if req.GenerationType != "" {
		mp.GenerationType = &req.GenerationType
	}
	if req.RegistriertSeit != "" {
		t, err := time.Parse("2006-01-02", req.RegistriertSeit)
		if err != nil {
			jsonError(w, "invalid registriert_seit format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		mp.RegistriertSeit = &t
	}

	if err := h.meterPointRepo.Create(r.Context(), mp); err != nil {
		jsonError(w, "failed to create meter point", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, mp)
}

// GetMeterPoint godoc
// @Summary     Get meter point
// @Description Returns a single meter point by its UUID.
// @Tags        Zählpunkte
// @Produce     json
// @Param       eegID         path      string  true  "EEG UUID"
// @Param       meterPointID  path      string  true  "Meter point UUID"
// @Success     200  {object}  domain.MeterPoint  "Meter point"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/meter-points/{meterPointID} [get]
// GetMeterPoint handles GET /eegs/{eegID}/meter-points/{meterPointID}
func (h *MeterPointHandler) GetMeterPoint(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	meterPointID, err := uuid.Parse(chi.URLParam(r, "meterPointID"))
	if err != nil {
		jsonError(w, "invalid meter point ID", http.StatusBadRequest)
		return
	}

	mp, err := h.meterPointRepo.GetByID(r.Context(), meterPointID)
	if err != nil || mp.EegID != eeg.ID {
		jsonError(w, "meter point not found", http.StatusNotFound)
		return
	}
	jsonOK(w, mp)
}

// UpdateMeterPoint godoc
// @Summary     Update meter point
// @Description Updates an existing meter point's attributes. Provided non-empty fields overwrite existing values. generation_type is cleared when sent as an empty string.
// @Tags        Zählpunkte
// @Accept      json
// @Produce     json
// @Param       eegID         path      string             true  "EEG UUID"
// @Param       meterPointID  path      string             true  "Meter point UUID"
// @Param       mp            body      meterPointRequest  true  "Meter point update data"
// @Success     200  {object}  domain.MeterPoint  "Updated meter point"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/meter-points/{meterPointID} [put]
// UpdateMeterPoint handles PUT /eegs/{eegID}/meter-points/{meterPointID}
func (h *MeterPointHandler) UpdateMeterPoint(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	meterPointID, err := uuid.Parse(chi.URLParam(r, "meterPointID"))
	if err != nil {
		jsonError(w, "invalid meter point ID", http.StatusBadRequest)
		return
	}

	existing, err := h.meterPointRepo.GetByID(r.Context(), meterPointID)
	if err != nil || existing.EegID != eeg.ID {
		jsonError(w, "meter point not found", http.StatusNotFound)
		return
	}

	var req meterPointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Energierichtung != "" {
		existing.Energierichtung = req.Energierichtung
	}
	if req.Verteilungsmodell != "" {
		existing.Verteilungsmodell = req.Verteilungsmodell
	}
	existing.ZugeteilteMenugePct = req.ZugeteilteMenugePct
	if req.Status != "" {
		existing.Status = req.Status
	}
	if req.GenerationType != "" {
		existing.GenerationType = &req.GenerationType
	} else {
		existing.GenerationType = nil
	}
	existing.Notes = req.Notes
	previousAbgemeldetAm := existing.AbgemeldetAm
	if req.AbgemeldetAm == "clear" {
		existing.AbgemeldetAm = nil
	} else if req.AbgemeldetAm != "" {
		t, err := time.Parse("2006-01-02", req.AbgemeldetAm)
		if err != nil {
			jsonError(w, "invalid abgemeldet_am format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		existing.AbgemeldetAm = &t
	}

	if err := h.meterPointRepo.Update(r.Context(), existing); err != nil {
		jsonError(w, "failed to update meter point", http.StatusInternalServerError)
		return
	}

	// Keep the registration-period history in sync with manual abgemeldet_am edits.
	if previousAbgemeldetAm == nil && existing.AbgemeldetAm != nil {
		if err := h.meterPointRepo.ClosePeriodManual(r.Context(), existing.EegID, existing.Zaehlpunkt, *existing.AbgemeldetAm, "Manuell gesetzt"); err != nil {
			jsonError(w, "failed to update registration history", http.StatusInternalServerError)
			return
		}
	} else if previousAbgemeldetAm != nil && existing.AbgemeldetAm == nil {
		if err := h.meterPointRepo.ReopenPeriodManual(r.Context(), existing.EegID, existing.Zaehlpunkt); err != nil {
			jsonError(w, "failed to update registration history", http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, existing)
}

// meterPointHistoryResponse combines confirmed registration periods with the
// individual EDA process attempts (sent/confirmed/rejected/error) that produced them,
// so the frontend can render one merged timeline instead of only the confirmed state.
type meterPointHistoryResponse struct {
	Periods   []domain.MeterPointRegistrationPeriod `json:"periods"`
	Processes []domain.EDAProcess                   `json:"processes"`
}

// GetMeterPointHistory godoc
// @Summary     Get meter point registration history
// @Description Returns the full Anmeldung/Abmeldung history for this meter point's Zählpunkt: confirmed periods plus the individual EDA process attempts (sent, confirmed, rejected, error), oldest first.
// @Tags        Zählpunkte
// @Produce     json
// @Param       eegID         path  string  true  "EEG UUID"
// @Param       meterPointID  path  string  true  "Meter point UUID"
// @Success     200  {object}  meterPointHistoryResponse
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/meter-points/{meterPointID}/history [get]
// GetMeterPointHistory handles GET /eegs/{eegID}/meter-points/{meterPointID}/history
func (h *MeterPointHandler) GetMeterPointHistory(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	meterPointID, err := uuid.Parse(chi.URLParam(r, "meterPointID"))
	if err != nil {
		jsonError(w, "invalid meter point ID", http.StatusBadRequest)
		return
	}

	mp, err := h.meterPointRepo.GetByID(r.Context(), meterPointID)
	if err != nil || mp.EegID != eeg.ID {
		jsonError(w, "meter point not found", http.StatusNotFound)
		return
	}

	periods, err := h.meterPointRepo.ListRegistrationHistory(r.Context(), mp.EegID, mp.Zaehlpunkt)
	if err != nil {
		jsonError(w, "failed to load registration history", http.StatusInternalServerError)
		return
	}
	processes, err := h.edaProcRepo.ListByZaehlpunkt(r.Context(), mp.EegID, mp.Zaehlpunkt)
	if err != nil {
		jsonError(w, "failed to load registration history", http.StatusInternalServerError)
		return
	}
	jsonOK(w, meterPointHistoryResponse{Periods: periods, Processes: processes})
}

// meterPointReadingRow is one energy_readings row enriched with the raw OBIS-code
// breakdown for its timestamp (parsed live from the matching EDA messages — see
// obis_index.go). OBIS is empty for xlsx-sourced rows or timestamps not covered by
// any cached EDA message.
type meterPointReadingRow struct {
	domain.EnergyReading
	OBIS []OBISEntry `json:"obis"`
}

type meterPointReadingsResponse struct {
	Readings   []meterPointReadingRow `json:"readings"`
	TotalCount int                     `json:"total_count"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
}

const (
	meterPointReadingsDefaultLimit = 100
	meterPointReadingsMaxLimit     = 500
)

// parseReadingsDebugParam parses a "from"/"to" query value for GetMeterPointReadings.
// Accepts either a full datetime ("2006-01-02T15:04", as produced by an HTML
// datetime-local input) or a bare date ("2006-01-02"); both are interpreted in Vienna
// local time, matching the EDA/reading conventions used throughout the codebase.
// A bare date given as the (inclusive) upper bound is extended to end-of-day so a
// single-day filter behaves intuitively; an explicit datetime is used exactly as given.
// Returns the zero time.Time (no bound) when v is empty or unparseable.
func parseReadingsDebugParam(v string, isUpperBound bool, loc *time.Location) time.Time {
	if v == "" {
		return time.Time{}
	}
	if parsed, err := time.ParseInLocation("2006-01-02T15:04", v, loc); err == nil {
		return parsed
	}
	if parsed, err := time.ParseInLocation("2006-01-02", v, loc); err == nil {
		if isUpperBound {
			return parsed.Add(24*time.Hour - time.Nanosecond)
		}
		return parsed
	}
	return time.Time{}
}

// GetMeterPointReadings godoc
// @Summary     List raw energy readings for one meter point (admin debug view)
// @Description Returns paginated energy_readings rows for one meter point, each enriched with
// @Description the individual OBIS-code values/quality parsed live from the matching raw EDA
// @Description messages (energy_readings only stores one merged "worst quality" per slot).
// @Tags        Zählpunkte
// @Produce     json
// @Param       eegID         path      string  true   "EEG UUID"
// @Param       meterPointID  path      string  true   "Meter point UUID"
// @Param       from          query     string  false  "Start datetime (YYYY-MM-DDTHH:MM) or date (YYYY-MM-DD), Vienna time, inclusive"
// @Param       to            query     string  false  "End datetime (YYYY-MM-DDTHH:MM) or date (YYYY-MM-DD), Vienna time, inclusive (a bare date is extended to end-of-day)"
// @Param       limit         query     int     false  "Page size, default 100, max 500"
// @Param       offset        query     int     false  "Page offset, default 0"
// @Success     200  {object}  meterPointReadingsResponse
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Meter point not found"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/meter-points/{meterPointID}/readings [get]
// GetMeterPointReadings handles GET /eegs/{eegID}/meter-points/{meterPointID}/readings
func (h *MeterPointHandler) GetMeterPointReadings(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	meterPointID, err := uuid.Parse(chi.URLParam(r, "meterPointID"))
	if err != nil {
		jsonError(w, "invalid meter point ID", http.StatusBadRequest)
		return
	}

	mp, err := h.meterPointRepo.GetByID(r.Context(), meterPointID)
	if err != nil || mp.EegID != eeg.ID {
		jsonError(w, "meter point not found", http.StatusNotFound)
		return
	}

	q := r.URL.Query()
	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	from := parseReadingsDebugParam(q.Get("from"), false, viennaLoc)
	to := parseReadingsDebugParam(q.Get("to"), true, viennaLoc)
	limit := meterPointReadingsDefaultLimit
	if v := q.Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > meterPointReadingsMaxLimit {
		limit = meterPointReadingsMaxLimit
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	total, err := h.readingRepo.CountByMeterPoint(r.Context(), meterPointID, from, to)
	if err != nil {
		jsonError(w, "failed to count readings", http.StatusInternalServerError)
		return
	}
	readings, err := h.readingRepo.ListByMeterPoint(r.Context(), meterPointID, from, to, limit, offset)
	if err != nil {
		jsonError(w, "failed to load readings", http.StatusInternalServerError)
		return
	}

	obisIndex, err := h.obisCache.get(r.Context(), h.edaMsgRepo, eeg.ID, mp.Zaehlpunkt)
	if err != nil {
		// OBIS enrichment is best-effort — a broken/unavailable index must not hide
		// the underlying energy_readings data, which is the primary source of truth.
		obisIndex = nil
	}

	rows := make([]meterPointReadingRow, len(readings))
	for i, rd := range readings {
		obis := obisIndex[rd.Ts.Unix()]
		if obis == nil {
			obis = []OBISEntry{}
		}
		rows[i] = meterPointReadingRow{EnergyReading: rd, OBIS: obis}
	}

	jsonOK(w, meterPointReadingsResponse{Readings: rows, TotalCount: total, Limit: limit, Offset: offset})
}

// DeleteMeterPoint godoc
// @Summary     Delete meter point
// @Description Permanently removes a meter point. This action is irreversible.
// @Tags        Zählpunkte
// @Param       eegID         path  string  true  "EEG UUID"
// @Param       meterPointID  path  string  true  "Meter point UUID"
// @Success     204  "No content"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/meter-points/{meterPointID} [delete]
// DeleteMeterPoint handles DELETE /eegs/{eegID}/meter-points/{meterPointID}
func (h *MeterPointHandler) DeleteMeterPoint(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	meterPointID, err := uuid.Parse(chi.URLParam(r, "meterPointID"))
	if err != nil {
		jsonError(w, "invalid meter point ID", http.StatusBadRequest)
		return
	}

	existing, err := h.meterPointRepo.GetByID(r.Context(), meterPointID)
	if err != nil || existing.EegID != eeg.ID {
		jsonError(w, "meter point not found", http.StatusNotFound)
		return
	}

	if active, err := h.edaProcRepo.HasActiveProcess(r.Context(), meterPointID); err == nil && active {
		jsonError(w, "Zählpunkt hat einen laufenden EDA-Vorgang (z.B. Anmeldung ausstehend) — bitte Bestätigung abwarten, sonst geht sie beim Löschen verloren", http.StatusConflict)
		return
	}

	if err := h.meterPointRepo.Delete(r.Context(), meterPointID); err != nil {
		jsonError(w, "failed to delete meter point", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
