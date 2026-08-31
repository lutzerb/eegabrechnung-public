package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// ReferralHandler handles the admin-facing "Mitglieder werben Mitglieder" bonus
// workflow: listing successful-but-unrewarded referrals and manually granting
// (or undoing) a bonus for one. See Migration 114.
type ReferralHandler struct {
	bonusRepo  *repository.ReferralBonusRepository
	memberRepo *repository.MemberRepository
	eegRepo    *repository.EEGRepository
}

func NewReferralHandler(bonusRepo *repository.ReferralBonusRepository, memberRepo *repository.MemberRepository, eegRepo *repository.EEGRepository) *ReferralHandler {
	return &ReferralHandler{bonusRepo: bonusRepo, memberRepo: memberRepo, eegRepo: eegRepo}
}

// ListEligible handles GET /eegs/{eegID}/referrals/eligible
// Successful referrals (converted member with referred_by_member_id set) that
// don't have a bonus granted yet.
func (h *ReferralHandler) ListEligible(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	list, err := h.bonusRepo.ListEligible(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to list eligible referrals", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []repository.EligibleReferral{}
	}
	jsonOK(w, list)
}

// List handles GET /eegs/{eegID}/referrals
// All granted bonuses (pending/applied/cancelled), newest first — audit trail.
func (h *ReferralHandler) List(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	bonuses, err := h.bonusRepo.List(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to list referral bonuses", http.StatusInternalServerError)
		return
	}
	if bonuses == nil {
		bonuses = []domain.MemberReferralBonus{}
	}
	jsonOK(w, bonuses)
}

// Grant handles POST /eegs/{eegID}/referrals/{referredMemberID}/grant-bonus
// Body: {"amount_eur": 5.0} — optional; defaults to eeg.ReferralBonusEur.
func (h *ReferralHandler) Grant(w http.ResponseWriter, r *http.Request) {
	claims, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	referredMemberID, err := uuid.Parse(chi.URLParam(r, "referredMemberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}

	var body struct {
		AmountEur *float64 `json:"amount_eur"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // body is optional

	amount := eeg.ReferralBonusEur
	if body.AmountEur != nil {
		if *body.AmountEur <= 0 {
			jsonError(w, "amount_eur must be positive", http.StatusBadRequest)
			return
		}
		amount = *body.AmountEur
	}

	referred, err := h.memberRepo.GetByID(r.Context(), referredMemberID)
	if err != nil || referred.EegID != eeg.ID {
		jsonError(w, "member not found", http.StatusNotFound)
		return
	}
	if referred.ReferredByMemberID == nil {
		jsonError(w, "member was not referred by another member", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	bonus, err := h.bonusRepo.Grant(r.Context(), eeg.ID, *referred.ReferredByMemberID, referredMemberID, amount, userID)
	if err != nil {
		jsonError(w, "eine Prämie für diese Werbung wurde bereits vergeben", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, bonus)
}

// CancelPending handles DELETE /eegs/{eegID}/referrals/{referredMemberID}/grant-bonus
// Undoes a grant, only while it's still pending (not yet applied to an invoice).
func (h *ReferralHandler) CancelPending(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	referredMemberID, err := uuid.Parse(chi.URLParam(r, "referredMemberID"))
	if err != nil {
		jsonError(w, "invalid member ID", http.StatusBadRequest)
		return
	}
	if err := h.bonusRepo.CancelPending(r.Context(), eeg.ID, referredMemberID); err != nil {
		jsonError(w, "nicht gefunden oder bereits verrechnet", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
