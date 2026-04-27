package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/oemag"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type OemagHandler struct {
	eegRepo *repository.EEGRepository
}

func NewOemagHandler(eegRepo *repository.EEGRepository) *OemagHandler {
	return &OemagHandler{eegRepo: eegRepo}
}

// GetMarktpreis handles GET /api/v1/oemag/marktpreis
// Returns static historical prices plus a fresh scrape of the current year.
// Also seeds the in-memory cache so subsequent Refresh calls can detect new months.
func (h *OemagHandler) GetMarktpreis(w http.ResponseWriter, r *http.Request) {
	result, err := oemag.Refresh() // seeds cache as a side-effect
	if err != nil {
		jsonError(w, "OeMAG-Preise konnten nicht abgerufen werden: "+err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w, &oemag.MarktpreisResult{
		Years:     result.All,
		ScrapedAt: result.ScrapedAt,
	})
}

// RefreshMarktpreis handles POST /api/v1/oemag/refresh
// Re-scrapes the OeMAG page and returns newly published months compared to the
// last known state.
func (h *OemagHandler) RefreshMarktpreis(w http.ResponseWriter, r *http.Request) {
	result, err := oemag.Refresh()
	if err != nil {
		jsonError(w, "OeMAG-Aktualisierung fehlgeschlagen: "+err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w, result)
}

type oemagSyncRequest struct {
	// PriceType selects the price column: "pv" (Photovoltaik & andere) or "wind".
	PriceType string `json:"price_type"`
	// Target selects which EEG price field to update: "producer_price" or "energy_price".
	Target string `json:"target"`
}

// SyncEEGPrice handles POST /api/v1/eegs/{eegID}/oemag/sync
// Fetches the latest available OeMAG price and writes it into the specified
// EEG price field.
func (h *OemagHandler) SyncEEGPrice(w http.ResponseWriter, r *http.Request) {
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

	var req oemagSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.PriceType != "pv" && req.PriceType != "wind" {
		jsonError(w, "price_type must be 'pv' or 'wind'", http.StatusBadRequest)
		return
	}
	if req.Target != "producer_price" && req.Target != "energy_price" {
		jsonError(w, "target must be 'producer_price' or 'energy_price'", http.StatusBadRequest)
		return
	}

	monthPrice, err := oemag.FetchCurrentMonth()
	if err != nil {
		jsonError(w, "OeMAG-Preis konnte nicht abgerufen werden: "+err.Error(), http.StatusBadGateway)
		return
	}

	price := monthPrice.PVPrice
	if req.PriceType == "wind" {
		price = monthPrice.WindPrice
	}

	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG nicht gefunden", http.StatusNotFound)
		return
	}

	if req.Target == "producer_price" {
		eeg.ProducerPrice = price
	} else {
		eeg.EnergyPrice = price
	}

	if err := h.eegRepo.Update(r.Context(), eeg); err != nil {
		jsonError(w, "EEG konnte nicht aktualisiert werden", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"month":      monthPrice.Month,
		"price":      price,
		"target":     req.Target,
		"price_type": req.PriceType,
		"eeg":        eeg,
	})
}
