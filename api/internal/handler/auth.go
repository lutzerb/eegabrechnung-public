package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
	secret   string
}

func NewAuthHandler(userRepo *repository.UserRepository, secret string) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, secret: secret}
}

// Login godoc
// @Summary     Authenticate user
// @Description Validates email/password credentials and returns a signed JWT together with basic user information.
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       credentials  body  object{email=string,password=string}  true  "Login credentials"
// @Success     200  {object}  object{token=string,user=object{id=string,email=string,name=string,organization_id=string,role=string}}  "JWT token and user info"
// @Failure     400  {object}  map[string]string  "Bad request"
// @Failure     401  {object}  map[string]string  "Invalid credentials"
// @Failure     500  {object}  map[string]string  "Internal error"
// @Router      /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.SignToken(h.secret, user.ID, user.OrganizationID, user.Role, 8*time.Hour)
	if err != nil {
		jsonError(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":              user.ID,
			"email":           user.Email,
			"name":            user.Name,
			"organization_id": user.OrganizationID,
			"role":            user.Role,
		},
	})
}
