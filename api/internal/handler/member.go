package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	_ "time/tzdata" // embed IANA timezone database (required in Alpine containers)

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/invoice"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

type MemberHandler struct {
	memberRepo     *repository.MemberRepository
	meterPointRepo *repository.MeterPointRepository
	eegRepo        *repository.EEGRepository
	edaProcRepo    *repository.EDAProcessRepository
	jobRepo        *repository.JobRepository
	participRepo   *repository.ParticipationRepository
}

func NewMemberHandler(
	memberRepo *repository.MemberRepository,
	meterPointRepo *repository.MeterPointRepository,
	eegRepo *repository.EEGRepository,
	edaProcRepo *repository.EDAProcessRepository,
	jobRepo *repository.JobRepository,
	participRepo *repository.ParticipationRepository,
) *MemberHandler {
	return &MemberHandler{
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		eegRepo:        eegRepo,
		edaProcRepo:    edaProcRepo,
		jobRepo:        jobRepo,
		participRepo:   participRepo,
	}
}

type memberRequest struct {
	MitgliedsNr  string   `json:"mitglieds_nr"`
	Name1        string   `json:"name1"`
	Name2        string   `json:"name2"`
	Email        string   `json:"email"`
	IBAN         string   `json:"iban"`
	Strasse      string   `json:"strasse"`
	Plz          string   `json:"plz"`
	Ort          string   `json:"ort"`
	// BusinessRole: "privat", "kleinunternehmer", "landwirt_pauschaliert",
	// "unternehmen", "gemeinde_bga", "gemeinde_hoheitlich"
	BusinessRole string   `json:"business_role"`
	// UidNummer: VAT ID of the member. When set → Reverse Charge on credit notes.
	UidNummer    string   `json:"uid_nummer"`
	// UseVat / VatPct: reserved for manual override on credit notes.
	// NOT applied to consumer invoices.
	UseVat       *bool    `json:"use_vat"`
	VatPct       *float64 `json:"vat_pct"`
	// Status: ACTIVE | REGISTERED | NEW | INACTIVE
	Status         string `json:"status"`
	// BeitrittsDatum / AustrittsDatum: YYYY-MM-DD date strings
	BeitrittsDatum string `json:"beitritts_datum"`
	AustrittsDatum string `json:"austritts_datum"`
}

// parseDateField parses an optional YYYY-MM-DD date string into *time.Time.
// Returns nil if the string is empty, and nil + error if parsing fails.
func parseDateField(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateMember godoc
// @Summary     Create member
// @Description Creates a new member in the given EEG. Field name1 is required. business_role defaults to "privat". MitgliedsNr is auto-generated if not provided.
// @Tags        Mitglieder
// @Accept      json
// @Produce     json
// @Param       eegID   path      string         true  "EEG UUID"
// @Param       member  body      memberRequest  true  "Member data (name1 required)"
// @Success     201  {object}  domain.Member  "Created member"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/members [post]
// CreateMember handles POST /eegs/{eegID}/members
func (h *MemberHandler) CreateMember(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	var req memberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name1 == "" {
		jsonError(w, "name1 is required", http.StatusBadRequest)
		return
	}
	if req.BusinessRole == "" {
		req.BusinessRole = "privat"
	}

	ctx := r.Context()

	// Auto-generate member number if not provided
	if req.MitgliedsNr == "" {
		nr, err := h.memberRepo.NextMemberNumber(ctx, eegID)
		if err != nil {
			jsonError(w, "failed to generate member number", http.StatusInternalServerError)
			return
		}
		req.MitgliedsNr = nr
	}

	beitrittsDatum, err := parseDateField(req.BeitrittsDatum)
	if err != nil {
		jsonError(w, "invalid beitritts_datum: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	austrittsDatum, err := parseDateField(req.AustrittsDatum)
	if err != nil {
		jsonError(w, "invalid austritts_datum: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	status := req.Status
	if status == "" {
		status = "ACTIVE"
	}
	m := &domain.Member{
		EegID:          eegID,
		MitgliedsNr:    req.MitgliedsNr,
		Name1:          req.Name1,
		Name2:          req.Name2,
		Email:          req.Email,
		IBAN:           req.IBAN,
		Strasse:        req.Strasse,
		Plz:            req.Plz,
		Ort:            req.Ort,
		BusinessRole:   req.BusinessRole,
		UidNummer:      req.UidNummer,
		UseVat:         req.UseVat,
		VatPct:         req.VatPct,
		Status:         status,
		BeitrittsDatum: beitrittsDatum,
		AustrittsDatum: austrittsDatum,
	}
	if err := h.memberRepo.Create(ctx, m); err != nil {
		jsonError(w, "failed to create member", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, m)
}

// GetMember godoc
// @Summary     Get member
// @Description Returns a single member with all their assigned meter points.
// @Tags        Mitglieder
// @Produce     json
// @Param       eegID     path      string  true  "EEG UUID"
// @Param       memberID  path      string  true  "Member UUID"
// @Success     200  {object}  memberWithMeterPoints  "Member with meter points"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/members/{memberID} [get]
// GetMember handles GET /eegs/{eegID}/members/{memberID}
func (h *MemberHandler) GetMember(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	m, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}
	mps, err := h.meterPointRepo.ListByMember(r.Context(), memberID)
	if err != nil {
		mps = nil
	}
	mpIDs := make([]uuid.UUID, 0, len(mps))
	for _, mp := range mps {
		mpIDs = append(mpIDs, mp.ID)
	}
	edaStatus, err := h.meterPointRepo.GetEDAStatusByMeterPoints(r.Context(), mpIDs)
	if err != nil {
		edaStatus = map[uuid.UUID]repository.MeterPointEDAStatus{}
	}
	factors, err := h.participRepo.GetCurrentFactorsByEEG(r.Context(), m.EegID)
	if err != nil {
		factors = map[uuid.UUID]float64{}
	}
	jsonOK(w, toMemberWithMPs(*m, mps, factors, edaStatus))
}

// UpdateMember godoc
// @Summary     Update member
// @Description Updates an existing member's details. Provided fields overwrite existing values; omitting a field typically retains the current value (except nullable fields which are cleared).
// @Tags        Mitglieder
// @Accept      json
// @Produce     json
// @Param       eegID     path      string         true  "EEG UUID"
// @Param       memberID  path      string         true  "Member UUID"
// @Param       member    body      memberRequest  true  "Member update data"
// @Success     200  {object}  domain.Member  "Updated member"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     404  {object}  map[string]string  "Not found"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/members/{memberID} [put]
// UpdateMember handles PUT /eegs/{eegID}/members/{memberID}
func (h *MemberHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	existing, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}

	var req memberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name1 != "" {
		existing.Name1 = req.Name1
	}
	existing.Name2 = req.Name2
	existing.Email = req.Email
	existing.IBAN = req.IBAN
	existing.Strasse = req.Strasse
	existing.Plz = req.Plz
	existing.Ort = req.Ort
	if req.BusinessRole != "" {
		existing.BusinessRole = req.BusinessRole
	}
	if req.MitgliedsNr != "" {
		existing.MitgliedsNr = req.MitgliedsNr
	}
	existing.UidNummer = req.UidNummer
	// UseVat and VatPct: always taken from request (nil = reset to inherit)
	existing.UseVat = req.UseVat
	existing.VatPct = req.VatPct
	if req.Status != "" {
		existing.Status = req.Status
	}

	// Parse optional date fields (empty string = clear the date)
	beitrittsDatum, err := parseDateField(req.BeitrittsDatum)
	if err != nil {
		jsonError(w, "invalid beitritts_datum: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	austrittsDatum, err := parseDateField(req.AustrittsDatum)
	if err != nil {
		jsonError(w, "invalid austritts_datum: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	existing.BeitrittsDatum = beitrittsDatum
	existing.AustrittsDatum = austrittsDatum

	if err := h.memberRepo.Update(r.Context(), existing); err != nil {
		jsonError(w, "failed to update member", http.StatusInternalServerError)
		return
	}
	jsonOK(w, existing)
}

// DeleteMember godoc
// @Summary     Delete member
// @Description Permanently removes a member and their data from the EEG. This action is irreversible.
// @Tags        Mitglieder
// @Param       eegID     path  string  true  "EEG UUID"
// @Param       memberID  path  string  true  "Member UUID"
// @Success     204  "No content"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Unauthorized"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Security    BearerAuth
// @Router      /eegs/{eegID}/members/{memberID} [delete]
// DeleteMember handles DELETE /eegs/{eegID}/members/{memberID}
func (h *MemberHandler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	if err := h.memberRepo.Delete(r.Context(), memberID); err != nil {
		jsonError(w, "failed to delete member", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// austrittRequest is the body for POST /members/{memberID}/austritt.
type austrittRequest struct {
	AustrittDatum string `json:"austritt_datum"` // YYYY-MM-DD, must be >= tomorrow (Vienna time)
}

// Austritt godoc
//
//	@Summary		Deregister member (Mitglieder-Austritt)
//	@Description	Sets the member to INACTIVE, stores the Austrittsdatum, and triggers CM_REV_SP for all active meter points (abgemeldet_am IS NULL).
//	@Tags			Mitglieder
//	@Accept			json
//	@Produce		json
//	@Param			eegID     path  string          true  "EEG UUID"
//	@Param			memberID  path  string          true  "Member UUID"
//	@Param			body      body  austrittRequest true  "Austritt date"
//	@Success		200  {object}  map[string]int  "abmeldungen_erstellt"
//	@Failure		400  {object}  map[string]string
//	@Failure		401  {object}  map[string]string
//	@Failure		404  {object}  map[string]string
//	@Failure		500  {object}  map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/members/{memberID}/austritt [post]
func (h *MemberHandler) Austritt(w http.ResponseWriter, r *http.Request) {
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
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	var req austrittRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.AustrittDatum == "" {
		jsonError(w, "austritt_datum is required", http.StatusBadRequest)
		return
	}

	// Validate: must be >= tomorrow (Vienna time) and <= today + 30 Austrian working days (EDA CM_REV_SP limit).
	viennaLoc, _ := time.LoadLocation("Europe/Vienna")
	nowVienna := time.Now().In(viennaLoc)
	todayVienna := time.Date(nowVienna.Year(), nowVienna.Month(), nowVienna.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := todayVienna.AddDate(0, 0, 1)
	maxDate := addAustrianWorkingDays(todayVienna, 30)

	austrittDate, err := time.Parse("2006-01-02", req.AustrittDatum)
	if err != nil {
		jsonError(w, "austritt_datum must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if austrittDate.Before(tomorrow) {
		jsonError(w, "austritt_datum muss mindestens morgen sein", http.StatusBadRequest)
		return
	}
	if austrittDate.After(maxDate) {
		jsonError(w, fmt.Sprintf("austritt_datum darf höchstens 30 Arbeitstage in der Zukunft liegen (max %s)", maxDate.Format("2006-01-02")), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Load the member
	member, err := h.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}

	// Load the EEG (needed for EDA credentials + IDs)
	eeg, err := h.eegRepo.GetByID(ctx, eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	// Collect all meter points of this member that are not yet deregistered
	allMPs, err := h.meterPointRepo.ListByMember(ctx, memberID)
	if err != nil {
		jsonError(w, "failed to load meter points", http.StatusInternalServerError)
		return
	}
	var activeMPs []domain.MeterPoint
	for _, mp := range allMPs {
		if mp.AbgemeldetAm == nil {
			activeMPs = append(activeMPs, mp)
		}
	}

	abmeldungenErstellt := 0

	// Only send EDA if the EEG has credentials configured
	if eeg.EdaMarktpartnerID != "" && eeg.EdaNetzbetreiberID != "" && !eeg.IsDemo {
		for _, mp := range activeMPs {
			// Idempotency: skip if a CM_REV_SP process for this Zählpunkt is already pending/sent
			alreadyPending, checkErr := h.edaProcRepo.HasPendingABM(ctx, eegID, mp.Zaehlpunkt)
			if checkErr != nil {
				jsonError(w, fmt.Sprintf("failed to check pending Widerruf for %s", mp.Zaehlpunkt), http.StatusInternalServerError)
				return
			}
			if alreadyPending {
				continue
			}

			// consent_id is required for CM_REV_SP — fail early if not stored yet.
			if mp.ConsentID == "" {
				jsonError(w, fmt.Sprintf("Zählpunkt %s hat keine gespeicherte Consent-ID — bitte manuell per EDA-Widerruf abmelden", mp.Zaehlpunkt), http.StatusUnprocessableEntity)
				return
			}

			// Derive Netzbetreiber-ID from Zählpunkt prefix.
			netzbetreiberTo := eeg.EdaNetzbetreiberID
			if len(mp.Zaehlpunkt) >= 8 {
				netzbetreiberTo = mp.Zaehlpunkt[:8]
			}

			convID := uuid.NewString()
			xmlBody, xmlErr := edaxml.BuildCMRevoke(edaxml.CMRevokeParams{
				From:           eeg.EdaMarktpartnerID,
				To:             netzbetreiberTo,
				MessageID:      uuid.NewString(),
				ConversationID: convID,
				MeteringPoint:  mp.Zaehlpunkt,
				ConsentID:      mp.ConsentID,
				ConsentEnd:     austrittDate,
			})
			if xmlErr != nil {
				jsonError(w, fmt.Sprintf("build XML for %s: %v", mp.Zaehlpunkt, xmlErr), http.StatusInternalServerError)
				return
			}

			mpID := mp.ID
			proc := &domain.EDAProcess{
				EegID:          eegID,
				MeterPointID:   &mpID,
				ProcessType:    "CM_REV_SP",
				Status:         "pending",
				ConversationID: convID,
				Zaehlpunkt:     mp.Zaehlpunkt,
				InitiatedAt:    time.Now(),
			}
			proc.ValidFrom = &austrittDate

			if err := h.edaProcRepo.Create(ctx, proc); err != nil {
				jsonError(w, fmt.Sprintf("failed to create EDA process for %s", mp.Zaehlpunkt), http.StatusInternalServerError)
				return
			}
			if err := h.jobRepo.EnqueueEDA(ctx, "CM_REV_SP", eeg.EdaMarktpartnerID, netzbetreiberTo,
				eeg.GemeinschaftID, convID, xmlBody, proc.ID, eegID); err != nil {
				jsonError(w, fmt.Sprintf("failed to queue EDA job for %s", mp.Zaehlpunkt), http.StatusInternalServerError)
				return
			}
			abmeldungenErstellt++
		}
	}

	// Update member: set INACTIVE + austritt_datum
	member.Status = "INACTIVE"
	member.AustrittsDatum = &austrittDate
	if err := h.memberRepo.Update(ctx, member); err != nil {
		jsonError(w, "failed to update member status", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]int{"abmeldungen_erstellt": abmeldungenErstellt})
}

// DownloadSepaMandat godoc
//
//	@Summary		Download SEPA mandate PDF for a member
//	@Description	Generates a SEPA mandate PDF containing the member's name, address, IBAN, mandate reference, creditor ID, acceptance timestamp, IP address, and the contract text accepted during onboarding.
//	@Tags			Mitglieder
//	@Produce		application/pdf
//	@Param			eegID		path	string	true	"EEG UUID"
//	@Param			memberID	path	string	true	"Member UUID"
//	@Success		200	{file}	application/pdf	"SEPA mandate PDF attachment"
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/members/{memberID}/sepa-mandat [get]
func (h *MemberHandler) DownloadSepaMandat(w http.ResponseWriter, r *http.Request) {
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
	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}
	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}

	pdfBytes, err := invoice.GenerateSepaMandatPDF(eeg, member)
	if err != nil {
		jsonError(w, "PDF-Generierung fehlgeschlagen: "+err.Error(), http.StatusInternalServerError)
		return
	}

	name := fmt.Sprintf("sepamandat_%s.pdf", memberID.String()[:8])
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Write(pdfBytes)
}
