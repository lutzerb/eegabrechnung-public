package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

var validGranularities = map[string]bool{"day": true, "month": true, "year": true, "15min": true}

var viennaLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// viennaDay returns midnight of the given date in Vienna local time.
// This ensures date-range boundaries align with the same timezone used
// in the database queries (AT TIME ZONE 'Europe/Vienna').
func viennaDay(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, viennaLoc)
}

type ReportHandler struct {
	reportRepo *repository.ReportRepository
	eegRepo    *repository.EEGRepository
}

func NewReportHandler(reportRepo *repository.ReportRepository, eegRepo *repository.EEGRepository) *ReportHandler {
	return &ReportHandler{reportRepo: reportRepo, eegRepo: eegRepo}
}

// GetMonthlyEnergy godoc
//
//	@Summary		Monthly energy report
//	@Description	Returns aggregated energy and financial data grouped by calendar month for the given year. Each row contains consumption/generation kWh and revenue/payout totals.
//	@Tags			Berichte
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			year	query		int		false	"Calendar year (default: current year)"
//	@Success		200		{array}		domain.MonthlyEnergyRow
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/reports/energy [get]
func (h *ReportHandler) GetMonthlyEnergy(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	year := time.Now().Year()
	if y, err := strconv.Atoi(r.URL.Query().Get("year")); err == nil && y > 2000 {
		year = y
	}

	rows, err := h.reportRepo.MonthlyEnergy(r.Context(), eeg.ID, year)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, rows)
}

// GetMemberStats godoc
//
//	@Summary		Per-member energy and invoice statistics
//	@Description	Returns aggregated invoice totals and energy volumes per member over an optional date range. Used for the members breakdown report.
//	@Tags			Berichte
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			from	query		string	false	"Start date (YYYY-MM-DD, inclusive)"
//	@Param			to		query		string	false	"End date (YYYY-MM-DD, inclusive)"
//	@Success		200		{array}		domain.MemberStat
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/reports/members [get]
func (h *ReportHandler) GetMemberStats(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	var from, to *time.Time
	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = &t
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			t = t.Add(24*time.Hour - time.Second)
			to = &t
		}
	}

	stats, err := h.reportRepo.MemberStats(r.Context(), eeg.ID, from, to)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, stats)
}

// GetRawMemberEnergy godoc
//
//	@Summary		Raw per-member energy export
//	@Description	Returns raw energy readings aggregated per member for a date range. Used for CSV/XLSX export of member energy data. Both from and to are required.
//	@Tags			Berichte
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			from	query		string	true	"Start date (YYYY-MM-DD, inclusive, Vienna time)"
//	@Param			to		query		string	true	"End date (YYYY-MM-DD, inclusive, Vienna time)"
//	@Success		200		{array}		domain.MemberStat
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/energy/members [get]
func (h *ReportHandler) GetRawMemberEnergy(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		jsonError(w, "from and to are required", http.StatusBadRequest)
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		jsonError(w, "invalid from date", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		jsonError(w, "invalid to date", http.StatusBadRequest)
		return
	}
	from = viennaDay(from)
	to = viennaDay(to).AddDate(0, 0, 1) // exclusive: start of next day in Vienna time

	var memberID *uuid.UUID
	if s := r.URL.Query().Get("member_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			memberID = &id
		}
	}

	stats, err := h.reportRepo.RawMemberEnergy(r.Context(), eeg.ID, from, to, memberID)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if stats == nil {
		stats = []domain.MemberStat{}
	}
	jsonOK(w, stats)
}

// GetEnergySummary godoc
//
//	@Summary		EEG-wide energy summary
//	@Description	Returns EEG-wide aggregated energy readings (consumption, generation, Restbedarf, Resteinspeisung) bucketed by the requested granularity. Date boundaries are interpreted in Vienna local time (Europe/Vienna).
//	@Tags			Berichte
//	@Produce		json
//	@Param			eegID		path		string	true	"EEG ID (UUID)"
//	@Param			from		query		string	true	"Start date (YYYY-MM-DD, inclusive, Vienna time)"
//	@Param			to			query		string	true	"End date (YYYY-MM-DD, inclusive, Vienna time)"
//	@Param			granularity	query		string	false	"Time bucket size: year, month, day, 15min (default: month)"	Enums(year,month,day,15min)
//	@Success		200			{array}		domain.EnergySummaryRow
//	@Failure		400			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/energy/summary [get]
func (h *ReportHandler) GetEnergySummary(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		jsonError(w, "from and to are required", http.StatusBadRequest)
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		jsonError(w, "invalid from date", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		jsonError(w, "invalid to date", http.StatusBadRequest)
		return
	}
	from = viennaDay(from)
	to = viennaDay(to).AddDate(0, 0, 1) // exclusive: start of next day in Vienna time

	granularity := r.URL.Query().Get("granularity")
	if !validGranularities[granularity] {
		granularity = "month"
	}

	var memberID *uuid.UUID
	if s := r.URL.Query().Get("member_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			memberID = &id
		}
	}

	rows, err := h.reportRepo.EnergySummary(r.Context(), eeg.ID, from, to, granularity, memberID)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []domain.EnergySummaryRow{}
	}
	jsonOK(w, rows)
}

// GetAnnualReport godoc
//
//	@Summary		Annual report (XLSX download)
//	@Description	Generates an XLSX annual report with two sheets: member list at the reference date and per-member energy + billing totals for the date range.
//	@Tags			Berichte
//	@Produce		application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			date	query		string	true	"Member reference date (YYYY-MM-DD) — members active on this date are listed"
//	@Param			from	query		string	true	"Period start (YYYY-MM-DD) for energy and billing"
//	@Param			to		query		string	true	"Period end (YYYY-MM-DD, inclusive) for energy and billing"
//	@Success		200		{file}		application/octet-stream
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/reports/annual [get]
func (h *ReportHandler) GetAnnualReport(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	dateStr := r.URL.Query().Get("date")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if dateStr == "" || fromStr == "" || toStr == "" {
		jsonError(w, "date, from and to are required", http.StatusBadRequest)
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		jsonError(w, "invalid date", http.StatusBadRequest)
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		jsonError(w, "invalid from date", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		jsonError(w, "invalid to date", http.StatusBadRequest)
		return
	}
	from = viennaDay(from)
	to = viennaDay(to).AddDate(0, 0, 1) // exclusive upper bound

	members, err := h.reportRepo.AnnualReport(r.Context(), eeg.ID, date, from, to)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := generateAnnualReportXLSX(eeg.Name, date, from, to, members)
	if err != nil {
		jsonError(w, "xlsx generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("jahresbericht_%s_%s_bis_%s.xlsx",
		eeg.Name, fromStr, toStr)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(data)
}
