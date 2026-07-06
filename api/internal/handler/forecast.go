package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type ForecastHandler struct {
	eegRepo     *repository.EEGRepository
	forecastURL string
}

func NewForecastHandler(eegRepo *repository.EEGRepository, forecastURL string) *ForecastHandler {
	return &ForecastHandler{eegRepo: eegRepo, forecastURL: forecastURL}
}

func (h *ForecastHandler) GetForecast(w http.ResponseWriter, r *http.Request) {
	_, _, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	eegID := chi.URLParam(r, "eegID")

	resp, err := http.Get(fmt.Sprintf("%s/forecast/%s", h.forecastURL, eegID))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"forecast service unavailable"}`))
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *ForecastHandler) TriggerTraining(w http.ResponseWriter, r *http.Request) {
	_, _, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	eegID := chi.URLParam(r, "eegID")

	resp, err := http.Post(fmt.Sprintf("%s/train/%s", h.forecastURL, eegID), "application/json", nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"forecast service unavailable"}`))
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
