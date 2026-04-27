package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type TariffHandler struct {
	tariffRepo *repository.TariffRepository
}

func NewTariffHandler(tariffRepo *repository.TariffRepository) *TariffHandler {
	return &TariffHandler{tariffRepo: tariffRepo}
}

// ListSchedules godoc
//
//	@Summary		Tarifpläne auflisten
//	@Description	Gibt alle Tarifpläne (TariffSchedules) einer EEG zurück.
//	@Tags			Tarifpläne
//	@Produce		json
//	@Param			eegID	path		string						true	"EEG ID (UUID)"
//	@Success		200		{array}		domain.TariffSchedule		"Liste der Tarifpläne"
//	@Failure		400		{object}	object						"Ungültige EEG ID"
//	@Failure		500		{object}	object						"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs [get]
func (h *TariffHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	schedules, err := h.tariffRepo.ListByEeg(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to list tariff schedules", http.StatusInternalServerError)
		return
	}
	if schedules == nil {
		schedules = []domain.TariffSchedule{}
	}
	jsonOK(w, schedules)
}

// CreateSchedule godoc
//
//	@Summary		Tarifplan anlegen
//	@Description	Legt einen neuen Tarifplan für eine EEG an. Granularität: annual, monthly, daily, quarter_hour (Standard: monthly).
//	@Tags			Tarifpläne
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			body	body		object	true	"Tarifplan-Daten"	SchemaExample({"name":"Tarif 2025","granularity":"monthly"})
//	@Success		200		{object}	domain.TariffSchedule	"Angelegter Tarifplan"
//	@Failure		400		{object}	object					"Ungültige EEG ID oder fehlender Name"
//	@Failure		500		{object}	object					"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs [post]
func (h *TariffHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Granularity string `json:"granularity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	valid := map[string]bool{"annual": true, "monthly": true, "daily": true, "quarter_hour": true}
	if !valid[req.Granularity] {
		req.Granularity = "monthly"
	}
	s := &domain.TariffSchedule{EegID: eegID, Name: req.Name, Granularity: req.Granularity}
	if err := h.tariffRepo.Create(r.Context(), s); err != nil {
		jsonError(w, "failed to create schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, s)
}

// GetSchedule godoc
//
//	@Summary		Tarifplan abrufen
//	@Description	Gibt einen einzelnen Tarifplan inklusive aller Tarifeinträge zurück.
//	@Tags			Tarifpläne
//	@Produce		json
//	@Param			eegID		path		string					true	"EEG ID (UUID)"
//	@Param			scheduleID	path		string					true	"Tarifplan-ID (UUID)"
//	@Success		200			{object}	domain.TariffSchedule	"Tarifplan mit Einträgen"
//	@Failure		400			{object}	object					"Ungültige ID"
//	@Failure		404			{object}	object					"Tarifplan nicht gefunden"
//	@Failure		500			{object}	object					"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs/{scheduleID} [get]
func (h *TariffHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		jsonError(w, "invalid schedule ID", http.StatusBadRequest)
		return
	}
	s, err := h.tariffRepo.GetWithEntries(r.Context(), scheduleID)
	if err != nil {
		jsonError(w, "failed to get schedule", http.StatusInternalServerError)
		return
	}
	if s == nil {
		jsonError(w, "schedule not found", http.StatusNotFound)
		return
	}
	jsonOK(w, s)
}

// UpdateSchedule godoc
//
//	@Summary		Tarifplan aktualisieren
//	@Description	Aktualisiert Name und Granularität eines bestehenden Tarifplans.
//	@Tags			Tarifpläne
//	@Accept			json
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			scheduleID	path		string	true	"Tarifplan-ID (UUID)"
//	@Param			body		body		object	true	"Aktualisierte Tarifplan-Daten"	SchemaExample({"name":"Tarif 2026","granularity":"monthly"})
//	@Success		200			{object}	domain.TariffSchedule	"Aktualisierter Tarifplan"
//	@Failure		400			{object}	object					"Ungültige ID oder Request"
//	@Failure		500			{object}	object					"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs/{scheduleID} [put]
func (h *TariffHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		jsonError(w, "invalid schedule ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Granularity string `json:"granularity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	s := &domain.TariffSchedule{ID: scheduleID, Name: req.Name, Granularity: req.Granularity}
	if err := h.tariffRepo.Update(r.Context(), s); err != nil {
		jsonError(w, "failed to update schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, s)
}

// DeleteSchedule godoc
//
//	@Summary		Tarifplan löschen
//	@Description	Löscht einen Tarifplan und alle zugehörigen Tarifeinträge endgültig.
//	@Tags			Tarifpläne
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			scheduleID	path		string	true	"Tarifplan-ID (UUID)"
//	@Success		200			{object}	object	"Bestätigung"
//	@Failure		400			{object}	object	"Ungültige ID"
//	@Failure		500			{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs/{scheduleID} [delete]
func (h *TariffHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		jsonError(w, "invalid schedule ID", http.StatusBadRequest)
		return
	}
	if err := h.tariffRepo.Delete(r.Context(), scheduleID); err != nil {
		jsonError(w, "failed to delete schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "deleted"})
}

// ActivateSchedule godoc
//
//	@Summary		Tarifplan aktivieren
//	@Description	Aktiviert einen Tarifplan für die EEG. Pro EEG kann nur ein Tarifplan gleichzeitig aktiv sein. Gibt 409 zurück, wenn bereits ein anderer aktiv ist.
//	@Tags			Tarifpläne
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			scheduleID	path		string	true	"Tarifplan-ID (UUID)"
//	@Success		200			{object}	object	"is_active: true"
//	@Failure		400			{object}	object	"Ungültige ID"
//	@Failure		409			{object}	object	"Ein anderer Tarifplan ist bereits aktiv"
//	@Failure		500			{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs/{scheduleID}/activate [post]
func (h *TariffHandler) ActivateSchedule(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		jsonError(w, "invalid schedule ID", http.StatusBadRequest)
		return
	}
	if err := h.tariffRepo.Activate(r.Context(), scheduleID, eegID); err != nil {
		jsonError(w, "failed to activate schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"is_active": true})
}

// DeactivateSchedule godoc
//
//	@Summary		Tarifplan deaktivieren
//	@Description	Deaktiviert den aktiven Tarifplan, ohne ihn zu löschen.
//	@Tags			Tarifpläne
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			scheduleID	path		string	true	"Tarifplan-ID (UUID)"
//	@Success		200			{object}	object	"is_active: false"
//	@Failure		400			{object}	object	"Ungültige ID"
//	@Failure		500			{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs/{scheduleID}/activate [delete]
func (h *TariffHandler) DeactivateSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		jsonError(w, "invalid schedule ID", http.StatusBadRequest)
		return
	}
	if err := h.tariffRepo.Deactivate(r.Context(), scheduleID); err != nil {
		jsonError(w, "failed to deactivate schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"is_active": false})
}

// SetEntries godoc
//
//	@Summary		Tarifeinträge setzen
//	@Description	Ersetzt alle Tarifeinträge eines Tarifplans vollständig durch die übergebene Liste. Jeder Eintrag enthält einen Gültigkeitszeitraum sowie Energie- und Einspeisepreis in ct/kWh.
//	@Tags			Tarifpläne
//	@Accept			json
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			scheduleID	path		string	true	"Tarifplan-ID (UUID)"
//	@Param			body		body		array	true	"Liste der Tarifeinträge"	SchemaExample([{"valid_from":"2025-01-01T00:00:00Z","valid_until":"2025-12-31T23:59:59Z","energy_price":8.5,"producer_price":5.0}])
//	@Success		200			{object}	object	"Anzahl gespeicherter Einträge"
//	@Failure		400			{object}	object	"Ungültige ID, Datumsformat oder Request"
//	@Failure		500			{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/tariffs/{scheduleID}/entries [put]
func (h *TariffHandler) SetEntries(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		jsonError(w, "invalid schedule ID", http.StatusBadRequest)
		return
	}
	var req []struct {
		ValidFrom     string  `json:"valid_from"`
		ValidUntil    string  `json:"valid_until"`
		EnergyPrice   float64 `json:"energy_price"`
		ProducerPrice float64 `json:"producer_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	entries := make([]domain.TariffEntry, 0, len(req))
	for _, e := range req {
		from, err := time.Parse(time.RFC3339, e.ValidFrom)
		if err != nil {
			jsonError(w, "invalid valid_from: "+e.ValidFrom, http.StatusBadRequest)
			return
		}
		until, err := time.Parse(time.RFC3339, e.ValidUntil)
		if err != nil {
			jsonError(w, "invalid valid_until: "+e.ValidUntil, http.StatusBadRequest)
			return
		}
		entries = append(entries, domain.TariffEntry{
			ScheduleID:    scheduleID,
			ValidFrom:     from,
			ValidUntil:    until,
			EnergyPrice:   e.EnergyPrice,
			ProducerPrice: e.ProducerPrice,
		})
	}
	if err := h.tariffRepo.ReplaceEntries(r.Context(), scheduleID, entries); err != nil {
		jsonError(w, "failed to save entries: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]int{"saved": len(entries)})
}
