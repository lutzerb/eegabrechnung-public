package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/netzbetreiber"
)

// nbReportedEntry is the latest Netzbetreiber-reported state for one Zählpunkt, folded
// from all stored SENDEN_ECP (EC_PODLIST / Zählpunktliste) messages for an EEG — the most
// recently received message wins per Zählpunkt (see buildNBReportedMap).
type nbReportedEntry struct {
	Factor          float64
	EnergyDirection string // CONSUMPTION | GENERATION, "" if unparseable
	ConsentID       string
	PlantCategory   string // mapped to our generation_type vocabulary (PV/Windkraft/Wasserkraft); "" if absent/unmapped
	// RegistriertSeit is DateActivate — the stable original registration/activation date
	// for this Zählpunkt's EC participation. Unlike DateFrom, it stays constant across all
	// MPTimeData periods of a Zählpunkt even as the factor/direction/etc. changes over time
	// (verified against real ECMPList payloads with several periods) — DateFrom only marks
	// the start of whichever period is currently selected as "current", so it drifts every
	// time any attribute changes and must never be used as "registered since". HasRealDate
	// is false when the NB reported no date at all (many SENDEN_ECP entries don't) — only
	// then is RegistriertSeit a meaningless zero value that must not be compared/displayed.
	RegistriertSeit time.Time
	HasRealDate     bool
	// FactorValidFrom is DateFrom (falling back to DateActivate, then the message's own
	// CreatedAt) — the start of the CURRENTLY selected MPTimeData period, i.e. the date
	// since which the reported Factor/EnergyDirection/PlantCategory have applied. This is
	// the correct valid_from for writing a reconciled Teilnahmefaktor — NOT RegistriertSeit,
	// which would backdate the change to the Zählpunkt's original registration.
	FactorValidFrom time.Time
	AbgemeldetAm    *time.Time // nil = NB reports this Zählpunkt as still open-ended
	ShareType       string     // = ECMPListResult.ECType, used as eeg_meter_participations.share_type
	SourceMessageID uuid.UUID
}

// plantCategoryToGenerationType maps the Netzbetreiber's PlantCategory values (as observed
// in real ECMPList payloads) onto our generation_type vocabulary. Unrecognized values are
// left unmapped (empty string) rather than guessed.
var plantCategoryToGenerationType = map[string]string{
	"SONNE":  "PV",
	"WIND":   "Windkraft",
	"WASSER": "Wasserkraft",
}

// buildNBReportedMap folds all stored SENDEN_ECP messages for eegID into
// map[Zählpunkt]→nbReportedEntry. ListECPListPayloadsByEEG returns messages oldest-first,
// so a later message's entry always overwrites an earlier one for the same Zählpunkt —
// this naturally handles BEG's multiple NB-scoped requests and repeated EC_PODLIST runs
// over time without needing to match against eda_processes/conversation IDs. lastReceivedPerNB
// maps each responding Netzbetreiber's Marktpartner-ID to its most recent SENDEN_ECP —
// a BEG requests one Zählpunktliste per distinct Netzbetreiber, and in practice they don't
// all answer equally reliably (verified: one NB never responded across 4 separate requests
// over ~2 months while its siblings answered every time), so a single overall "last received"
// would hide that one Netzbetreiber's Zählpunkte are being compared against stale — or no —
// data. Surfaced per-NB in the response instead.
func (h *EDAHandler) buildNBReportedMap(ctx context.Context, eegID uuid.UUID) (map[string]nbReportedEntry, map[string]time.Time, error) {
	payloads, err := h.edaMsgRepo.ListECPListPayloadsByEEG(ctx, eegID)
	if err != nil {
		return nil, nil, fmt.Errorf("list ECP list payloads: %w", err)
	}
	lastReceivedPerNB := make(map[string]time.Time)
	for _, p := range payloads {
		if p.FromAddress != "" {
			lastReceivedPerNB[p.FromAddress] = p.CreatedAt // ascending order — later overwrites earlier
		}
	}
	result := make(map[string]nbReportedEntry)
	for _, p := range payloads {
		parsed, err := edaxml.ParseECMPList(p.XMLPayload)
		if err != nil {
			continue // skip unparseable historical messages, don't fail the whole request
		}
		for _, e := range parsed.Entries {
			factor, ferr := strconv.ParseFloat(e.ECPartFact, 64)
			if ferr != nil {
				continue // skip entries with an unparseable factor
			}
			registeredSince := e.DateActivate
			if registeredSince.IsZero() {
				registeredSince = e.DateFrom // better than nothing if DateActivate is absent
			}
			hasRealDate := !registeredSince.IsZero()

			factorValidFrom := e.DateFrom
			if factorValidFrom.IsZero() {
				factorValidFrom = e.DateActivate
			}
			if factorValidFrom.IsZero() {
				factorValidFrom = p.CreatedAt // default valid_from for the factor write only
			}

			var abgemeldetAm *time.Time
			if !e.DateTo.IsZero() {
				d := e.DateTo
				abgemeldetAm = &d
			}
			result[e.MeteringPoint] = nbReportedEntry{
				Factor:          factor,
				EnergyDirection: e.EnergyDirection,
				ConsentID:       e.ConsentID,
				PlantCategory:   plantCategoryToGenerationType[e.PlantCategory],
				RegistriertSeit: registeredSince,
				HasRealDate:     hasRealDate,
				FactorValidFrom: factorValidFrom,
				AbgemeldetAm:    abgemeldetAm,
				ShareType:       parsed.ECType,
				SourceMessageID: p.ID,
			}
		}
	}
	return result, lastReceivedPerNB, nil
}

func formatFactor(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

type podListFieldDiff struct {
	Field      string `json:"field"`      // participation_factor | energy_direction | consent_id | generation_type | registriert_seit | abgemeldet_am
	Label      string `json:"label"`      // display text, e.g. "Teilnahmefaktor"
	OurValue   string `json:"our_value"`
	NBValue    string `json:"nb_value"`
	Applicable bool   `json:"applicable"` // true = applied by POST .../apply; false = informational, manual review only
}

type podListReconcileRow struct {
	MeterPointID    uuid.UUID          `json:"meter_point_id"`
	Zaehlpunkt      string             `json:"zaehlpunkt"`
	MemberName      string             `json:"member_name"`
	Diffs           []podListFieldDiff `json:"diffs"`
	EffectiveFrom   string             `json:"effective_from"` // YYYY-MM-DD, the NB's own effective date for this entry
	SourceMessageID uuid.UUID          `json:"source_message_id"`
}

type podListUnknownZP struct {
	Zaehlpunkt      string    `json:"zaehlpunkt"`
	ReportedFactor  float64   `json:"reported_factor"`
	SourceMessageID uuid.UUID `json:"source_message_id"`
}

type podListNoNBData struct {
	MeterPointID uuid.UUID `json:"meter_point_id"`
	Zaehlpunkt   string    `json:"zaehlpunkt"`
	MemberName   string    `json:"member_name"`
}

// podListNBFreshness reports when the Zählpunktliste last arrived from one Netzbetreiber
// currently serving this EEG's active meter points. LastReceivedAt is nil if that
// Netzbetreiber has never answered a SENDEN_ECP for this EEG at all.
type podListNBFreshness struct {
	NetzbetreiberID   string     `json:"netzbetreiber_id"`
	NetzbetreiberName string     `json:"netzbetreiber_name"`
	LastReceivedAt    *time.Time `json:"last_received_at"`
}

type podListReconcileResponse struct {
	Mismatches         []podListReconcileRow `json:"mismatches"`
	UnknownZaehlpunkte []podListUnknownZP    `json:"unknown_zaehlpunkte"`
	NoNBData           []podListNoNBData     `json:"no_nb_data"`
	HasAnyData         bool                  `json:"has_any_data"` // false = no SENDEN_ECP message stored yet for this EEG
	// PodlistFreshness lists every Netzbetreiber currently serving an active meter point of
	// this EEG with its own last-received timestamp — one entry for EEG/GEA (single NB), one
	// per distinct NB for a BEG. Lets the admin see when ANY of them is stale or has never
	// answered, which a single overall timestamp would hide.
	PodlistFreshness []podListNBFreshness `json:"podlist_freshness"`
}

// PODListReconcile godoc
//
//	@Summary		Teilnahmefaktor-/Zählpunktdaten-Abgleich (EC_PODLIST)
//	@Description	Compares our stored meter point data (Teilnahmefaktor, Energierichtung, Consent-ID, Anlagentyp, Registrierungsdaten) against the latest Netzbetreiber-reported state from all stored EC_PODLIST (Zählpunktliste) responses for this EEG, and returns proposed changes for review.
//	@Tags			EDA
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Success		200		{object}	podListReconcileResponse
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/podlist/reconcile [get]
func (h *EDAHandler) PODListReconcile(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	ctx := r.Context()

	mps, err := h.mpRepo.ListByEeg(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to list meter points", http.StatusInternalServerError)
		return
	}
	members, err := h.memberRepo.ListByEeg(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to list members", http.StatusInternalServerError)
		return
	}
	memberNames := make(map[uuid.UUID]string, len(members))
	for _, m := range members {
		name := m.Name1
		if m.Name2 != "" {
			name += " " + m.Name2
		}
		memberNames[m.ID] = name
	}

	var activeIDs []uuid.UUID
	knownZP := make(map[string]bool, len(mps))
	for _, mp := range mps {
		knownZP[mp.Zaehlpunkt] = true
		if mp.AbgemeldetAm == nil {
			activeIDs = append(activeIDs, mp.ID)
		}
	}

	factors, err := h.participRepo.GetCurrentFactorsByEEG(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to load participation factors", http.StatusInternalServerError)
		return
	}
	edaStatus, err := h.mpRepo.GetEDAStatusByMeterPoints(ctx, activeIDs)
	if err != nil {
		jsonError(w, "failed to load EDA status", http.StatusInternalServerError)
		return
	}
	nbMap, lastReceivedPerNB, err := h.buildNBReportedMap(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to load Netzbetreiber data", http.StatusInternalServerError)
		return
	}

	freshness := make([]podListNBFreshness, 0, 1)
	for _, nb := range netzbetreiber.ActiveFromMeterPoints(mps) {
		entry := podListNBFreshness{NetzbetreiberID: nb.ID, NetzbetreiberName: nb.Name}
		if t, ok := lastReceivedPerNB[nb.ID]; ok {
			entry.LastReceivedAt = &t
		}
		freshness = append(freshness, entry)
	}

	resp := podListReconcileResponse{
		HasAnyData:         len(nbMap) > 0,
		PodlistFreshness:   freshness,
		Mismatches:         []podListReconcileRow{},
		UnknownZaehlpunkte: []podListUnknownZP{},
		NoNBData:           []podListNoNBData{},
	}

	for _, mp := range mps {
		if mp.AbgemeldetAm != nil {
			continue // only active meter points are compared
		}
		nb, found := nbMap[mp.Zaehlpunkt]
		if !found {
			resp.NoNBData = append(resp.NoNBData, podListNoNBData{
				MeterPointID: mp.ID,
				Zaehlpunkt:   mp.Zaehlpunkt,
				MemberName:   memberNames[mp.MemberID],
			})
			continue
		}

		var diffs []podListFieldDiff

		ours := currentFactor(mp.ID, factors, edaStatus)
		if math.Abs(ours-nb.Factor) > 0.01 {
			diffs = append(diffs, podListFieldDiff{
				Field: "participation_factor", Label: "Teilnahmefaktor",
				OurValue: formatFactor(ours), NBValue: formatFactor(nb.Factor), Applicable: true,
			})
		}

		if nb.EnergyDirection != "" && nb.EnergyDirection != mp.Energierichtung {
			diffs = append(diffs, podListFieldDiff{
				Field: "energy_direction", Label: "Energierichtung",
				OurValue: mp.Energierichtung, NBValue: nb.EnergyDirection, Applicable: true,
			})
		}

		if nb.ConsentID != "" && nb.ConsentID != mp.ConsentID {
			diffs = append(diffs, podListFieldDiff{
				Field: "consent_id", Label: "Consent-ID",
				OurValue: mp.ConsentID, NBValue: nb.ConsentID, Applicable: true,
			})
		}

		if nb.EnergyDirection == "GENERATION" && nb.PlantCategory != "" {
			ourGenType := ""
			if mp.GenerationType != nil {
				ourGenType = *mp.GenerationType
			}
			if nb.PlantCategory != ourGenType {
				diffs = append(diffs, podListFieldDiff{
					Field: "generation_type", Label: "Anlagentyp",
					OurValue: ourGenType, NBValue: nb.PlantCategory, Applicable: true,
				})
			}
		}

		if nb.HasRealDate {
			ourReg := ""
			if mp.RegistriertSeit != nil {
				ourReg = mp.RegistriertSeit.Format("2006-01-02")
			}
			nbReg := nb.RegistriertSeit.Format("2006-01-02")
			if ourReg != nbReg {
				diffs = append(diffs, podListFieldDiff{
					Field: "registriert_seit", Label: "Registriert seit (laut Netzbetreiber)",
					OurValue: ourReg, NBValue: nbReg, Applicable: false,
				})
			}
		}

		if nb.AbgemeldetAm != nil {
			// mp.AbgemeldetAm is always nil here (filtered above) — this flags the case
			// where the Netzbetreiber already has an end date for a Zählpunkt we still
			// treat as active.
			diffs = append(diffs, podListFieldDiff{
				Field: "abgemeldet_am", Label: "Abgemeldet am (laut Netzbetreiber)",
				OurValue: "", NBValue: nb.AbgemeldetAm.Format("2006-01-02"), Applicable: false,
			})
		}

		if len(diffs) == 0 {
			continue
		}
		resp.Mismatches = append(resp.Mismatches, podListReconcileRow{
			MeterPointID:    mp.ID,
			Zaehlpunkt:      mp.Zaehlpunkt,
			MemberName:      memberNames[mp.MemberID],
			Diffs:           diffs,
			EffectiveFrom:   nb.FactorValidFrom.Format("2006-01-02"),
			SourceMessageID: nb.SourceMessageID,
		})
	}

	for zp, nb := range nbMap {
		if !knownZP[zp] {
			resp.UnknownZaehlpunkte = append(resp.UnknownZaehlpunkte, podListUnknownZP{
				Zaehlpunkt: zp, ReportedFactor: nb.Factor, SourceMessageID: nb.SourceMessageID,
			})
		}
	}

	jsonOK(w, resp)
}

type podListApplyRequest struct {
	MeterPointIDs []uuid.UUID `json:"meter_point_ids"`
}

type podListApplyResultItem struct {
	MeterPointID uuid.UUID `json:"meter_point_id"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// PODListApply godoc
//
//	@Summary		Zählpunktdaten-Abgleich übernehmen
//	@Description	Applies the applicable (non-informational) diffs found by GET .../podlist/reconcile for the given meter points. Only meter_point_ids are accepted from the client — the Netzbetreiber-reported values are always re-derived server-side from stored EC_PODLIST messages, never trusted from the request body.
//	@Tags			EDA
//	@Accept			json
//	@Produce		json
//	@Param			eegID	path		string					true	"EEG ID (UUID)"
//	@Param			body	body		podListApplyRequest		true	"Meter point IDs to reconcile"
//	@Success		200		{array}		podListApplyResultItem
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/eda/podlist/apply [post]
func (h *EDAHandler) PODListApply(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	ctx := r.Context()

	var req podListApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.MeterPointIDs) == 0 {
		jsonError(w, "meter_point_ids must not be empty", http.StatusBadRequest)
		return
	}

	mps, err := h.mpRepo.ListByEeg(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to list meter points", http.StatusInternalServerError)
		return
	}
	mpByID := make(map[uuid.UUID]domain.MeterPoint, len(mps))
	var allMPIDs []uuid.UUID
	for _, mp := range mps {
		mpByID[mp.ID] = mp
		allMPIDs = append(allMPIDs, mp.ID)
	}

	factors, err := h.participRepo.GetCurrentFactorsByEEG(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to load participation factors", http.StatusInternalServerError)
		return
	}
	edaStatus, err := h.mpRepo.GetEDAStatusByMeterPoints(ctx, allMPIDs)
	if err != nil {
		jsonError(w, "failed to load EDA status", http.StatusInternalServerError)
		return
	}
	nbMap, _, err := h.buildNBReportedMap(ctx, eeg.ID)
	if err != nil {
		jsonError(w, "failed to load Netzbetreiber data", http.StatusInternalServerError)
		return
	}

	results := make([]podListApplyResultItem, 0, len(req.MeterPointIDs))
	for _, mpID := range req.MeterPointIDs {
		// mpByID/nbMap are re-derived from the DB/stored messages above — never from the
		// request body — so a client cannot push an arbitrary factor or value through
		// this endpoint, and an ID outside this (org-scoped) EEG simply isn't found.
		mp, found := mpByID[mpID]
		if !found {
			results = append(results, podListApplyResultItem{MeterPointID: mpID, Success: false, Error: "meter point not found in this EEG"})
			continue
		}
		nb, found := nbMap[mp.Zaehlpunkt]
		if !found {
			results = append(results, podListApplyResultItem{MeterPointID: mpID, Success: false, Error: "no Netzbetreiber data for this Zählpunkt"})
			continue
		}

		var errs []string

		ours := currentFactor(mp.ID, factors, edaStatus)
		if math.Abs(ours-nb.Factor) > 0.01 {
			if err := h.participRepo.UpsertReconciledFactor(ctx, eeg.ID, mp.ID, nb.Factor, nb.ShareType, nb.FactorValidFrom); err != nil {
				errs = append(errs, fmt.Sprintf("Teilnahmefaktor: %v", err))
			}
		}

		mpChanged := false
		if nb.EnergyDirection != "" && nb.EnergyDirection != mp.Energierichtung {
			mp.Energierichtung = nb.EnergyDirection
			mpChanged = true
		}
		if nb.EnergyDirection == "GENERATION" && nb.PlantCategory != "" {
			ourGenType := ""
			if mp.GenerationType != nil {
				ourGenType = *mp.GenerationType
			}
			if nb.PlantCategory != ourGenType {
				gt := nb.PlantCategory
				mp.GenerationType = &gt
				mpChanged = true
			}
		}
		if mpChanged {
			if err := h.mpRepo.Update(ctx, &mp); err != nil {
				errs = append(errs, fmt.Sprintf("Zählpunkt-Update: %v", err))
			}
		}

		if nb.ConsentID != "" && nb.ConsentID != mp.ConsentID {
			if err := h.mpRepo.UpdateConsentID(ctx, mp.ID, nb.ConsentID); err != nil {
				errs = append(errs, fmt.Sprintf("Consent-ID: %v", err))
			}
		}

		if len(errs) > 0 {
			results = append(results, podListApplyResultItem{MeterPointID: mpID, Success: false, Error: strings.Join(errs, "; ")})
		} else {
			results = append(results, podListApplyResultItem{MeterPointID: mpID, Success: true})
		}
	}

	jsonOK(w, results)
}
