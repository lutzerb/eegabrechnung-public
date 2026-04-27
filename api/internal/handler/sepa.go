package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	"github.com/lutzerb/eegabrechnung/internal/sepa"
)

type SEPAHandler struct {
	eegRepo     *repository.EEGRepository
	memberRepo  *repository.MemberRepository
	invoiceRepo *repository.InvoiceRepository
}

func NewSEPAHandler(eegRepo *repository.EEGRepository, memberRepo *repository.MemberRepository, invoiceRepo *repository.InvoiceRepository) *SEPAHandler {
	return &SEPAHandler{eegRepo: eegRepo, memberRepo: memberRepo, invoiceRepo: invoiceRepo}
}

// DownloadPain001 godoc
//
//	@Summary		Download SEPA pain.001 credit-transfer file
//	@Description	Generates and returns a pain.001.001.03 SEPA credit-transfer XML for the invoices of the given EEG. Optionally filtered by billing_run_id or legacy period_start/period_end query params. Only invoices with status draft, pending, or sent are included.
//	@Tags			SEPA
//	@Produce		application/xml
//	@Param			eegID			path		string	true	"EEG ID (UUID)"
//	@Param			billing_run_id	query		string	false	"Filter by billing run ID (UUID)"
//	@Param			period_start	query		string	false	"Legacy filter: period start date (YYYY-MM-DD)"
//	@Param			period_end		query		string	false	"Legacy filter: period end date (YYYY-MM-DD)"
//	@Success		200				{file}		application/xml	"pain.001 XML file attachment"
//	@Failure		400				{object}	map[string]string
//	@Failure		422				{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/sepa/pain001 [get]
func (h *SEPAHandler) DownloadPain001(w http.ResponseWriter, r *http.Request) {
	eeg, invoices, membersByID, err := h.loadData(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	entries := buildP001Entries(invoices, membersByID)
	xmlBytes, err := sepa.GeneratePain001(eeg, entries)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	filename := fmt.Sprintf("pain001_%s_%s.xml", eeg.GemeinschaftID, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Write(xmlBytes)
}

// DownloadPain008 godoc
//
//	@Summary		Download SEPA pain.008 direct-debit file
//	@Description	Generates and returns a pain.008.001.02 SEPA direct-debit XML for the invoices of the given EEG. Optionally filtered by billing_run_id or legacy period_start/period_end query params. Only invoices with status draft, pending, or sent are included.
//	@Tags			SEPA
//	@Produce		application/xml
//	@Param			eegID			path		string	true	"EEG ID (UUID)"
//	@Param			billing_run_id	query		string	false	"Filter by billing run ID (UUID)"
//	@Param			period_start	query		string	false	"Legacy filter: period start date (YYYY-MM-DD)"
//	@Param			period_end		query		string	false	"Legacy filter: period end date (YYYY-MM-DD)"
//	@Success		200				{file}		application/xml	"pain.008 XML file attachment"
//	@Failure		400				{object}	map[string]string
//	@Failure		422				{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/sepa/pain008 [get]
func (h *SEPAHandler) DownloadPain008(w http.ResponseWriter, r *http.Request) {
	eeg, invoices, membersByID, err := h.loadData(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	entries := buildP008Entries(invoices, membersByID)
	xmlBytes, err := sepa.GeneratePain008(eeg, entries)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	filename := fmt.Sprintf("pain008_%s_%s.xml", eeg.GemeinschaftID, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Write(xmlBytes)
}

func (h *SEPAHandler) loadData(r *http.Request) (*domain.EEG, []domain.Invoice, map[uuid.UUID]*domain.Member, error) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return nil, nil, nil, fmt.Errorf("unauthorized")
	}

	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid EEG ID")
	}

	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("EEG not found")
	}

	invoices, err := h.invoiceRepo.ListByEeg(r.Context(), eegID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invoices laden fehlgeschlagen")
	}

	// Filter by billing_run_id if provided; otherwise fall back to period params
	q := r.URL.Query()
	if runIDStr := q.Get("billing_run_id"); runIDStr != "" {
		runID, err := uuid.Parse(runIDStr)
		if err == nil {
			var filtered []domain.Invoice
			for _, inv := range invoices {
				if inv.BillingRunID != nil && *inv.BillingRunID == runID {
					filtered = append(filtered, inv)
				}
			}
			invoices = filtered
		}
	} else {
		// Legacy period filter
		if ps := q.Get("period_start"); ps != "" {
			t, err := time.Parse("2006-01-02", ps)
			if err == nil {
				var filtered []domain.Invoice
				for _, inv := range invoices {
					if !inv.PeriodStart.Before(t) {
						filtered = append(filtered, inv)
					}
				}
				invoices = filtered
			}
		}
		if pe := q.Get("period_end"); pe != "" {
			t, err := time.Parse("2006-01-02", pe)
			if err == nil {
				var filtered []domain.Invoice
				for _, inv := range invoices {
					if !inv.PeriodEnd.After(t) {
						filtered = append(filtered, inv)
					}
				}
				invoices = filtered
			}
		}
	}

	// Only include draft/pending/sent invoices
	var active []domain.Invoice
	for _, inv := range invoices {
		s := inv.Status
		if s == "pending" || s == "sent" || s == "draft" {
			active = append(active, inv)
		}
	}
	invoices = active

	members, err := h.memberRepo.ListByEeg(r.Context(), eegID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("mitglieder laden fehlgeschlagen")
	}
	membersByID := make(map[uuid.UUID]*domain.Member, len(members))
	for i := range members {
		membersByID[members[i].ID] = &members[i]
	}

	return eeg, invoices, membersByID, nil
}

func buildP001Entries(invoices []domain.Invoice, membersByID map[uuid.UUID]*domain.Member) []sepa.Pain001Entry {
	var entries []sepa.Pain001Entry
	for i := range invoices {
		inv := &invoices[i]
		m, ok := membersByID[inv.MemberID]
		if !ok {
			continue
		}
		entries = append(entries, sepa.Pain001Entry{Invoice: inv, Member: m})
	}
	return entries
}

func buildP008Entries(invoices []domain.Invoice, membersByID map[uuid.UUID]*domain.Member) []sepa.Pain008Entry {
	var entries []sepa.Pain008Entry
	for i := range invoices {
		inv := &invoices[i]
		m, ok := membersByID[inv.MemberID]
		if !ok {
			continue
		}
		entries = append(entries, sepa.Pain008Entry{Invoice: inv, Member: m})
	}
	return entries
}

// ImportCAMT054 godoc
//
//	@Summary		CAMT.054 Rücklastschriften importieren
//	@Description	Importiert eine CAMT.054 Bankbenachrichtigung und erfasst automatisch Rücklastschriften für die betroffenen Rechnungen.
//	@Tags			SEPA
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			file	formData	file	true	"CAMT.054 XML-Datei"
//	@Success		200		{object}	object	"Ergebnis: matched, not_found, already_returned"
//	@Failure		400		{object}	map[string]string
//	@Failure		422		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/sepa/camt054 [post]
func (h *SEPAHandler) ImportCAMT054(w http.ResponseWriter, r *http.Request) {
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

	// Check EEG access
	_, err = h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		jsonError(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	entries, err := sepa.ParseCamt054(data)
	if err != nil {
		jsonError(w, "failed to parse CAMT.054: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	matched := 0
	notFound := 0
	alreadyReturned := 0

	for _, entry := range entries {
		invoiceID, parseErr := uuid.Parse(entry.EndToEndID)
		if parseErr != nil {
			notFound++
			continue
		}

		inv, fetchErr := h.invoiceRepo.GetByID(r.Context(), invoiceID)
		if fetchErr != nil {
			notFound++
			continue
		}
		// Verify invoice belongs to this EEG
		if inv.EegID != eegID {
			notFound++
			continue
		}

		if inv.SepaReturnAt != nil {
			alreadyReturned++
			continue
		}

		if setErr := h.invoiceRepo.SetSepaReturn(r.Context(), invoiceID, entry.BookingDate, entry.ReasonCode, entry.AdditionalInfo); setErr != nil {
			// Log but continue
			continue
		}
		matched++
	}

	jsonOK(w, map[string]int{
		"matched":          matched,
		"not_found":        notFound,
		"already_returned": alreadyReturned,
	})
}
