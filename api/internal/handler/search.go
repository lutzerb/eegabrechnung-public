package handler

import (
	"net/http"
	"strings"

	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type SearchHandler struct {
	memberRepo     *repository.MemberRepository
	meterPointRepo *repository.MeterPointRepository
	invoiceRepo    *repository.InvoiceRepository
	eegRepo        *repository.EEGRepository
}

func NewSearchHandler(
	memberRepo *repository.MemberRepository,
	meterPointRepo *repository.MeterPointRepository,
	invoiceRepo *repository.InvoiceRepository,
	eegRepo *repository.EEGRepository,
) *SearchHandler {
	return &SearchHandler{
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		invoiceRepo:    invoiceRepo,
		eegRepo:        eegRepo,
	}
}

// Search handles GET /api/v1/eegs/{eegID}/search?q=
//
//	@Summary		Search members, meter points and invoices
//	@Description	Full-text search across members (name, email, member number), meter points (Zählpunktnummer), and invoices (invoice number). The query string must be at least 2 characters long. Returns empty arrays for each category when no matches are found.
//	@Tags			System
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			q		query		string	true	"Search query (minimum 2 characters)"
//	@Success		200		{object}	map[string]interface{}	"members, meter_points, invoices arrays"
//	@Failure		400		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/search [get]
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	const searchLimit = 25

	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		jsonOK(w, map[string]any{
			"members":      []any{},
			"meter_points": []any{},
			"invoices":     []any{},
		})
		return
	}

	matchedMembers := []map[string]any{}
	if members, err := h.memberRepo.SearchPreviewByEeg(r.Context(), eeg.ID, q, searchLimit); err == nil {
		for _, m := range members {
			matchedMembers = append(matchedMembers, map[string]any{
				"id":           m.ID,
				"name":         strings.TrimSpace(m.Name1 + " " + m.Name2),
				"email":        m.Email,
				"mitglieds_nr": m.MitgliedsNr,
			})
		}
	}

	matchedMeterPoints := []map[string]any{}
	if meterPoints, err := h.meterPointRepo.SearchByEeg(r.Context(), eeg.ID, q, searchLimit); err == nil {
		for _, mp := range meterPoints {
			matchedMeterPoints = append(matchedMeterPoints, map[string]any{
				"id":          mp.ID,
				"zaehlpunkt":  mp.Zaehlpunkt,
				"direction":   mp.Energierichtung,
				"member_id":   mp.MemberID,
				"member_name": mp.MemberName,
			})
		}
	}

	matchedInvoices := []map[string]any{}
	if invoices, err := h.invoiceRepo.SearchByEeg(r.Context(), eeg.ID, q, searchLimit); err == nil {
		for _, inv := range invoices {
			matchedInvoices = append(matchedInvoices, map[string]any{
				"id":             inv.ID,
				"invoice_number": *inv.InvoiceNumber,
				"total_amount":   inv.TotalAmount,
				"member_id":      inv.MemberID,
				"status":         inv.Status,
				"period_start":   inv.PeriodStart,
				"period_end":     inv.PeriodEnd,
			})
		}
	}

	if matchedMembers == nil {
		matchedMembers = []map[string]any{}
	}
	if matchedMeterPoints == nil {
		matchedMeterPoints = []map[string]any{}
	}
	if matchedInvoices == nil {
		matchedInvoices = []map[string]any{}
	}

	jsonOK(w, map[string]any{
		"members":      matchedMembers,
		"meter_points": matchedMeterPoints,
		"invoices":     matchedInvoices,
	})
}
