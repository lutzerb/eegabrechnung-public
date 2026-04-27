package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userRepo *repository.UserRepository
	eegRepo  *repository.EEGRepository
}

func NewUserHandler(userRepo *repository.UserRepository, eegRepo *repository.EEGRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo, eegRepo: eegRepo}
}

// ListUsers handles GET /api/v1/admin/users
//
//	@Summary		List all users
//	@Description	Returns all users belonging to the caller's organisation. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Produce		json
//	@Success		200	{array}		domain.User
//	@Failure		500	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	users, err := h.userRepo.List(r.Context(), claims.OrganizationID)
	if err != nil {
		jsonError(w, "failed to list users: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, users)
}

// GetUser handles GET /api/v1/admin/users/{userID}
//
//	@Summary		Get a user by ID
//	@Description	Returns a single user record. The user must belong to the caller's organisation. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Produce		json
//	@Param			userID	path		string	true	"User UUID"
//	@Success		200		{object}	domain.User
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users/{userID} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		jsonError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || user.OrganizationID != claims.OrganizationID {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	jsonOK(w, user)
}

// createUserRequest is the body for POST /api/v1/admin/users.
type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin" or "user"
}

// CreateUser handles POST /api/v1/admin/users
//
//	@Summary		Create a new user
//	@Description	Creates a new user in the caller's organisation. The password is bcrypt-hashed before storage. Role defaults to 'user' if not specified or invalid. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Accept			json
//	@Produce		json
//	@Param			body	body		createUserRequest	true	"New user details"
//	@Success		201		{object}	domain.User
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		jsonError(w, "name, email and password are required", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		req.Role = "user"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.Create(r.Context(), claims.OrganizationID, req.Email, string(hash), req.Name, req.Role)
	if err != nil {
		jsonError(w, "failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// updateUserRequest is the body for PUT /api/v1/admin/users/{userID}.
type updateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`     // "admin" or "user"
	Password string `json:"password"` // optional — omit to keep existing password
}

// UpdateUser handles PUT /api/v1/admin/users/{userID}
//
//	@Summary		Update a user
//	@Description	Updates name, email, and role for the given user. If password is provided, it is re-hashed and stored. The user must belong to the caller's organisation. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Accept			json
//	@Produce		json
//	@Param			userID	path		string				true	"User UUID"
//	@Param			body	body		updateUserRequest	true	"Updated user fields"
//	@Success		200		{object}	domain.User
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users/{userID} [put]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		jsonError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	// Verify the target user belongs to the caller's organisation.
	existing, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || existing.OrganizationID != claims.OrganizationID {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Password string `json:"password"` // optional
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		req.Role = "user"
	}

	if err := h.userRepo.Update(r.Context(), userID, req.Name, req.Email, req.Role); err != nil {
		jsonError(w, "failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			jsonError(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		if err := h.userRepo.SetPassword(r.Context(), userID, string(hash)); err != nil {
			jsonError(w, "failed to update password: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		jsonError(w, "user not found after update", http.StatusInternalServerError)
		return
	}
	jsonOK(w, user)
}

// DeleteUser handles DELETE /api/v1/admin/users/{userID}
//
//	@Summary		Delete a user
//	@Description	Permanently deletes the given user. The user must belong to the caller's organisation. Self-deletion is not allowed. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Param			userID	path	string	true	"User UUID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users/{userID} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		jsonError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	// Verify the target user belongs to the caller's organisation.
	target, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || target.OrganizationID != claims.OrganizationID {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	// Prevent self-deletion.
	selfID, _ := uuid.Parse(claims.RegisteredClaims.Subject)
	if userID == selfID {
		jsonError(w, "cannot delete your own account", http.StatusBadRequest)
		return
	}
	if err := h.userRepo.Delete(r.Context(), userID); err != nil {
		jsonError(w, "failed to delete user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetUserEEGs handles GET /api/v1/admin/users/{userID}/eegs
//
//	@Summary		Get EEG assignments for a user
//	@Description	Returns the list of EEG UUIDs that the user has been explicitly assigned to. Admins have implicit access to all EEGs. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Produce		json
//	@Param			userID	path		string	true	"User UUID"
//	@Success		200		{array}		string	"Array of EEG UUIDs"
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users/{userID}/eegs [get]
func (h *UserHandler) GetUserEEGs(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		jsonError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	target, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || target.OrganizationID != claims.OrganizationID {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	ids, err := h.userRepo.GetEEGAssignments(r.Context(), userID)
	if err != nil {
		jsonError(w, "failed to get assignments", http.StatusInternalServerError)
		return
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}
	jsonOK(w, ids)
}

// SetUserEEGs handles PUT /api/v1/admin/users/{userID}/eegs
//
//	@Summary		Set EEG assignments for a user
//	@Description	Replaces the full set of EEG assignments for the user. Pass an empty array to remove all assignments. Requires admin role.
//	@Tags			Benutzerverwaltung
//	@Accept			json
//	@Param			userID	path	string		true	"User UUID"
//	@Param			body	body	[]string	true	"Array of EEG UUIDs to assign"
//	@Success		204		"No Content"
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/admin/users/{userID}/eegs [put]
func (h *UserHandler) SetUserEEGs(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		jsonError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	target, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || target.OrganizationID != claims.OrganizationID {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	var eegIDs []uuid.UUID
	if err := json.NewDecoder(r.Body).Decode(&eegIDs); err != nil {
		jsonError(w, "invalid request body: expected array of EEG UUIDs", http.StatusBadRequest)
		return
	}
	if err := h.userRepo.SetEEGAssignments(r.Context(), userID, eegIDs); err != nil {
		jsonError(w, "failed to set assignments", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
