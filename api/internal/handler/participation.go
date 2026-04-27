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

// ParticipationHandler manages EEG meter point participation records (Mehrfachteilnahme).
type ParticipationHandler struct {
	repo    *repository.ParticipationRepository
	eegRepo *repository.EEGRepository
}

func NewParticipationHandler(repo *repository.ParticipationRepository, eegRepo *repository.EEGRepository) *ParticipationHandler {
	return &ParticipationHandler{repo: repo, eegRepo: eegRepo}
}

// ListByEEG handles GET /api/v1/eegs/{eegID}/participations
//
//	@Summary		List participations for an EEG
//	@Description	Returns all Mehrfachteilnahme records (meter point participations) for the given EEG.
//	@Tags			Mehrfachteilnahme
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Success		200		{array}		domain.EEGMeterParticipation
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/participations [get]
func (h *ParticipationHandler) ListByEEG(w http.ResponseWriter, r *http.Request) {
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
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	list, err := h.repo.ListByEEG(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to list participations", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []domain.EEGMeterParticipation{}
	}
	jsonOK(w, list)
}

// ListByMeterPoint handles GET /api/v1/eegs/{eegID}/meter-points/{mpID}/participations
func (h *ParticipationHandler) ListByMeterPoint(w http.ResponseWriter, r *http.Request) {
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
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	mpID, err := uuid.Parse(chi.URLParam(r, "mpID"))
	if err != nil {
		jsonError(w, "invalid meter point ID", http.StatusBadRequest)
		return
	}
	list, err := h.repo.ListByMeterPoint(r.Context(), mpID)
	if err != nil {
		jsonError(w, "failed to list participations", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []domain.EEGMeterParticipation{}
	}
	jsonOK(w, list)
}

// participationRequest is the body for creating/updating a participation.
type participationRequest struct {
	MeterPointID        string  `json:"meter_point_id"`
	ParticipationFactor float64 `json:"participation_factor"` // 0.0001–100
	ShareType           string  `json:"share_type"`
	ValidFrom           string  `json:"valid_from"`  // YYYY-MM-DD
	ValidUntil          string  `json:"valid_until"` // YYYY-MM-DD or empty
	Notes               string  `json:"notes"`
}

// Create handles POST /api/v1/eegs/{eegID}/participations
//
//	@Summary		Create a meter point participation
//	@Description	Creates a new Mehrfachteilnahme record linking a meter point to the EEG with a given participation factor, share type (GC/RC_R/RC_L/CC), and validity period.
//	@Tags			Mehrfachteilnahme
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG UUID"
//	@Param			body	body		participationRequest	true	"Participation details"
//	@Success		201		{object}	domain.EEGMeterParticipation
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/participations [post]
func (h *ParticipationHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	var req participationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	mpID, err := uuid.Parse(req.MeterPointID)
	if err != nil {
		jsonError(w, "meter_point_id is required", http.StatusBadRequest)
		return
	}
	if req.ParticipationFactor <= 0 || req.ParticipationFactor > 100 {
		jsonError(w, "participation_factor must be between 0 and 100", http.StatusBadRequest)
		return
	}
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		jsonError(w, "valid_from must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	p := &domain.EEGMeterParticipation{
		EegID:               eegID,
		MeterPointID:        mpID,
		ParticipationFactor: req.ParticipationFactor,
		ShareType:           req.ShareType,
		ValidFrom:           validFrom,
		Notes:               req.Notes,
	}
	if req.ShareType == "" {
		p.ShareType = "GC"
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err != nil {
			jsonError(w, "valid_until must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		p.ValidUntil = &t
	}

	if err := h.repo.Create(r.Context(), p); err != nil {
		jsonError(w, "failed to create participation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, p)
}

// Update handles PUT /api/v1/eegs/{eegID}/participations/{id}
//
//	@Summary		Update a meter point participation
//	@Description	Updates an existing Mehrfachteilnahme record. All writable fields are replaced.
//	@Tags			Mehrfachteilnahme
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG UUID"
//	@Param			id		path		string				true	"Participation UUID"
//	@Param			body	body		participationRequest	true	"Updated participation details"
//	@Success		200		{object}	domain.EEGMeterParticipation
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/participations/{id} [put]
func (h *ParticipationHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid participation ID", http.StatusBadRequest)
		return
	}

	var req participationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ParticipationFactor <= 0 || req.ParticipationFactor > 100 {
		jsonError(w, "participation_factor must be between 0 and 100", http.StatusBadRequest)
		return
	}
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		jsonError(w, "valid_from must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	p := &domain.EEGMeterParticipation{
		ID:                  id,
		EegID:               eegID,
		ParticipationFactor: req.ParticipationFactor,
		ShareType:           req.ShareType,
		ValidFrom:           validFrom,
		Notes:               req.Notes,
	}
	if req.ShareType == "" {
		p.ShareType = "GC"
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err != nil {
			jsonError(w, "valid_until must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		p.ValidUntil = &t
	}

	if err := h.repo.Update(r.Context(), p); err != nil {
		jsonError(w, "failed to update participation", http.StatusInternalServerError)
		return
	}
	jsonOK(w, p)
}

// Delete handles DELETE /api/v1/eegs/{eegID}/participations/{id}
//
//	@Summary		Delete a meter point participation
//	@Description	Permanently deletes a Mehrfachteilnahme record.
//	@Tags			Mehrfachteilnahme
//	@Param			eegID	path	string	true	"EEG UUID"
//	@Param			id		path	string	true	"Participation UUID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/participations/{id} [delete]
func (h *ParticipationHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid participation ID", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		jsonError(w, "failed to delete participation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
