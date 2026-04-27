package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeTempFile writes an uploaded file to a temp location and returns cleanup func.
func writeTempFile(r io.Reader, originalName string) (string, func(), error) {
	ext := filepath.Ext(originalName)
	f, err := os.CreateTemp("", "upload-*"+ext)
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, err
	}
	f.Close()
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

func requireEEGAccess(w http.ResponseWriter, r *http.Request, eegRepo *repository.EEGRepository) (*auth.Claims, *domain.EEG, bool) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}

	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return nil, nil, false
	}

	var eeg *domain.EEG
	if claims.Role == "admin" {
		eeg, err = eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	} else {
		userID, parseErr := uuid.Parse(claims.Subject)
		if parseErr != nil {
			jsonError(w, "invalid token subject", http.StatusUnauthorized)
			return nil, nil, false
		}
		eeg, err = eegRepo.GetByIDForUser(r.Context(), eegID, userID)
	}
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return nil, nil, false
	}
	return claims, eeg, true
}

func requireAdminEEGAccess(w http.ResponseWriter, r *http.Request, eegRepo *repository.EEGRepository) (*auth.Claims, *domain.EEG, bool) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}
	if claims.Role != "admin" {
		jsonError(w, "forbidden", http.StatusForbidden)
		return nil, nil, false
	}
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return nil, nil, false
	}
	eeg, err := eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return nil, nil, false
	}
	return claims, eeg, true
}
