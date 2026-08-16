package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// ExtraMeterHandler manages Zusatzzähler — manually-read submeters that are NOT
// Netzbetreiber smart meters (e.g. Wärmepumpe, Werkstatt) and never receive EDA data.
type ExtraMeterHandler struct {
	extraMeterRepo *repository.ExtraMeterRepository
	memberRepo     *repository.MemberRepository
	eegRepo        *repository.EEGRepository
}

func NewExtraMeterHandler(extraMeterRepo *repository.ExtraMeterRepository, memberRepo *repository.MemberRepository, eegRepo *repository.EEGRepository) *ExtraMeterHandler {
	return &ExtraMeterHandler{extraMeterRepo: extraMeterRepo, memberRepo: memberRepo, eegRepo: eegRepo}
}

// verifyMember confirms memberID belongs to eegID, returning the member or writing a 404.
func (h *ExtraMeterHandler) verifyMember(w http.ResponseWriter, r *http.Request, eegID, memberID uuid.UUID) (*domain.Member, bool) {
	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil || member.EegID != eegID {
		jsonError(w, "member not found", http.StatusNotFound)
		return nil, false
	}
	return member, true
}

// verifyExtraMeter confirms extraMeterID belongs to (eegID, memberID), returning it or writing a 404.
func (h *ExtraMeterHandler) verifyExtraMeter(w http.ResponseWriter, r *http.Request, eegID, memberID, extraMeterID uuid.UUID) (*domain.ExtraMeter, bool) {
	m, err := h.extraMeterRepo.GetByID(r.Context(), extraMeterID)
	if err != nil || m.EegID != eegID || m.MemberID != memberID {
		jsonError(w, "extra meter not found", http.StatusNotFound)
		return nil, false
	}
	return m, true
}

// ListExtraMeters handles GET /eegs/{eegID}/members/{memberID}/extra-meters
func (h *ExtraMeterHandler) ListExtraMeters(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyMember(w, r, eeg.ID, memberID); !ok {
		return
	}
	list, err := h.extraMeterRepo.ListByMember(r.Context(), memberID)
	if err != nil {
		jsonError(w, "failed to list extra meters", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []domain.ExtraMeter{}
	}
	jsonOK(w, list)
}

// CreateExtraMeter handles POST /eegs/{eegID}/members/{memberID}/extra-meters
func (h *ExtraMeterHandler) CreateExtraMeter(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	if !eeg.ExtraMetersEnabled {
		jsonError(w, "Zusatzzähler sind für diese EEG nicht aktiviert", http.StatusForbidden)
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyMember(w, r, eeg.ID, memberID); !ok {
		return
	}
	var req struct {
		Label string `json:"label"`
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Label == "" {
		jsonError(w, "label is required", http.StatusBadRequest)
		return
	}
	m := &domain.ExtraMeter{
		EegID:    eeg.ID,
		MemberID: memberID,
		Label:    req.Label,
		Status:   "ACTIVE",
		Notes:    req.Notes,
	}
	if err := h.extraMeterRepo.Create(r.Context(), m); err != nil {
		jsonError(w, "failed to create extra meter", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, m)
}

// UpdateExtraMeter handles PUT /eegs/{eegID}/members/{memberID}/extra-meters/{id}
func (h *ExtraMeterHandler) UpdateExtraMeter(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	extraMeterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid extra meter ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyExtraMeter(w, r, eeg.ID, memberID, extraMeterID); !ok {
		return
	}
	var req struct {
		Label  string `json:"label"`
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Status != "ACTIVE" && req.Status != "INACTIVE" {
		jsonError(w, "status must be ACTIVE or INACTIVE", http.StatusBadRequest)
		return
	}
	m := &domain.ExtraMeter{ID: extraMeterID, Label: req.Label, Status: req.Status, Notes: req.Notes}
	if err := h.extraMeterRepo.Update(r.Context(), m); err != nil {
		jsonError(w, "failed to update extra meter: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, m)
}

// DeleteExtraMeter handles DELETE /eegs/{eegID}/members/{memberID}/extra-meters/{id}
func (h *ExtraMeterHandler) DeleteExtraMeter(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	extraMeterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid extra meter ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyExtraMeter(w, r, eeg.ID, memberID, extraMeterID); !ok {
		return
	}
	if err := h.extraMeterRepo.Delete(r.Context(), extraMeterID); err != nil {
		jsonError(w, "failed to delete extra meter", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "deleted"})
}

// ListReadings handles GET /eegs/{eegID}/members/{memberID}/extra-meters/{id}/readings
func (h *ExtraMeterHandler) ListReadings(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	extraMeterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid extra meter ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyExtraMeter(w, r, eeg.ID, memberID, extraMeterID); !ok {
		return
	}
	list, err := h.extraMeterRepo.ListReadings(r.Context(), extraMeterID)
	if err != nil {
		jsonError(w, "failed to list readings", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []domain.ExtraMeterReading{}
	}
	jsonOK(w, list)
}

// CreateReading handles POST /eegs/{eegID}/members/{memberID}/extra-meters/{id}/readings
func (h *ExtraMeterHandler) CreateReading(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	extraMeterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid extra meter ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyExtraMeter(w, r, eeg.ID, memberID, extraMeterID); !ok {
		return
	}
	var req struct {
		ReadingDate  string  `json:"reading_date"`
		CounterValue float64 `json:"counter_value"`
		Notes        string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	date, err := time.Parse("2006-01-02", req.ReadingDate)
	if err != nil {
		jsonError(w, "invalid reading_date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	rd := &domain.ExtraMeterReading{
		ExtraMeterID: extraMeterID,
		ReadingDate:  date,
		CounterValue: req.CounterValue,
		Notes:        req.Notes,
	}
	claims := auth.ClaimsFromContext(r.Context())
	if claims != nil {
		if userID, err := uuid.Parse(claims.Subject); err == nil {
			rd.CreatedBy = &userID
		}
	}
	if err := h.extraMeterRepo.CreateReading(r.Context(), rd); err != nil {
		jsonError(w, "failed to create reading: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, rd)
}

// DeleteReading handles DELETE /eegs/{eegID}/members/{memberID}/extra-meters/{id}/readings/{readingID}
func (h *ExtraMeterHandler) DeleteReading(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	extraMeterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid extra meter ID", http.StatusBadRequest)
		return
	}
	if _, ok := h.verifyExtraMeter(w, r, eeg.ID, memberID, extraMeterID); !ok {
		return
	}
	readingID, err := uuid.Parse(chi.URLParam(r, "readingID"))
	if err != nil {
		jsonError(w, "invalid reading ID", http.StatusBadRequest)
		return
	}
	if err := h.extraMeterRepo.DeleteReading(r.Context(), readingID); err != nil {
		jsonError(w, "failed to delete reading", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "deleted"})
}
