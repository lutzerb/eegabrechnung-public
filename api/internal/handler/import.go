package handler

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/importer"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type ImportHandler struct {
	eegRepo        *repository.EEGRepository
	memberRepo     *repository.MemberRepository
	meterPointRepo *repository.MeterPointRepository
	readingRepo    *repository.ReadingRepository
}

func NewImportHandler(
	eegRepo *repository.EEGRepository,
	memberRepo *repository.MemberRepository,
	meterPointRepo *repository.MeterPointRepository,
	readingRepo *repository.ReadingRepository,
) *ImportHandler {
	return &ImportHandler{
		eegRepo:        eegRepo,
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		readingRepo:    readingRepo,
	}
}

// filterByTransitionDate removes readings at or after the EDA transition date
// and returns the filtered slice plus the count of skipped readings.
func filterByTransitionDate(readings []domain.EnergyReading, cutoff *time.Time) ([]domain.EnergyReading, int) {
	if cutoff == nil {
		return readings, 0
	}
	filtered := readings[:0]
	skipped := 0
	for _, r := range readings {
		if !r.Ts.Before(*cutoff) {
			skipped++
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered, skipped
}

// resolvedEnergieDaten is the result of parsing + resolving an Energiedaten XLSX.
type resolvedEnergieDaten struct {
	readings      []domain.EnergyReading
	meterPointIDs []uuid.UUID
	periodStart   time.Time
	periodEnd     time.Time
	skippedMeters int
}

// resolveEnergieDaten parses an Energiedaten XLSX and maps Zählpunkt IDs to meter_point UUIDs.
func (h *ImportHandler) resolveEnergieDaten(ctx context.Context, tmpPath string) (*resolvedEnergieDaten, error) {
	energyRows, err := importer.ParseEnergieDaten(tmpPath)
	if err != nil {
		return nil, err
	}

	zaehlpunktCache := map[string]uuid.UUID{}
	notFound := map[string]bool{}
	mpIDSet := map[uuid.UUID]bool{}

	var readings []domain.EnergyReading
	var minTs, maxTs time.Time
	first := true

	for _, er := range energyRows {
		mpID, ok := zaehlpunktCache[er.MeterID]
		if !ok {
			if notFound[er.MeterID] {
				continue
			}
			mp, err := h.meterPointRepo.GetByZaehlpunkt(ctx, er.MeterID)
			if err != nil {
				slog.Warn("meter point not found, skipping", "zaehlpunkt", er.MeterID)
				notFound[er.MeterID] = true
				continue
			}
			mpID = mp.ID
			zaehlpunktCache[er.MeterID] = mpID
		}
		mpIDSet[mpID] = true
		if first || er.Ts.Before(minTs) {
			minTs = er.Ts
		}
		if first || er.Ts.After(maxTs) {
			maxTs = er.Ts
		}
		first = false
		readings = append(readings, domain.EnergyReading{
			MeterPointID: mpID,
			Ts:           er.Ts,
			WhTotal:      er.WhTotal,
			WhCommunity:  er.WhCommunity,
			WhSelf:       er.WhSelf,
		})
	}

	mpIDs := make([]uuid.UUID, 0, len(mpIDSet))
	for id := range mpIDSet {
		mpIDs = append(mpIDs, id)
	}

	return &resolvedEnergieDaten{
		readings:      readings,
		meterPointIDs: mpIDs,
		periodStart:   minTs,
		periodEnd:     maxTs,
		skippedMeters: len(notFound),
	}, nil
}

// ImportStammdaten godoc
//
//	@Summary		Stammdaten importieren
//	@Description	Importiert Mitglieder und Zählpunkte aus einer XLSX-Datei (multipart/form-data, Feld "file"). Vorhandene Datensätze werden per Upsert aktualisiert.
//	@Tags			Energiedaten
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			file	formData	file	true	"XLSX-Datei mit Stammdaten"
//	@Success		200		{object}	object	"Anzahl importierter Mitglieder und Zählpunkte"
//	@Failure		400		{object}	object	"Ungültige EEG ID, fehlendes Dateifeld oder Parse-Fehler"
//	@Failure		404		{object}	object	"EEG nicht gefunden"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/import/stammdaten [post]
func (h *ImportHandler) ImportStammdaten(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	slog.Info("importing Stammdaten", "filename", header.Filename, "eeg_id", eegID)

	tmpPath, cleanup, err := writeTempFile(file, header.Filename)
	if err != nil {
		jsonError(w, "failed to write temp file", http.StatusInternalServerError)
		return
	}
	defer cleanup()

	rows, err := importer.ParseStammdaten(tmpPath)
	if err != nil {
		jsonError(w, "failed to parse Stammdaten: "+err.Error(), http.StatusBadRequest)
		return
	}

	memberCount := 0
	meterCount := 0
	processedMembers := map[string]uuid.UUID{}

	for _, row := range rows {
		memberID, ok := processedMembers[row.MitgliedsNr]
		if !ok {
			m := &domain.Member{
				EegID:        eegID,
				MitgliedsNr:  row.MitgliedsNr,
				Name1:        row.Name1,
				Name2:        row.Name2,
				Email:        row.Email,
				IBAN:         row.IBAN,
				BusinessRole: row.BusinessRole,
			}
			if m.BusinessRole == "" {
				m.BusinessRole = "privat"
			}
			if err := h.memberRepo.Upsert(r.Context(), m); err != nil {
				slog.Error("failed to upsert member", "mitglieds_nr", row.MitgliedsNr, "error", err)
				jsonError(w, "failed to upsert member: "+err.Error(), http.StatusInternalServerError)
				return
			}
			memberID = m.ID
			processedMembers[row.MitgliedsNr] = memberID
			memberCount++
		}

		mp := &domain.MeterPoint{
			MemberID:            memberID,
			EegID:               eegID,
			Zaehlpunkt:          row.Zaehlpunkt,
			Energierichtung:     row.Energierichtung,
			Verteilungsmodell:   row.Verteilungsmodell,
			ZugeteilteMenugePct: row.ZugeteilteMenugePct,
			Status:              row.Zaehlpunktstatus,
		}
		if row.Verteilungsmodell == "" {
			mp.Verteilungsmodell = "DYNAMIC"
		}
		if row.RegistriertSeit != "" {
			t := parseRegistriertSeit(row.RegistriertSeit)
			if t != nil {
				mp.RegistriertSeit = t
			}
		}

		if err := h.meterPointRepo.Upsert(r.Context(), mp); err != nil {
			slog.Error("failed to upsert meter point", "zaehlpunkt", row.Zaehlpunkt, "error", err)
			jsonError(w, "failed to upsert meter point: "+err.Error(), http.StatusInternalServerError)
			return
		}
		meterCount++
	}

	slog.Info("Stammdaten import complete", "members", memberCount, "meter_points", meterCount)
	jsonOK(w, map[string]any{
		"members":      memberCount,
		"meter_points": meterCount,
	})
}

// PreviewEnergieDaten godoc
//
//	@Summary		Energiedaten-Import vorschau
//	@Description	Parst eine Energiedaten-XLSX und vergleicht sie mit den bestehenden DB-Daten. Gibt Statistiken zu neuen, identischen und konfliktierenden Zeilen zurück, ohne etwas zu speichern.
//	@Tags			Energiedaten
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			file	formData	file	true	"XLSX-Datei mit Energiedaten"
//	@Success		200		{object}	object	"Vorschau-Statistiken (total_rows, new_rows, identical_rows, conflict_rows, skipped_meters, skipped_transition)"
//	@Failure		400		{object}	object	"Ungültige EEG ID, fehlendes Dateifeld oder Parse-Fehler"
//	@Failure		404		{object}	object	"EEG nicht gefunden"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/import/energiedaten/preview [post]
func (h *ImportHandler) PreviewEnergieDaten(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		jsonError(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	slog.Info("previewing Energiedaten", "filename", header.Filename, "eeg_id", eegID)

	tmpPath, cleanup, err := writeTempFile(file, header.Filename)
	if err != nil {
		jsonError(w, "failed to write temp file", http.StatusInternalServerError)
		return
	}
	defer cleanup()

	parsed, err := h.resolveEnergieDaten(r.Context(), tmpPath)
	if err != nil {
		jsonError(w, "failed to parse Energiedaten: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Enforce EDA transition date in preview too.
	var skippedTransition int
	if eeg, eegErr := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); eegErr == nil {
		parsed.readings, skippedTransition = filterByTransitionDate(parsed.readings, eeg.EdaTransitionDate)
	}

	if len(parsed.readings) == 0 {
		jsonOK(w, map[string]any{
			"total_rows": 0, "new_rows": 0, "identical_rows": 0,
			"conflict_rows": 0, "skipped_meters": parsed.skippedMeters,
			"skipped_transition": skippedTransition,
		})
		return
	}

	existing, err := h.readingRepo.GetInPeriod(r.Context(), parsed.meterPointIDs, parsed.periodStart, parsed.periodEnd)
	if err != nil {
		jsonError(w, "failed to check existing readings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	const eps = 0.001
	newRows, identicalRows, conflictRows := 0, 0, 0
	for _, rd := range parsed.readings {
		key := fmt.Sprintf("%s|%d", rd.MeterPointID, rd.Ts.Unix())
		ex, exists := existing[key]
		if !exists {
			newRows++
			continue
		}
		if math.Abs(ex.WhTotal-rd.WhTotal) < eps &&
			math.Abs(ex.WhCommunity-rd.WhCommunity) < eps &&
			math.Abs(ex.WhSelf-rd.WhSelf) < eps {
			identicalRows++
		} else {
			conflictRows++
		}
	}

	resp := map[string]any{
		"total_rows":         len(parsed.readings),
		"new_rows":           newRows,
		"identical_rows":     identicalRows,
		"conflict_rows":      conflictRows,
		"skipped_meters":     parsed.skippedMeters,
		"skipped_transition": skippedTransition,
	}
	if !parsed.periodStart.IsZero() {
		resp["period_start"] = parsed.periodStart.Format("2006-01-02")
		resp["period_end"] = parsed.periodEnd.Format("2006-01-02")
	}
	jsonOK(w, resp)
}

// ImportEnergieDaten godoc
//
//	@Summary		Energiedaten importieren
//	@Description	Importiert Energiemesswerte aus einer Energiedaten-XLSX. Mit ?mode=overwrite (Standard) werden bestehende Werte überschrieben; mit ?mode=skip werden Konflikte übersprungen. Messwerte ab dem EDA-Übergangsdatum werden automatisch verworfen.
//	@Tags			Energiedaten
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			mode	query		string	false	"Konfliktbehandlung: overwrite (Standard) oder skip"	Enums(overwrite, skip)
//	@Param			file	formData	file	true	"XLSX-Datei mit Energiedaten"
//	@Success		200		{object}	object	"Importergebnis (rows_parsed, rows_inserted, skipped_meters, mode)"
//	@Failure		400		{object}	object	"Ungültige EEG ID, fehlendes Dateifeld oder Parse-Fehler"
//	@Failure		404		{object}	object	"EEG nicht gefunden"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/import/energiedaten [post]
func (h *ImportHandler) ImportEnergieDaten(w http.ResponseWriter, r *http.Request) {
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

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "overwrite"
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		jsonError(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	slog.Info("importing Energiedaten", "filename", header.Filename, "eeg_id", eegID, "mode", mode)

	tmpPath, cleanup, err := writeTempFile(file, header.Filename)
	if err != nil {
		jsonError(w, "failed to write temp file", http.StatusInternalServerError)
		return
	}
	defer cleanup()

	parsed, err := h.resolveEnergieDaten(r.Context(), tmpPath)
	if err != nil {
		jsonError(w, "failed to parse Energiedaten: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Enforce EDA transition date: reject XLSX readings at or after the cutoff.
	var skippedTransition int
	if eeg, eegErr := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); eegErr == nil {
		parsed.readings, skippedTransition = filterByTransitionDate(parsed.readings, eeg.EdaTransitionDate)
	}

	var inserted int
	if mode == "skip" {
		inserted, err = h.readingRepo.BulkInsertSkipExisting(r.Context(), parsed.readings)
	} else {
		inserted, err = h.readingRepo.BulkUpsert(r.Context(), parsed.readings)
	}
	if err != nil {
		jsonError(w, "failed to insert readings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Energiedaten import complete",
		"rows_parsed", len(parsed.readings)+skippedTransition,
		"rows_inserted", inserted,
		"skipped_meters", parsed.skippedMeters,
		"skipped_transition", skippedTransition,
		"mode", mode,
	)
	resp := map[string]any{
		"rows_parsed":    len(parsed.readings) + skippedTransition,
		"rows_inserted":  inserted,
		"skipped_meters": parsed.skippedMeters,
		"mode":           mode,
	}
	if skippedTransition > 0 {
		resp["skipped_transition"] = skippedTransition
	}
	jsonOK(w, resp)
}

// GetCoverage godoc
//
//	@Summary		Datenabdeckung abrufen
//	@Description	Gibt für jedes Kalendertag des angegebenen Jahres an, ob Messwerte für die EEG vorhanden sind. Nützlich für die Darstellung der Datenverfügbarkeit vor einem Import.
//	@Tags			Energiedaten
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			year	query		integer	false	"Jahr (Standard: aktuelles Jahr)"
//	@Success		200		{object}	object	"Abdeckungsdaten (year, days[], optional gruendungsdatum)"
//	@Failure		400		{object}	object	"Ungültige EEG ID"
//	@Failure		404		{object}	object	"EEG nicht gefunden"
//	@Failure		500		{object}	object	"Interner Fehler"
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/readings/coverage [get]
func (h *ImportHandler) GetCoverage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	days, err := h.readingRepo.GetCoverageByYear(r.Context(), eegID, year)
	if err != nil {
		jsonError(w, "failed to get coverage: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if days == nil {
		days = []repository.CoverageDay{}
	}

	resp := map[string]any{
		"year": year,
		"days": days,
	}
	if eeg.Gruendungsdatum != nil {
		resp["gruendungsdatum"] = eeg.Gruendungsdatum.Format("2006-01-02")
	}
	jsonOK(w, resp)
}

func parseRegistriertSeit(s string) *time.Time {
	s = strings.TrimSpace(s)
	formats := []string{"2.1.2006", "1.1.2006", "2006-01-02", "01.01.2006"}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return &t
		}
	}
	return nil
}
