package handler

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/billing"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/invoice"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	"github.com/xuri/excelize/v2"
)

type BillingHandler struct {
	billingSvc     *billing.Service
	invoiceRepo    *repository.InvoiceRepository
	billingRunRepo *repository.BillingRunRepository
	memberRepo     *repository.MemberRepository
	eegRepo        *repository.EEGRepository
}

func NewBillingHandler(
	billingSvc *billing.Service,
	invoiceRepo *repository.InvoiceRepository,
	billingRunRepo *repository.BillingRunRepository,
	memberRepo *repository.MemberRepository,
	eegRepo *repository.EEGRepository,
) *BillingHandler {
	return &BillingHandler{
		billingSvc:     billingSvc,
		invoiceRepo:    invoiceRepo,
		billingRunRepo: billingRunRepo,
		memberRepo:     memberRepo,
		eegRepo:        eegRepo,
	}
}

// RunBilling godoc
//
//	@Summary		Abrechnungslauf starten
//	@Description	Startet einen neuen Abrechnungslauf für den angegebenen Zeitraum. Gibt 409 zurück, wenn sich der Zeitraum mit einem bestehenden Lauf überschneidet (außer force=true).
//	@Tags			Abrechnung
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			body	body		object	true	"Abrechnungsparameter"	SchemaExample({"period_start":"2025-01-01","period_end":"2025-12-31","member_ids":[],"billing_type":"","force":false,"preview":false})
//	@Success		200		{object}	object	"Ergebnis mit billing_run, invoices_created, invoices, preview"
//	@Failure		400		{object}	object	"Ungültige Eingabe"
//	@Failure		409		{object}	object	"Zeitraumüberschneidung mit bestehendem Abrechnungslauf"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/run [post]
func (h *BillingHandler) RunBilling(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error

	var req struct {
		PeriodStart string   `json:"period_start"`
		PeriodEnd   string   `json:"period_end"`
		MemberIDs   []string `json:"member_ids"`
		BillingType string   `json:"billing_type"`
		Force       bool     `json:"force"`
		Preview     bool     `json:"preview"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	start, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		jsonError(w, "invalid period_start format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	end, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		jsonError(w, "invalid period_end format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	// Include the full last day
	end = end.Add(24*time.Hour - time.Second)

	var memberIDs []uuid.UUID
	for _, idStr := range req.MemberIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			jsonError(w, "invalid member ID: "+idStr, http.StatusBadRequest)
			return
		}
		memberIDs = append(memberIDs, id)
	}

	opts := billing.RunOptions{
		PeriodStart: start,
		PeriodEnd:   end,
		MemberIDs:   memberIDs,
		BillingType: req.BillingType,
		Force:       req.Force,
		Preview:     req.Preview,
	}

	result, err := h.billingSvc.RunBilling(r.Context(), eeg.ID, opts)
	if err != nil {
		var overlapErr *billing.OverlapError
		if errors.As(err, &overlapErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":                 "period overlaps with existing billing run",
				"existing_run_id":       overlapErr.Existing.ID,
				"existing_period_start": overlapErr.Existing.PeriodStart.Format("2006-01-02"),
				"existing_period_end":   overlapErr.Existing.PeriodEnd.Format("2006-01-02"),
			})
			return
		}
		jsonError(w, "billing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"billing_run":      result.BillingRun,
		"invoices_created": len(result.Invoices),
		"invoices":         result.Invoices,
		"preview":          req.Preview,
	})
}

// GetInvoicePDF godoc
//
//	@Summary		Rechnungs-PDF abrufen
//	@Description	Liefert das PDF einer Rechnung als Binärdatei. Mit ?regenerate=true wird das PDF neu generiert.
//	@Tags			Abrechnung
//	@Produce		application/pdf
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			invoiceID	path		string	true	"Rechnungs-ID (UUID)"
//	@Param			regenerate	query		boolean	false	"PDF neu generieren statt aus Datei lesen"
//	@Success		200			{file}		binary	"PDF-Datei"
//	@Failure		400			{object}	object	"Ungültige ID"
//	@Failure		404			{object}	object	"Rechnung oder PDF nicht gefunden"
//	@Failure		500			{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/invoices/{invoiceID}/pdf [get]
func (h *BillingHandler) GetInvoicePDF(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoiceID"))
	if err != nil {
		jsonError(w, "invalid invoice ID", http.StatusBadRequest)
		return
	}

	// ?regenerate=true — re-run PDF generation with current code and serve fresh bytes
	if r.URL.Query().Get("regenerate") == "true" {
		inv, err := h.invoiceRepo.GetByID(r.Context(), invoiceID)
		if err != nil || inv.EegID != eeg.ID {
			jsonError(w, "invoice not found", http.StatusNotFound)
			return
		}
		pdfData, err := h.billingSvc.RegeneratePDF(r.Context(), invoiceID)
		if err != nil {
			jsonError(w, "regenerate PDF: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"Rechnung_%s.pdf\"", invoiceID.String()[:8]))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
		w.Write(pdfData)
		return
	}

	inv, err := h.invoiceRepo.GetByID(r.Context(), invoiceID)
	if err != nil || inv.EegID != eeg.ID {
		jsonError(w, "invoice not found", http.StatusNotFound)
		return
	}

	if inv.PdfPath == "" {
		jsonError(w, "PDF not available for this invoice", http.StatusNotFound)
		return
	}

	f, err := os.Open(inv.PdfPath)
	if err != nil {
		jsonError(w, "PDF file not found on disk", http.StatusNotFound)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		jsonError(w, "could not stat PDF file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Rechnung_%s.pdf\"", invoiceID.String()[:8]))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	http.ServeContent(w, r, inv.PdfPath, fi.ModTime(), f)
}

// FinalizeBillingRun godoc
//
//	@Summary		Abrechnungslauf abschließen
//	@Description	Setzt einen Abrechnungslauf im Status 'draft' auf 'finalized'. Finalisierte Läufe sind unveränderlich.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			runID	path		string	true	"Abrechnungslauf-ID (UUID)"
//	@Success		200		{object}	object	"Aktualisierter Abrechnungslauf"
//	@Failure		400		{object}	object	"Ungültige ID oder Lauf bereits finalisiert"
//	@Failure		404		{object}	object	"Abrechnungslauf nicht gefunden"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID}/finalize [post]
func (h *BillingHandler) FinalizeBillingRun(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		jsonError(w, "invalid billing run ID", http.StatusBadRequest)
		return
	}

	run, err := h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil || run.EegID != eeg.ID {
		jsonError(w, "billing run not found", http.StatusNotFound)
		return
	}

	if err := h.billingRunRepo.Finalize(r.Context(), runID); err != nil {
		if err.Error() == "billing run not found" {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonError(w, "failed to finalize billing run: "+err.Error(), http.StatusBadRequest)
		return
	}

	run, err = h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil {
		jsonOK(w, map[string]string{"status": "finalized"})
		return
	}
	jsonOK(w, run)
}

// DeleteBillingRun godoc
//
//	@Summary		Abrechnungslauf löschen
//	@Description	Löscht einen Abrechnungslauf im Status 'draft' endgültig, inklusive aller zugehörigen Rechnungen. Nur Entwürfe können gelöscht werden.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path	string	true	"EEG ID (UUID)"
//	@Param			runID	path	string	true	"Abrechnungslauf-ID (UUID)"
//	@Success		204		"Erfolgreich gelöscht"
//	@Failure		400		{object}	object	"Ungültige ID oder Lauf nicht im Draft-Status"
//	@Failure		404		{object}	object	"Abrechnungslauf nicht gefunden"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID} [delete]
func (h *BillingHandler) DeleteBillingRun(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		jsonError(w, "invalid billing run ID", http.StatusBadRequest)
		return
	}

	run, err := h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil || run.EegID != eeg.ID {
		jsonError(w, "billing run not found", http.StatusNotFound)
		return
	}

	if err := h.billingRunRepo.DeleteDraft(r.Context(), runID); err != nil {
		if err.Error() == "billing run not found" {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonError(w, "failed to delete billing run: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CancelBillingRun godoc
//
//	@Summary		Abrechnungslauf stornieren
//	@Description	Storniert einen finalisierten Abrechnungslauf: erzeugt Stornorechnungs-PDFs für alle nicht bezahlten Rechnungen und setzt den Lauf sowie die Rechnungen auf 'cancelled'.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			runID	path		string	true	"Abrechnungslauf-ID (UUID)"
//	@Success		200		{object}	object	"Stornierter Abrechnungslauf"
//	@Failure		400		{object}	object	"Ungültige ID"
//	@Failure		404		{object}	object	"EEG oder Abrechnungslauf nicht gefunden"
//	@Failure		409		{object}	object	"Lauf bereits storniert"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID}/cancel [post]
func (h *BillingHandler) CancelBillingRun(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		jsonError(w, "invalid billing run ID", http.StatusBadRequest)
		return
	}

	run, err := h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil || run.EegID != eeg.ID {
		jsonError(w, "billing run not found", http.StatusNotFound)
		return
	}

	// Get invoices to storno before cancelling
	invoiceIDs, err := h.billingRunRepo.ListNonPaidInvoiceIDs(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to list invoices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate storno PDFs for each non-paid invoice
	for _, invID := range invoiceIDs {
		inv, err := h.invoiceRepo.GetByID(r.Context(), invID)
		if err != nil {
			continue
		}
		member, err := h.memberRepo.GetByID(r.Context(), inv.MemberID)
		if err != nil {
			continue
		}
		pdfData, err := invoice.GenerateStornorechnung(inv, eeg, member)
		if err != nil {
			continue
		}
		pdfDir := fmt.Sprintf("/data/invoices/%s/storno", eeg.ID)
		if err := os.MkdirAll(pdfDir, 0755); err == nil {
			pdfPath := fmt.Sprintf("%s/%s.pdf", pdfDir, invID)
			if err := os.WriteFile(pdfPath, pdfData, 0644); err == nil {
				_ = h.invoiceRepo.UpdateStornoPdfPath(r.Context(), invID, pdfPath)
			}
		}
	}

	if err := h.billingRunRepo.Cancel(r.Context(), runID); err != nil {
		if err.Error() == "billing run is already cancelled" {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		if err.Error() == "billing run not found" {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonError(w, "failed to cancel billing run: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the updated run
	run, err = h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil {
		jsonOK(w, map[string]string{"status": "cancelled"})
		return
	}
	jsonOK(w, run)
}

// ListInvoices godoc
//
//	@Summary		Rechnungen auflisten
//	@Description	Gibt alle Rechnungen einer EEG zurück.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG ID (UUID)"
//	@Success		200		{array}		domain.Invoice		"Liste der Rechnungen"
//	@Failure		400		{object}	object				"Ungültige EEG ID"
//	@Failure		500		{object}	object				"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/invoices [get]
func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	var invoices []domain.Invoice
	var err error
	if r.URL.Query().Get("sepa_returned") == "true" {
		invoices, err = h.invoiceRepo.ListByEegWithReturns(r.Context(), eeg.ID)
	} else {
		invoices, err = h.invoiceRepo.ListByEeg(r.Context(), eeg.ID)
	}
	if err != nil {
		jsonError(w, "failed to list invoices", http.StatusInternalServerError)
		return
	}
	if invoices == nil {
		invoices = []domain.Invoice{}
	}
	jsonOK(w, invoices)
}

// SetSepaReturn godoc
//
//	@Summary		SEPA-Rücklastschrift erfassen oder entfernen
//	@Description	Setzt oder löscht eine SEPA-Rücklastschrift für eine Rechnung. Wenn alle Felder leer sind, wird die Rücklastschrift gelöscht.
//	@Tags			Abrechnung
//	@Accept			json
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			invoiceID	path		string	true	"Rechnungs-ID (UUID)"
//	@Param			body		body		object	false	"Rücklastschrift-Daten"	SchemaExample({"return_at":"2026-04-10","reason":"AC01","note":"Falsche IBAN"})
//	@Success		200			{object}	domain.Invoice	"Aktualisierte Rechnung"
//	@Failure		400			{object}	object			"Ungültige Eingabe"
//	@Failure		500			{object}	object			"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/invoices/{invoiceID}/sepa-return [patch]
func (h *BillingHandler) SetSepaReturn(w http.ResponseWriter, r *http.Request) {
	_, _, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoiceID"))
	if err != nil {
		jsonError(w, "invalid invoice ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ReturnAt string `json:"return_at"`
		Reason   string `json:"reason"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// If all fields empty → clear the return
	if req.ReturnAt == "" && req.Reason == "" && req.Note == "" {
		if err := h.invoiceRepo.ClearSepaReturn(r.Context(), invoiceID); err != nil {
			jsonError(w, "failed to clear sepa return", http.StatusInternalServerError)
			return
		}
	} else {
		var returnAt time.Time
		if req.ReturnAt != "" {
			returnAt, err = time.Parse("2006-01-02", req.ReturnAt)
			if err != nil {
				jsonError(w, "invalid return_at date (expected YYYY-MM-DD)", http.StatusBadRequest)
				return
			}
		} else {
			returnAt = time.Now()
		}
		if err := h.invoiceRepo.SetSepaReturn(r.Context(), invoiceID, returnAt, req.Reason, req.Note); err != nil {
			jsonError(w, "failed to set sepa return", http.StatusInternalServerError)
			return
		}
	}

	inv, err := h.invoiceRepo.GetByID(r.Context(), invoiceID)
	if err != nil {
		jsonError(w, "failed to fetch updated invoice", http.StatusInternalServerError)
		return
	}
	jsonOK(w, inv)
}

// UpdateInvoiceStatus godoc
//
//	@Summary		Rechnungsstatus aktualisieren
//	@Description	Setzt den Status einer Rechnung. Gültige Werte: draft, pending, sent, paid, cancelled.
//	@Tags			Abrechnung
//	@Accept			json
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			invoiceID	path		string	true	"Rechnungs-ID (UUID)"
//	@Param			body		body		object	true	"Neuer Status"	SchemaExample({"status":"paid"})
//	@Success		200			{object}	object	"Gesetzter Status"
//	@Failure		400			{object}	object	"Ungültige ID oder ungültiger Status"
//	@Failure		500			{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/invoices/{invoiceID}/status [patch]
func (h *BillingHandler) UpdateInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoiceID"))
	if err != nil {
		jsonError(w, "invalid invoice ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	validStatuses := map[string]bool{
		"draft": true, "pending": true, "sent": true, "paid": true, "cancelled": true,
	}
	if !validStatuses[req.Status] {
		jsonError(w, "invalid status: must be one of draft, pending, sent, paid, cancelled", http.StatusBadRequest)
		return
	}

	inv, err := h.invoiceRepo.GetByID(r.Context(), invoiceID)
	if err != nil || inv.EegID != eeg.ID {
		jsonError(w, "invoice not found", http.StatusNotFound)
		return
	}

	if err := h.invoiceRepo.UpdateStatus(r.Context(), invoiceID, req.Status); err != nil {
		jsonError(w, "failed to update invoice status", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": req.Status})
}

// ListBillingRuns godoc
//
//	@Summary		Abrechnungsläufe auflisten
//	@Description	Gibt alle Abrechnungsläufe einer EEG zurück.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path		string					true	"EEG ID (UUID)"
//	@Success		200		{array}		domain.BillingRun		"Liste der Abrechnungsläufe"
//	@Failure		400		{object}	object					"Ungültige EEG ID"
//	@Failure		500		{object}	object					"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs [get]
func (h *BillingHandler) ListBillingRuns(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	runs, err := h.billingRunRepo.ListByEeg(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to list billing runs", http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []domain.BillingRun{}
	}
	jsonOK(w, runs)
}

// ListInvoicesByRun godoc
//
//	@Summary		Rechnungen eines Abrechnungslaufs auflisten
//	@Description	Gibt alle Rechnungen zurück, die einem bestimmten Abrechnungslauf zugeordnet sind.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path		string				true	"EEG ID (UUID)"
//	@Param			runID	path		string				true	"Abrechnungslauf-ID (UUID)"
//	@Success		200		{array}		domain.Invoice		"Liste der Rechnungen"
//	@Failure		400		{object}	object				"Ungültige ID"
//	@Failure		500		{object}	object				"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID}/invoices [get]
func (h *BillingHandler) ListInvoicesByRun(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		jsonError(w, "invalid run ID", http.StatusBadRequest)
		return
	}
	run, err := h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil || run.EegID != eeg.ID {
		jsonError(w, "billing run not found", http.StatusNotFound)
		return
	}
	invoices, err := h.invoiceRepo.ListByBillingRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to list invoices", http.StatusInternalServerError)
		return
	}
	if invoices == nil {
		invoices = []domain.Invoice{}
	}
	jsonOK(w, invoices)
}

// SendAllInvoices godoc
//
//	@Summary		Alle Rechnungen versenden
//	@Description	Versendet alle Rechnungen eines Abrechnungslaufs per E-Mail (Route mit runID) oder alle offenen Rechnungen einer EEG (Legacy-Route ohne runID).
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			runID	path		string	false	"Abrechnungslauf-ID (UUID); nur bei der Lauf-Route"
//	@Success		200		{object}	object	"Versandergebnis"
//	@Failure		400		{object}	object	"Ungültige ID"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID}/send-all [post]
func (h *BillingHandler) SendAllInvoices(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error

	var billingRunID *uuid.UUID
	if runIDStr := chi.URLParam(r, "runID"); runIDStr != "" {
		parsed, err := uuid.Parse(runIDStr)
		if err != nil {
			jsonError(w, "invalid billing run ID", http.StatusBadRequest)
			return
		}
		billingRunID = &parsed
		run, runErr := h.billingRunRepo.GetByID(r.Context(), parsed)
		if runErr != nil || run.EegID != eeg.ID {
			jsonError(w, "billing run not found", http.StatusNotFound)
			return
		}
	}

	result, err := h.billingSvc.SendAll(r.Context(), eeg.ID, billingRunID)
	if err != nil {
		jsonError(w, "send all failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, result)
}

// ResendInvoice godoc
//
//	@Summary		Rechnung erneut versenden
//	@Description	Versendet eine einzelne Rechnung erneut per E-Mail an die hinterlegte Mitglieds-E-Mail-Adresse.
//	@Tags			Abrechnung
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			invoiceID	path		string	true	"Rechnungs-ID (UUID)"
//	@Success		200			{object}	object	"Erfolgsmeldung"
//	@Failure		400			{object}	object	"Ungültige ID, kein PDF oder keine E-Mail-Adresse"
//	@Failure		404			{object}	object	"Rechnung, Mitglied oder EEG nicht gefunden"
//	@Failure		500			{object}	object	"Versandfehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/invoices/{invoiceID}/resend [post]
func (h *BillingHandler) ResendInvoice(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoiceID"))
	if err != nil {
		jsonError(w, "invalid invoice ID", http.StatusBadRequest)
		return
	}

	inv, err := h.invoiceRepo.GetByID(r.Context(), invoiceID)
	if err != nil {
		jsonError(w, "invoice not found", http.StatusNotFound)
		return
	}
	if inv.EegID != eeg.ID {
		jsonError(w, "invoice does not belong to this EEG", http.StatusBadRequest)
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), inv.MemberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}

	if inv.PdfPath == "" {
		jsonError(w, "no PDF available for this invoice", http.StatusBadRequest)
		return
	}

	pdfData, err := os.ReadFile(inv.PdfPath)
	if err != nil {
		jsonError(w, "PDF file not found on disk", http.StatusInternalServerError)
		return
	}

	if member.Email == "" {
		jsonError(w, "member has no email address", http.StatusBadRequest)
		return
	}

	if eeg.IsDemo {
		jsonError(w, "email sending disabled in demo mode", http.StatusForbidden)
		return
	}

	smtpCfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}
	if err := invoice.SendInvoice(smtpCfg, member, eeg, inv, pdfData); err != nil {
		jsonError(w, "failed to send invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.invoiceRepo.MarkSent(r.Context(), inv.ID); err != nil {
		// Non-fatal: log but return success since email was sent
		_ = err
	}

	jsonOK(w, map[string]string{"message": "invoice resent successfully"})
}

// ZipBillingRun godoc
//
//	@Summary		Abrechnungslauf als ZIP herunterladen
//	@Description	Erzeugt ein ZIP-Archiv aller Rechnungs-PDFs eines Abrechnungslaufs und streamt es als Download.
//	@Tags			Abrechnung
//	@Produce		application/zip
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			runID	path		string	true	"Abrechnungslauf-ID (UUID)"
//	@Success		200		{file}		binary	"ZIP-Archiv der PDFs"
//	@Failure		400		{object}	object	"Ungültige ID"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID}/zip [get]
func (h *BillingHandler) ZipBillingRun(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		jsonError(w, "invalid run ID", http.StatusBadRequest)
		return
	}
	run, err := h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil || run.EegID != eeg.ID {
		jsonError(w, "billing run not found", http.StatusNotFound)
		return
	}

	invoices, err := h.invoiceRepo.ListByBillingRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to list invoices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Rechnungen_%s.zip\"", runID.String()[:8]))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, inv := range invoices {
		if inv.PdfPath == "" {
			continue
		}
		data, err := os.ReadFile(inv.PdfPath)
		if err != nil {
			continue
		}
		nr := inv.ID.String()[:8]
		if inv.InvoiceNumber != nil {
			nr = fmt.Sprintf("%d", *inv.InvoiceNumber)
		}
		f, err := zw.Create(fmt.Sprintf("Rechnung_%s.pdf", nr))
		if err != nil {
			continue
		}
		_, _ = f.Write(data)
	}
}

// ExportBillingRun godoc
//
//	@Summary		Abrechnungslauf als XLSX exportieren
//	@Description	Gibt eine XLSX-Datei mit allen Rechnungsdaten des Abrechnungslaufs zurück (Rechnungsnummer, Mitglied, Zeitraum, kWh, Betrag, Status).
//	@Tags			Abrechnung
//	@Produce		application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			runID	path		string	true	"Abrechnungslauf-ID (UUID)"
//	@Success		200		{file}		binary	"XLSX-Datei"
//	@Failure		400		{object}	object	"Ungültige ID"
//	@Failure		404		{object}	object	"EEG nicht gefunden"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/billing/runs/{runID}/export [get]
func (h *BillingHandler) ExportBillingRun(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		jsonError(w, "invalid run ID", http.StatusBadRequest)
		return
	}

	run, err := h.billingRunRepo.GetByID(r.Context(), runID)
	if err != nil || run.EegID != eeg.ID {
		jsonError(w, "billing run not found", http.StatusNotFound)
		return
	}

	invoices, err := h.invoiceRepo.ListByBillingRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to list invoices", http.StatusInternalServerError)
		return
	}

	xf := excelize.NewFile()
	sheet := "Abrechnung"
	xf.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Rechnungsnummer", "Mitglied-ID", "Zeitraum von", "Zeitraum bis",
		"Bezug kWh", "Einspeisung kWh", "Betrag (inkl. MwSt.)", "Status", "Versendet am",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		xf.SetCellValue(sheet, cell, h)
	}

	digits := eeg.InvoiceNumberDigits
	if digits <= 0 {
		digits = 4
	}

	for row, inv := range invoices {
		nr := inv.ID.String()[:8]
		if inv.InvoiceNumber != nil {
			nr = fmt.Sprintf("%s%0*d", eeg.InvoiceNumberPrefix, digits, *inv.InvoiceNumber)
		}
		sentAt := ""
		if inv.SentAt != nil {
			sentAt = inv.SentAt.Format("02.01.2006")
		}
		values := []any{
			nr,
			inv.MemberID.String(),
			inv.PeriodStart.Format("02.01.2006"),
			inv.PeriodEnd.Format("02.01.2006"),
			inv.ConsumptionKwh,
			inv.GenerationKwh,
			inv.TotalAmount,
			inv.Status,
			sentAt,
		}
		for col, val := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row+2)
			xf.SetCellValue(sheet, cell, val)
		}
	}

	buf, err := xf.WriteToBuffer()
	if err != nil {
		jsonError(w, "failed to generate XLSX", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Abrechnung_%s.xlsx", strings.ReplaceAll(runID.String()[:8], "-", ""))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	_, _ = w.Write(buf.Bytes())
}
