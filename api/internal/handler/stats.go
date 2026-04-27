package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type StatsHandler struct {
	eegRepo        *repository.EEGRepository
	edaMessageRepo *repository.EDAMessageRepository
}

type edaMessagesResponse struct {
	Messages   []domain.EDAMessage `json:"messages"`
	TotalCount int                 `json:"total_count"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
}

func NewStatsHandler(eegRepo *repository.EEGRepository, edaMessageRepo *repository.EDAMessageRepository) *StatsHandler {
	return &StatsHandler{eegRepo: eegRepo, edaMessageRepo: edaMessageRepo}
}

// GetStats handles GET /api/v1/eegs/{eegID}/stats
//
//	@Summary		Get EEG statistics
//	@Description	Returns aggregate statistics for the EEG such as member counts, meter point counts, and energy totals.
//	@Tags			System
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/stats [get]
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	stats, err := h.eegRepo.GetStats(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to get stats", http.StatusInternalServerError)
		return
	}
	jsonOK(w, stats)
}

// GetEDAMessages handles GET /api/v1/eegs/{eegID}/eda/messages
//
//	@Summary		List EDA messages for an EEG
//	@Description	Returns EDA protocol messages (inbound and outbound) for the given EEG, ordered by received/sent time descending. The result is limited to 100 entries by default; use the limit and offset query parameters for pagination (max 500 per page).
//	@Tags			System
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			limit	query		int		false	"Maximum number of messages to return (default 100, max 500)"
//	@Param			offset	query		int		false	"Number of messages to skip for pagination (default 0)"
//	@Success		200		{object}	edaMessagesResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/messages [get]
func (h *StatsHandler) GetEDAMessages(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	const maxLimit = 500
	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	totalCount, err := h.edaMessageRepo.CountByEEG(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to count EDA messages", http.StatusInternalServerError)
		return
	}

	msgs, err := h.edaMessageRepo.ListByEEG(r.Context(), eegID, limit, offset)
	if err != nil {
		jsonError(w, "failed to list EDA messages", http.StatusInternalServerError)
		return
	}
	if msgs == nil {
		msgs = []domain.EDAMessage{}
	}
	jsonOK(w, edaMessagesResponse{
		Messages:   msgs,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	})
}

// GetEDAMessageXML handles GET /api/v1/eegs/{eegID}/eda/messages/{msgID}/xml
// Returns the raw XML payload for a single EDA message as an attachment download.
//
//	@Summary		Download raw XML for an EDA message
//	@Description	Returns the raw MaKo XML payload for the specified EDA message as a downloadable attachment (application/xml). The message must belong to the given EEG.
//	@Tags			System
//	@Produce		application/xml
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			msgID	path		string	true	"EDA message UUID"
//	@Success		200		{file}		binary
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/messages/{msgID}/xml [get]
func (h *StatsHandler) GetEDAMessageXML(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		jsonError(w, "invalid message ID", http.StatusBadRequest)
		return
	}
	payload, err := h.edaMessageRepo.GetXMLPayload(r.Context(), msgID, eegID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			jsonError(w, "message not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get XML payload", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="eda-message-%s.xml"`, msgID))
	w.Write([]byte(payload))
}
