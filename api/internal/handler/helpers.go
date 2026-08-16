package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/mailutil"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// cleanText trims surrounding whitespace and strips CR/LF and other control
// characters. Use it on user-supplied free text (names, addresses, email) at the
// point it is persisted, so it can never smuggle extra lines into an email header
// downstream. This backs up the sanitisation done in mailutil at header build time.
func cleanText(s string) string {
	return mailutil.SanitizeHeaderValue(strings.TrimSpace(s))
}

// asciiFilename returns an ASCII-only rendering of s for use as the legacy
// Content-Disposition filename= fallback. German umlauts are transliterated
// (ä→ae, ö→oe, ü→ue, ß→ss) rather than dropped; any other character outside
// the printable ASCII range, plus quotes/slashes/control characters that
// would break the quoted-string header or look like a path, is replaced
// with "_".
func asciiFilename(s string) string {
	s = strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue",
		"Ä", "Ae", "Ö", "Oe", "Ü", "Ue",
		"ß", "ss",
	).Replace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 0x20 || r > 0x7e:
			b.WriteByte('_')
		case r == '"' || r == '/' || r == '\\':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// setContentDisposition sets a Content-Disposition header (disposition is
// "attachment" or "inline") that survives filenames containing non-ASCII
// characters, e.g. an EEG name with umlauts. Browsers are only guaranteed to
// read the legacy filename= parameter as ISO-8859-1, so it carries an
// ASCII-transliterated fallback; filename*=UTF-8''... (RFC 5987/6266) carries
// the real name and takes precedence in every current major browser.
func setContentDisposition(w http.ResponseWriter, disposition, filename string) {
	filename = strings.TrimSpace(filename)
	fallback := asciiFilename(filename)
	header := fmt.Sprintf(`%s; filename="%s"`, disposition, fallback)
	if fallback != filename {
		header += "; filename*=UTF-8''" + url.PathEscape(filename)
	}
	w.Header().Set("Content-Disposition", header)
}

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
		eeg, err = eegRepo.GetByIDForUser(r.Context(), eegID, userID, claims.OrganizationID)
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
