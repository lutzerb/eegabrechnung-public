package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type EAHandler struct {
	eaRepo     *repository.EARepository
	invoiceDir string // base dir for beleg storage (reuses invoice dir)
}

func NewEAHandler(eaRepo *repository.EARepository, invoiceDir string) *EAHandler {
	return &EAHandler{eaRepo: eaRepo, invoiceDir: invoiceDir}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *EAHandler) parseEegID(r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "eegID"))
	return id, err == nil
}

func (h *EAHandler) ensureSeeded(w http.ResponseWriter, r *http.Request, eegID uuid.UUID) bool {
	exists, err := h.eaRepo.KontenExists(r.Context(), eegID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return false
	}
	if !exists {
		if err := h.eaRepo.SeedDefaultKonten(r.Context(), eegID); err != nil {
			jsonError(w, "seed konten: "+err.Error(), http.StatusInternalServerError)
			return false
		}
	}
	return true
}

func parseOptDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseYear(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

// ── Settings ──────────────────────────────────────────────────────────────────

func (h *EAHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	s, err := h.eaRepo.GetSettings(r.Context(), eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, s)
}

func (h *EAHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req domain.EASettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	req.EegID = eegID
	if req.UvaPeriodentyp != "MONAT" && req.UvaPeriodentyp != "QUARTAL" {
		req.UvaPeriodentyp = "QUARTAL"
	}
	if err := h.eaRepo.UpdateSettings(r.Context(), &req); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w, req)
}

// ── Konten ────────────────────────────────────────────────────────────────────

func (h *EAHandler) ListKonten(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if !h.ensureSeeded(w, r, eegID) {
		return
	}
	nurAktiv := r.URL.Query().Get("aktiv") != "false"
	konten, err := h.eaRepo.ListKonten(r.Context(), eegID, nurAktiv)
	if err != nil {
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if konten == nil {
		konten = []domain.EAKonto{}
	}
	jsonOK(w, konten)
}

type kontoRequest struct {
	Nummer         string   `json:"nummer"`
	Name           string   `json:"name"`
	Typ            string   `json:"typ"`
	UstRelevanz    string   `json:"ust_relevanz"`
	StandardUstPct *float64 `json:"standard_ust_pct"`
	UvaKZ          string   `json:"uva_kz"`
	K1KZ           string   `json:"k1_kz"`
	Sortierung     int      `json:"sortierung"`
	Aktiv          *bool    `json:"aktiv"`
}

func (h *EAHandler) CreateKonto(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req kontoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Nummer == "" || req.Name == "" {
		jsonError(w, "nummer and name required", http.StatusBadRequest)
		return
	}
	aktiv := true
	if req.Aktiv != nil {
		aktiv = *req.Aktiv
	}
	k := &domain.EAKonto{
		EegID:          eegID,
		Nummer:         req.Nummer,
		Name:           req.Name,
		Typ:            req.Typ,
		UstRelevanz:    req.UstRelevanz,
		StandardUstPct: req.StandardUstPct,
		UvaKZ:          req.UvaKZ,
		K1KZ:           req.K1KZ,
		Sortierung:     req.Sortierung,
		Aktiv:          aktiv,
	}
	if err := h.eaRepo.CreateKonto(r.Context(), k); err != nil {
		jsonError(w, "create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, k)
}

func (h *EAHandler) UpdateKonto(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	kontoID, err := uuid.Parse(chi.URLParam(r, "kontoID"))
	if err != nil {
		jsonError(w, "invalid konto ID", http.StatusBadRequest)
		return
	}
	existing, err := h.eaRepo.GetKonto(r.Context(), kontoID)
	if err != nil || existing.EegID != eegID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	var req kontoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Nummer != "" {
		existing.Nummer = req.Nummer
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Typ != "" {
		existing.Typ = req.Typ
	}
	if req.UstRelevanz != "" {
		existing.UstRelevanz = req.UstRelevanz
	}
	existing.StandardUstPct = req.StandardUstPct
	existing.UvaKZ = req.UvaKZ
	existing.K1KZ = req.K1KZ
	existing.Sortierung = req.Sortierung
	if req.Aktiv != nil {
		existing.Aktiv = *req.Aktiv
	}
	if err := h.eaRepo.UpdateKonto(r.Context(), existing); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w, existing)
}

func (h *EAHandler) DeleteKonto(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	kontoID, err := uuid.Parse(chi.URLParam(r, "kontoID"))
	if err != nil {
		jsonError(w, "invalid konto ID", http.StatusBadRequest)
		return
	}
	if err := h.eaRepo.DeleteKonto(r.Context(), kontoID, eegID); err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Buchungen ─────────────────────────────────────────────────────────────────

type buchungRequest struct {
	ZahlungDatum string  `json:"zahlung_datum"`
	BelegDatum   string  `json:"beleg_datum"`
	Belegnr      string  `json:"belegnr"`
	Beschreibung string  `json:"beschreibung"`
	KontoID      string  `json:"konto_id"`
	BetragBrutto float64 `json:"betrag_brutto"`
	UstCode      string  `json:"ust_code"`
	Gegenseite   string  `json:"gegenseite"`
	Notizen      string  `json:"notizen"`
	BelegID      string  `json:"beleg_id"`
	Reason       string  `json:"reason"` // BAO §131: Änderungsgrund
}

// changedByFromContext extracts the user UUID string from JWT claims, or "system" as fallback.
func changedByFromContext(r *http.Request) string {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return "system"
	}
	return claims.Subject
}

func (h *EAHandler) ListBuchungen(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	var filter repository.BuchungFilter
	filter.Geschaeftsjahr = parseYear(q.Get("jahr"), 0)
	if v := q.Get("von"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.Von = t
		}
	}
	if v := q.Get("bis"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.Bis = t
		}
	}
	if v := q.Get("konto_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.KontoID = id
		}
	}
	filter.Richtung = q.Get("richtung")
	filter.NurBezahlt = q.Get("bezahlt") == "true"
	filter.NurOffen = q.Get("offen") == "true"
	filter.InclDeleted = q.Get("incl_deleted") == "true"

	buchungen, err := h.eaRepo.ListBuchungen(r.Context(), eegID, filter)
	if err != nil {
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Enrich with Konto data
	kontenCache := map[uuid.UUID]*domain.EAKonto{}
	for i := range buchungen {
		if _, ok := kontenCache[buchungen[i].KontoID]; !ok {
			k, err := h.eaRepo.GetKonto(r.Context(), buchungen[i].KontoID)
			if err == nil {
				kontenCache[buchungen[i].KontoID] = k
			}
		}
		buchungen[i].Konto = kontenCache[buchungen[i].KontoID]
	}
	if buchungen == nil {
		buchungen = []domain.EABuchung{}
	}
	jsonOK(w, buchungen)
}

func (h *EAHandler) GetBuchung(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	buchungID, err := uuid.Parse(chi.URLParam(r, "buchungID"))
	if err != nil {
		jsonError(w, "invalid buchung ID", http.StatusBadRequest)
		return
	}
	b, err := h.eaRepo.GetBuchung(r.Context(), buchungID, eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if k, err := h.eaRepo.GetKonto(r.Context(), b.KontoID); err == nil {
		b.Konto = k
	}
	b.Belege, _ = h.eaRepo.ListBelegeForBuchung(r.Context(), b.ID)
	jsonOK(w, b)
}

func (h *EAHandler) CreateBuchung(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req buchungRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Beschreibung == "" {
		jsonError(w, "beschreibung required", http.StatusBadRequest)
		return
	}
	kontoID, err := uuid.Parse(req.KontoID)
	if err != nil {
		jsonError(w, "invalid konto_id", http.StatusBadRequest)
		return
	}
	konto, err := h.eaRepo.GetKonto(r.Context(), kontoID)
	if err != nil || konto.EegID != eegID {
		jsonError(w, "konto not found", http.StatusBadRequest)
		return
	}
	if req.UstCode == "" {
		req.UstCode = "KEINE"
		if konto.UstRelevanz == "RC" && konto.StandardUstPct != nil {
			if *konto.StandardUstPct == 13 {
				req.UstCode = "RC_13"
			} else {
				req.UstCode = "RC_20"
			}
		} else if konto.UstRelevanz == "VST" {
			req.UstCode = "VST_20"
		} else if konto.UstRelevanz == "STEUERBAR" {
			req.UstCode = "UST_20"
		}
	}
	zahlungDatum, _ := parseOptDate(req.ZahlungDatum)
	belegDatum, _ := parseOptDate(req.BelegDatum)
	var belegID *uuid.UUID
	if req.BelegID != "" {
		id, err := uuid.Parse(req.BelegID)
		if err == nil {
			belegID = &id
		}
	}

	pct, ust, netto := repository.CalcUSt(req.BetragBrutto, req.UstCode)
	var ustPct *float64
	if pct > 0 {
		p := pct
		ustPct = &p
	}

	claims := auth.ClaimsFromContext(r.Context())
	var erstelltVon *uuid.UUID
	if claims != nil {
		id, err := uuid.Parse(claims.Subject)
		if err == nil {
			erstelltVon = &id
		}
	}

	jahr := time.Now().Year()
	if zahlungDatum != nil {
		jahr = zahlungDatum.Year()
	}
	nr, _ := h.eaRepo.NextBuchungsnr(r.Context(), eegID, jahr)

	b := &domain.EABuchung{
		EegID:          eegID,
		Geschaeftsjahr: jahr,
		Buchungsnr:     nr,
		ZahlungDatum:   zahlungDatum,
		BelegDatum:     belegDatum,
		Belegnr:        req.Belegnr,
		Beschreibung:   req.Beschreibung,
		KontoID:        kontoID,
		Richtung:       konto.Typ,
		BetragBrutto:   req.BetragBrutto,
		UstCode:        req.UstCode,
		UstPct:         ustPct,
		UstBetrag:      ust,
		BetragNetto:    netto,
		Gegenseite:     req.Gegenseite,
		Quelle:         "manual",
		BelegID:        belegID,
		Notizen:        req.Notizen,
		ErstelltVon:    erstelltVon,
	}
	if b.Richtung == "SONSTIG" {
		b.Richtung = "AUSGABE"
	}
	if err := h.eaRepo.CreateBuchung(r.Context(), b, changedByFromContext(r)); err != nil {
		jsonError(w, "create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	b.Konto = konto
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, b)
}

func (h *EAHandler) UpdateBuchung(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	buchungID, err := uuid.Parse(chi.URLParam(r, "buchungID"))
	if err != nil {
		jsonError(w, "invalid buchung ID", http.StatusBadRequest)
		return
	}
	existing, err := h.eaRepo.GetBuchung(r.Context(), buchungID, eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	var req buchungRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Beschreibung != "" {
		existing.Beschreibung = req.Beschreibung
	}
	zahlungDatum, _ := parseOptDate(req.ZahlungDatum)
	belegDatum, _ := parseOptDate(req.BelegDatum)
	existing.ZahlungDatum = zahlungDatum
	existing.BelegDatum = belegDatum
	if req.Belegnr != "" {
		existing.Belegnr = req.Belegnr
	}
	existing.Gegenseite = req.Gegenseite
	existing.Notizen = req.Notizen
	if req.BetragBrutto > 0 {
		existing.BetragBrutto = req.BetragBrutto
	}
	if req.UstCode != "" {
		existing.UstCode = req.UstCode
	}
	if req.BelegID != "" {
		id, err := uuid.Parse(req.BelegID)
		if err == nil {
			existing.BelegID = &id
		}
	}
	pct, ust, netto := repository.CalcUSt(existing.BetragBrutto, existing.UstCode)
	var ustPct *float64
	if pct > 0 {
		p := pct
		ustPct = &p
	}
	existing.UstPct = ustPct
	existing.UstBetrag = ust
	existing.BetragNetto = netto
	if err := h.eaRepo.UpdateBuchung(r.Context(), existing, changedByFromContext(r), req.Reason); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w, existing)
}

func (h *EAHandler) DeleteBuchung(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	buchungID, err := uuid.Parse(chi.URLParam(r, "buchungID"))
	if err != nil {
		jsonError(w, "invalid buchung ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	// Body is optional — ignore decode errors (DELETE may have no body)
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.eaRepo.DeleteBuchung(r.Context(), buchungID, eegID, changedByFromContext(r), req.Reason); err != nil {
		jsonError(w, "delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Changelog (BAO §131) ──────────────────────────────────────────────────────

func (h *EAHandler) GetBuchungChangelog(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	buchungID, err := uuid.Parse(chi.URLParam(r, "buchungID"))
	if err != nil {
		jsonError(w, "invalid buchung ID", http.StatusBadRequest)
		return
	}
	entries, err := h.eaRepo.GetBuchungChangelog(r.Context(), buchungID, eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if entries == nil {
		entries = []domain.EABuchungChangelog{}
	}
	jsonOK(w, entries)
}

func (h *EAHandler) ListChangelog(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	var f repository.ChangelogFilter
	if v := q.Get("von"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.Von = t
		}
	}
	if v := q.Get("bis"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.Bis = t
		}
	}
	f.ChangedBy = q.Get("user")
	f.Operation = q.Get("operation")
	if v := q.Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &f.Limit)
	}
	if v := q.Get("offset"); v != "" {
		fmt.Sscanf(v, "%d", &f.Offset)
	}

	entries, err := h.eaRepo.ListChangelog(r.Context(), eegID, f)
	if err != nil {
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []domain.EABuchungChangelog{}
	}
	jsonOK(w, entries)
}

// ── Belege ────────────────────────────────────────────────────────────────────

func (h *EAHandler) UploadBeleg(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("datei")
	if err != nil {
		jsonError(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dir := filepath.Join(h.invoiceDir, "belege", eegID.String())
	if err := os.MkdirAll(dir, 0755); err != nil {
		jsonError(w, "storage error", http.StatusInternalServerError)
		return
	}
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	dest := filepath.Join(dir, filename)
	f, err := os.Create(dest)
	if err != nil {
		jsonError(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	size, _ := io.Copy(f, file)

	var buchungID *uuid.UUID
	if v := r.FormValue("buchung_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			buchungID = &id
		}
	}
	sz := int(size)
	beleg := &domain.EABeleg{
		EegID:     eegID,
		BuchungID: buchungID,
		Dateiname: header.Filename,
		Pfad:      dest,
		Groesse:   &sz,
		MimeTyp:   header.Header.Get("Content-Type"),
		Beschreibung: r.FormValue("beschreibung"),
	}
	claims := auth.ClaimsFromContext(r.Context())
	if claims != nil {
		if id, err := uuid.Parse(claims.Subject); err == nil {
			beleg.HochgeladenVon = &id
		}
	}
	if err := h.eaRepo.CreateBeleg(r.Context(), beleg); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if buchungID != nil {
		_ = h.eaRepo.LinkBelegToBuchung(r.Context(), beleg.ID, *buchungID, eegID)
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, beleg)
}

func (h *EAHandler) GetBeleg(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	belegID, err := uuid.Parse(chi.URLParam(r, "belegID"))
	if err != nil {
		jsonError(w, "invalid beleg ID", http.StatusBadRequest)
		return
	}
	beleg, err := h.eaRepo.GetBeleg(r.Context(), belegID, eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	f, err := os.Open(beleg.Pfad)
	if err != nil {
		jsonError(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	ct := beleg.MimeTyp
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, beleg.Dateiname))
	io.Copy(w, f)
}

func (h *EAHandler) DeleteBeleg(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	belegID, err := uuid.Parse(chi.URLParam(r, "belegID"))
	if err != nil {
		jsonError(w, "invalid beleg ID", http.StatusBadRequest)
		return
	}
	pfad, err := h.eaRepo.DeleteBeleg(r.Context(), belegID, eegID)
	if err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	_ = os.Remove(pfad)
	w.WriteHeader(http.StatusNoContent)
}

// ── Reports ────────────────────────────────────────────────────────────────────

func (h *EAHandler) GetSaldenliste(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	von, _ := parseOptDate(q.Get("von"))
	bis, _ := parseOptDate(q.Get("bis"))
	if jahr := parseYear(q.Get("jahr"), 0); jahr > 0 && von == nil && bis == nil {
		v := time.Date(jahr, 1, 1, 0, 0, 0, 0, time.UTC)
		b := time.Date(jahr, 12, 31, 23, 59, 59, 0, time.UTC)
		von, bis = &v, &b
	}
	rows, err := h.eaRepo.Saldenliste(r.Context(), eegID, von, bis)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	if q.Get("format") == "xlsx" {
		data, err := EASaldenlisteXLSX(rows, von, bis)
		if err != nil {
			jsonError(w, "xlsx: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="saldenliste.xlsx"`)
		w.Write(data)
		return
	}
	if rows == nil {
		rows = []domain.EASaldenlisteEintrag{}
	}
	jsonOK(w, rows)
}

func (h *EAHandler) GetKontenblatt(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	kontoID, err := uuid.Parse(chi.URLParam(r, "kontoID"))
	if err != nil {
		jsonError(w, "invalid konto ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	von, _ := parseOptDate(q.Get("von"))
	bis, _ := parseOptDate(q.Get("bis"))
	kb, err := h.eaRepo.Kontenblatt(r.Context(), eegID, kontoID, von, bis)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, kb)
}

func (h *EAHandler) GetJahresabschluss(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	jahr := parseYear(r.URL.Query().Get("jahr"), time.Now().Year())
	ja, err := h.eaRepo.Jahresabschluss(r.Context(), eegID, jahr)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("format") == "xlsx" {
		data, err := EAJahresabschlussXLSX(ja)
		if err != nil {
			jsonError(w, "xlsx: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="jahresabschluss_%d.xlsx"`, jahr))
		w.Write(data)
		return
	}
	jsonOK(w, ja)
}

// ── UVA ───────────────────────────────────────────────────────────────────────

func (h *EAHandler) ListUVA(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	jahr := parseYear(r.URL.Query().Get("jahr"), 0)
	periods, err := h.eaRepo.ListUVA(r.Context(), eegID, jahr)
	if err != nil {
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if periods == nil {
		periods = []domain.EAUVAPeriode{}
	}
	jsonOK(w, periods)
}

func (h *EAHandler) GetUVAKennzahlen(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	uvaID, err := uuid.Parse(chi.URLParam(r, "uvaID"))
	if err != nil {
		jsonError(w, "invalid UVA ID", http.StatusBadRequest)
		return
	}
	uva, err := h.eaRepo.GetUVA(r.Context(), uvaID, eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	// Recalculate live
	live, err := h.eaRepo.CalcUVAKennzahlen(r.Context(), eegID, uva.DatumVon, uva.DatumBis)
	if err != nil {
		jsonError(w, "calc failed", http.StatusInternalServerError)
		return
	}
	live.ID = uva.ID
	live.EegID = eegID
	live.Jahr = uva.Jahr
	live.Periodentyp = uva.Periodentyp
	live.PeriodeNr = uva.PeriodeNr
	live.Status = uva.Status
	live.EingereichtAm = uva.EingereichtAm
	live.PeriodeLabel = uva.PeriodeLabel
	jsonOK(w, live)
}

func (h *EAHandler) CreateUVAPeriode(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Jahr        int    `json:"jahr"`
		Periodentyp string `json:"periodentyp"`
		PeriodeNr   int    `json:"periode_nr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	von, bis := uvaPeriodeDates(req.Jahr, req.Periodentyp, req.PeriodeNr)
	u := &domain.EAUVAPeriode{
		EegID:       eegID,
		Jahr:        req.Jahr,
		Periodentyp: req.Periodentyp,
		PeriodeNr:   req.PeriodeNr,
		DatumVon:    von,
		DatumBis:    bis,
		Status:      "entwurf",
	}
	if err := h.eaRepo.UpsertUVA(r.Context(), u); err != nil {
		jsonError(w, "create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	u.PeriodeLabel = uvaPeriodeLabelFromParts(req.Jahr, req.Periodentyp, req.PeriodeNr)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, u)
}

func (h *EAHandler) SetUVAEingereicht(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	uvaID, err := uuid.Parse(chi.URLParam(r, "uvaID"))
	if err != nil {
		jsonError(w, "invalid UVA ID", http.StatusBadRequest)
		return
	}
	if err := h.eaRepo.SetUVAEingereicht(r.Context(), uvaID, eegID); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EAHandler) ExportUVAXML(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	uvaID, err := uuid.Parse(chi.URLParam(r, "uvaID"))
	if err != nil {
		jsonError(w, "invalid UVA ID", http.StatusBadRequest)
		return
	}
	uva, err := h.eaRepo.GetUVA(r.Context(), uvaID, eegID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	// Recalculate live kennzahlen for export
	live, err := h.eaRepo.CalcUVAKennzahlen(r.Context(), eegID, uva.DatumVon, uva.DatumBis)
	if err != nil {
		jsonError(w, "calc failed", http.StatusInternalServerError)
		return
	}
	live.ID = uva.ID
	live.Jahr = uva.Jahr
	live.Periodentyp = uva.Periodentyp
	live.PeriodeNr = uva.PeriodeNr

	settings, _ := h.eaRepo.GetSettings(r.Context(), eegID)

	format := r.URL.Query().Get("format")
	if format == "xml" || format == "finanz-online-xml" {
		xmlData, err := EAUVAFinanzOnlineXML(live, settings)
		if err != nil {
			jsonError(w, "xml: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="uva_%d_%d.xml"`, uva.Jahr, uva.PeriodeNr))
		w.Write(xmlData)
		return
	}
	// default: JSON summary
	jsonOK(w, live)
}

// ── Import from EEG invoices ──────────────────────────────────────────────────

func (h *EAHandler) ImportPreview(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	von, _ := parseOptDate(q.Get("von"))
	bis, _ := parseOptDate(q.Get("bis"))

	rows, err := h.buildImportPreview(r, eegID, von, bis)
	if err != nil {
		jsonError(w, "preview failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, rows)
}

func (h *EAHandler) ImportRechnungen(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req struct {
		InvoiceIDs []string `json:"invoice_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	selectedIDs := make(map[string]bool, len(req.InvoiceIDs))
	for _, id := range req.InvoiceIDs {
		selectedIDs[id] = true
	}

	// Load all preview rows (no date filter — selection drives what gets imported)
	rows, err := h.buildImportPreview(r, eegID, nil, nil)
	if err != nil {
		jsonError(w, "import failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !h.ensureSeeded(w, r, eegID) {
		return
	}
	konten, err := h.eaRepo.ListKonten(r.Context(), eegID, true)
	if err != nil {
		jsonError(w, "load konten failed", http.StatusInternalServerError)
		return
	}
	kontenByNr := map[string]domain.EAKonto{}
	for _, k := range konten {
		kontenByNr[k.Nummer] = k
	}

	result := domain.EAImportResult{}
	for _, row := range rows {
		// Only import rows whose invoice_id was explicitly selected
		if !selectedIDs[row.InvoiceID.String()] {
			continue
		}
		if row.AlreadyImported {
			result.Skipped++
			continue
		}
		konto, exists := kontenByNr[row.KontoNummer]
		if !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("konto %s not found for invoice %s", row.KontoNummer, row.InvoiceNr))
			continue
		}
		pct, ust, netto := repository.CalcUSt(row.BetragBrutto, row.UstCode)
		var ustPct *float64
		if pct > 0 {
			p := pct
			ustPct = &p
		}
		qid := row.InvoiceID
		jahr := row.Datum.Year()
		nr, _ := h.eaRepo.NextBuchungsnr(r.Context(), eegID, jahr)
		b := &domain.EABuchung{
			EegID:          eegID,
			Geschaeftsjahr: jahr,
			Buchungsnr:     nr,
			BelegDatum:     &row.Datum,
			Belegnr:        row.InvoiceNr,
			Beschreibung:   row.Beschreibung,
			KontoID:        konto.ID,
			Richtung:       konto.Typ,
			BetragBrutto:   row.BetragBrutto,
			UstCode:        row.UstCode,
			UstPct:         ustPct,
			UstBetrag:      ust,
			BetragNetto:    netto,
			Gegenseite:     row.MitgliedName,
			Quelle:         "eeg_rechnung",
			QuelleID:       &qid,
		}
		if b.Richtung == "SONSTIG" {
			b.Richtung = "AUSGABE"
		}
		if err := h.eaRepo.CreateBuchung(r.Context(), b, changedByFromContext(r)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %s: %v", row.InvoiceNr, err))
			continue
		}
		// Auto-attach the invoice PDF as a Beleg (only for rows that carry the PDF path,
		// i.e. not the einspeisung half of a prosumer split to avoid duplicate Belege).
		if row.PdfPath != "" {
			if fi, statErr := os.Stat(row.PdfPath); statErr == nil {
				sz := int(fi.Size())
				beleg := &domain.EABeleg{
					EegID:     eegID,
					BuchungID: &b.ID,
					Dateiname: filepath.Base(row.PdfPath),
					Pfad:      row.PdfPath,
					Groesse:   &sz,
					MimeTyp:   "application/pdf",
				}
				_ = h.eaRepo.CreateBeleg(r.Context(), beleg)
			}
		}
		result.Imported++
	}
	jsonOK(w, result)
}

// buildImportPreview queries invoices and maps them to EA import rows.
func (h *EAHandler) buildImportPreview(r *http.Request, eegID uuid.UUID, von, bis *time.Time) ([]domain.EAImportPreviewRow, error) {
	// Query invoices directly from the database via the invoice handler's repo
	// We use a raw query here since we don't inject invoiceRepo
	rows, err := h.eaRepo.QueryInvoicesForImport(r.Context(), eegID, von, bis)
	if err != nil {
		return nil, fmt.Errorf("query invoices: %w", err)
	}
	return rows, nil
}

// ── Bank import ───────────────────────────────────────────────────────────────

func (h *EAHandler) ImportBank(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		jsonError(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	format := strings.ToUpper(r.FormValue("format"))
	if format == "" {
		switch strings.ToLower(filepath.Ext(header.Filename)) {
		case ".sta", ".mt940":
			format = "MT940"
		case ".xml":
			format = "CAMT053"
		default:
			format = "MT940"
		}
	}

	var transakts []domain.EABankTransaktion
	switch format {
	case "CAMT053":
		transakts, err = ParseCAMT053(data, eegID)
	default: // MT940
		transakts, err = ParseMT940(data, eegID)
	}
	if err != nil {
		jsonError(w, "parse error: "+err.Error(), http.StatusBadRequest)
		return
	}

	imported := 0
	for i := range transakts {
		// Auto-match
		betrag := transakts[i].Betrag
		candidates, _ := h.eaRepo.FindMatchCandidates(r.Context(), eegID, betrag, transakts[i].Buchungsdatum)
		if len(candidates) > 0 {
			c := candidates[0]
			// Check amount match within 0.1%
			cBetrag := c.BetragBrutto
			if betrag < 0 {
				cBetrag = -cBetrag
			}
			diff := betrag - cBetrag
			if diff < 0 {
				diff = -diff
			}
			konfidenz := 0.0
			if diff/cBetrag < 0.001 {
				konfidenz = 0.90
			}
			if konfidenz >= 0.90 {
				transakts[i].MatchedBuchungID = &c.ID
				transakts[i].MatchKonfidenz = &konfidenz
				transakts[i].MatchStatus = "auto"
			}
		}
		if err := h.eaRepo.InsertBankTransaktion(r.Context(), &transakts[i]); err != nil {
			slog.Error("insert bank transaction", "error", err)
			continue
		}
		imported++
	}
	jsonOK(w, map[string]any{"imported": imported, "total": len(transakts)})
}

func (h *EAHandler) ListBankTransaktionen(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	status := r.URL.Query().Get("status")
	result, err := h.eaRepo.ListBankTransaktionen(r.Context(), eegID, status)
	if err != nil {
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if result == nil {
		result = []domain.EABankTransaktion{}
	}
	jsonOK(w, result)
}

func (h *EAHandler) BestaetigeMatch(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	var req []struct {
		TransaktionID string `json:"transaktion_id"`
		BuchungID     string `json:"buchung_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	matched := 0
	for _, m := range req {
		tid, err := uuid.Parse(m.TransaktionID)
		if err != nil {
			continue
		}
		bid, err := uuid.Parse(m.BuchungID)
		if err != nil {
			continue
		}
		if err := h.eaRepo.SetBankMatch(r.Context(), tid, bid, eegID, 1.0, "bestaetigt"); err == nil {
			matched++
		}
	}
	jsonOK(w, map[string]int{"matched": matched})
}

func (h *EAHandler) IgnoriereBankTransaktion(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	tid, err := uuid.Parse(chi.URLParam(r, "transaktionID"))
	if err != nil {
		jsonError(w, "invalid ID", http.StatusBadRequest)
		return
	}
	if err := h.eaRepo.SetBankStatus(r.Context(), tid, eegID, "ignoriert"); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── UVA date helpers ──────────────────────────────────────────────────────────

func uvaPeriodeDates(jahr int, typ string, nr int) (time.Time, time.Time) {
	if typ == "QUARTAL" {
		startMonth := time.Month((nr-1)*3 + 1)
		endMonth := startMonth + 2
		von := time.Date(jahr, startMonth, 1, 0, 0, 0, 0, time.UTC)
		bis := time.Date(jahr, endMonth+1, 0, 23, 59, 59, 0, time.UTC) // last day of month
		return von, bis
	}
	// MONAT
	von := time.Date(jahr, time.Month(nr), 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(jahr, time.Month(nr)+1, 0, 23, 59, 59, 0, time.UTC)
	return von, bis
}

func uvaPeriodeLabelFromParts(jahr int, typ string, nr int) string {
	if typ == "QUARTAL" {
		quarters := []string{"", "Q1 (Jän–Mär)", "Q2 (Apr–Jun)", "Q3 (Jul–Sep)", "Q4 (Okt–Dez)"}
		if nr >= 1 && nr <= 4 {
			return fmt.Sprintf("%d / %s", jahr, quarters[nr])
		}
	}
	months := []string{"", "Jänner", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember"}
	if nr >= 1 && nr <= 12 {
		return fmt.Sprintf("%s %d", months[nr], jahr)
	}
	return fmt.Sprintf("%d/%d", jahr, nr)
}

// ── Jahreserklärungen ─────────────────────────────────────────────────────────

func (h *EAHandler) GetU1(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	jahr := parseYear(r.URL.Query().Get("jahr"), time.Now().Year())
	perioden, err := h.eaRepo.ListUVA(r.Context(), eegID, jahr)
	if err != nil {
		jsonError(w, "list uva failed", http.StatusInternalServerError)
		return
	}
	settings, _ := h.eaRepo.GetSettings(r.Context(), eegID)

	// Compute live annual Kennzahlen from buchungen for the full year.
	von := time.Date(jahr, 1, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(jahr, 12, 31, 23, 59, 59, 0, time.UTC)
	annual, err := h.eaRepo.CalcUVAKennzahlen(r.Context(), eegID, von, bis)
	if err != nil {
		jsonError(w, "calc kennzahlen failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "xml" {
		xmlData, err := EAU1FinanzOnlineXML(annual, settings, jahr)
		if err != nil {
			jsonError(w, "xml: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="u1_%d.xml"`, jahr))
		w.Write(xmlData)
		return
	}
	jsonOK(w, map[string]any{
		"jahr":     jahr,
		"perioden": perioden,
		"kz_000":   annual.KZ000,
		"kz_022":   annual.KZ022,
		"kz_029":   annual.KZ029,
		"kz_044":   annual.KZ044,
		"kz_056":   annual.KZ056,
		"kz_057":   annual.KZ057,
		"kz_060":   annual.KZ060,
		"kz_065":   annual.KZ065,
		"kz_066":   annual.KZ066,
		"kz_083":   annual.KZ083,
		"zahllast": annual.Zahllast,
	})
}

func (h *EAHandler) GetK1(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	jahr := parseYear(r.URL.Query().Get("jahr"), time.Now().Year())
	ja, err := h.eaRepo.Jahresabschluss(r.Context(), eegID, jahr)
	if err != nil {
		jsonError(w, "jahresabschluss failed", http.StatusInternalServerError)
		return
	}
	settings, _ := h.eaRepo.GetSettings(r.Context(), eegID)
	format := r.URL.Query().Get("format")
	if format == "xml" {
		xmlData, err := EAK1Summary(ja, settings)
		if err != nil {
			jsonError(w, "xml: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="kst_basis_%d.xml"`, jahr))
		w.Write(xmlData)
		return
	}
	jsonOK(w, ja)
}

func (h *EAHandler) GetK2(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	jahr := parseYear(r.URL.Query().Get("jahr"), time.Now().Year())
	ja, err := h.eaRepo.Jahresabschluss(r.Context(), eegID, jahr)
	if err != nil {
		jsonError(w, "jahresabschluss failed", http.StatusInternalServerError)
		return
	}
	settings, _ := h.eaRepo.GetSettings(r.Context(), eegID)
	format := r.URL.Query().Get("format")
	if format == "xml" {
		xmlData, err := EAK2Summary(ja, settings)
		if err != nil {
			jsonError(w, "xml: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="kst_k2_%d.xml"`, jahr))
		w.Write(xmlData)
		return
	}
	jsonOK(w, ja)
}

func (h *EAHandler) ExportBuchungenXLSX(w http.ResponseWriter, r *http.Request) {
	eegID, ok := h.parseEegID(r)
	if !ok {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	var filter repository.BuchungFilter
	filter.Geschaeftsjahr = parseYear(q.Get("jahr"), 0)
	buchungen, err := h.eaRepo.ListBuchungen(r.Context(), eegID, filter)
	if err != nil {
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Enrich with konto
	kontenCache := map[uuid.UUID]*domain.EAKonto{}
	for i := range buchungen {
		if _, ok := kontenCache[buchungen[i].KontoID]; !ok {
			k, err := h.eaRepo.GetKonto(r.Context(), buchungen[i].KontoID)
			if err == nil {
				kontenCache[buchungen[i].KontoID] = k
			}
		}
		buchungen[i].Konto = kontenCache[buchungen[i].KontoID]
	}
	data, err := EABuchungenXLSX(buchungen)
	if err != nil {
		jsonError(w, "xlsx: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jahr := parseYear(q.Get("jahr"), time.Now().Year())
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="buchungsjournal_%d.xlsx"`, jahr))
	w.Write(data)
}
