package handler

import (
	"net/http"
	"strconv"

	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// EmailLogHandler serves the email activity log.
type EmailLogHandler struct {
	emailLogRepo *repository.EmailLogRepository
	eegRepo      *repository.EEGRepository
}

func NewEmailLogHandler(emailLogRepo *repository.EmailLogRepository, eegRepo *repository.EEGRepository) *EmailLogHandler {
	return &EmailLogHandler{emailLogRepo: emailLogRepo, eegRepo: eegRepo}
}

// List handles GET /eegs/{eegID}/email-log
func (h *EmailLogHandler) List(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	limit := 100
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, total, err := h.emailLogRepo.ListByEEG(r.Context(), eeg.ID, limit, offset)
	if err != nil {
		jsonError(w, "failed to load email log", http.StatusInternalServerError)
		return
	}

	type response struct {
		Total   int         `json:"total"`
		Entries interface{} `json:"entries"`
	}
	if entries == nil {
		entries = []domain.EmailLogEntry{}
	}
	jsonOK(w, response{Total: total, Entries: entries})
}
