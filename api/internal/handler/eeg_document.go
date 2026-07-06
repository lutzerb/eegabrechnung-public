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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// EEGDocumentHandler handles EEG document upload, listing, download and deletion.
type EEGDocumentHandler struct {
	docRepo    *repository.EEGDocumentRepository
	portalRepo *repository.MemberPortalRepository
	eegRepo    *repository.EEGRepository
}

func NewEEGDocumentHandler(docRepo *repository.EEGDocumentRepository, portalRepo *repository.MemberPortalRepository, eegRepo *repository.EEGRepository) *EEGDocumentHandler {
	return &EEGDocumentHandler{docRepo: docRepo, portalRepo: portalRepo, eegRepo: eegRepo}
}

// ListDocuments handles GET /eegs/{eegID}/documents (admin)
func (h *EEGDocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	docs, err := h.docRepo.List(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to list documents", http.StatusInternalServerError)
		return
	}
	jsonOK(w, docs)
}

// UploadDocument handles POST /eegs/{eegID}/documents (admin, multipart/form-data)
// Fields: title (required), description (optional), sort_order (optional int), file (required)
func (h *EEGDocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}
	description := strings.TrimSpace(r.FormValue("description"))

	sortOrder := 0
	if s := r.FormValue("sort_order"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			sortOrder = n
		}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Store file on disk: /data/documents/{eegID}/{uuid}_{filename}
	docID := uuid.New()
	safeFilename := filepath.Base(header.Filename)
	storedFilename := fmt.Sprintf("%s_%s", docID.String(), safeFilename)
	dir := filepath.Join("/data/documents", eeg.ID.String())
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("failed to create documents dir", "error", err, "dir", dir)
		jsonError(w, "failed to create storage directory", http.StatusInternalServerError)
		return
	}
	filePath := filepath.Join(dir, storedFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		slog.Error("failed to write document file", "error", err, "path", filePath)
		jsonError(w, "failed to store file", http.StatusInternalServerError)
		return
	}

	showInOnboarding := r.FormValue("show_in_onboarding") == "true"

	doc := &domain.EEGDocument{
		ID:               docID,
		EegID:            eeg.ID,
		Title:            title,
		Description:      description,
		Filename:         safeFilename,
		FilePath:         filePath,
		MimeType:         mimeType,
		FileSizeBytes:    int64(len(data)),
		SortOrder:        sortOrder,
		ShowInOnboarding: showInOnboarding,
	}

	if err := h.docRepo.Create(r.Context(), doc); err != nil {
		slog.Error("failed to store document in DB", "error", err)
		// Clean up file on DB error
		os.Remove(filePath) //nolint:errcheck
		jsonError(w, "failed to save document", http.StatusInternalServerError)
		return
	}

	jsonOK(w, doc)
}

// DeleteDocument handles DELETE /eegs/{eegID}/documents/{docID} (admin)
func (h *EEGDocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	docID, err := uuid.Parse(chi.URLParam(r, "docID"))
	if err != nil {
		jsonError(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	// Fetch before deletion to get file path for cleanup
	doc, err := h.docRepo.GetByID(r.Context(), docID)
	if err != nil {
		jsonError(w, "document not found", http.StatusNotFound)
		return
	}

	if err := h.docRepo.Delete(r.Context(), docID, eeg.ID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "document not found", http.StatusNotFound)
		} else {
			jsonError(w, "failed to delete document", http.StatusInternalServerError)
		}
		return
	}

	// Remove file from disk (non-fatal)
	if doc.FilePath != "" {
		if err := os.Remove(doc.FilePath); err != nil {
			slog.Warn("failed to remove document file", "path", doc.FilePath, "error", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// DownloadDocument handles GET /eegs/{eegID}/documents/{docID}/download (admin)
func (h *EEGDocumentHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	docID, err := uuid.Parse(chi.URLParam(r, "docID"))
	if err != nil {
		jsonError(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	doc, err := h.docRepo.GetByID(r.Context(), docID)
	if err != nil || doc.EegID != eeg.ID {
		jsonError(w, "document not found", http.StatusNotFound)
		return
	}

	h.serveDocumentFile(w, r, doc)
}

// PortalListDocuments handles GET /api/v1/public/portal/documents (member portal)
// Authenticates via X-Portal-Session header.
func (h *EEGDocumentHandler) PortalListDocuments(w http.ResponseWriter, r *http.Request) {
	_, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	docs, err := h.docRepo.List(r.Context(), eegID)
	if err != nil {
		jsonError(w, "failed to list documents", http.StatusInternalServerError)
		return
	}
	jsonOK(w, docs)
}

// PortalDownloadDocument handles GET /api/v1/public/portal/documents/{docID} (member portal)
// Authenticates via X-Portal-Session header.
func (h *EEGDocumentHandler) PortalDownloadDocument(w http.ResponseWriter, r *http.Request) {
	_, eegID, ok := h.portalAuth(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	docID, err := uuid.Parse(chi.URLParam(r, "docID"))
	if err != nil {
		jsonError(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	doc, err := h.docRepo.GetByID(r.Context(), docID)
	if err != nil || doc.EegID != eegID {
		jsonError(w, "document not found", http.StatusNotFound)
		return
	}

	h.serveDocumentFile(w, r, doc)
}

// portalAuth validates the X-Portal-Session header and returns member and EEG IDs.
func (h *EEGDocumentHandler) portalAuth(r *http.Request) (memberID, eegID uuid.UUID, ok bool) {
	token := r.Header.Get("X-Portal-Session")
	if token == "" {
		return uuid.Nil, uuid.Nil, false
	}
	mID, eID, err := h.portalRepo.FindBySessionToken(r.Context(), token)
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return mID, eID, true
}

// PublicDownloadDocument handles GET /api/v1/public/eegs/{eegID}/documents/{docID}
// No authentication required — intended for the public onboarding page.
func (h *EEGDocumentHandler) PublicDownloadDocument(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	docID, err := uuid.Parse(chi.URLParam(r, "docID"))
	if err != nil {
		jsonError(w, "invalid document ID", http.StatusBadRequest)
		return
	}
	doc, err := h.docRepo.GetByID(r.Context(), docID)
	if err != nil || doc.EegID != eegID {
		jsonError(w, "document not found", http.StatusNotFound)
		return
	}
	h.serveDocumentFile(w, r, doc)
}

// PatchDocument handles PATCH /eegs/{eegID}/documents/{docID} (admin)
// Accepts JSON body: {"show_in_onboarding": true|false}
func (h *EEGDocumentHandler) PatchDocument(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	docID, err := uuid.Parse(chi.URLParam(r, "docID"))
	if err != nil {
		jsonError(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	var body struct {
		ShowInOnboarding *bool `json:"show_in_onboarding"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.ShowInOnboarding != nil {
		// Verify document belongs to this EEG before updating
		doc, err := h.docRepo.GetByID(r.Context(), docID)
		if err != nil || doc.EegID != eeg.ID {
			jsonError(w, "document not found", http.StatusNotFound)
			return
		}
		if err := h.docRepo.SetShowInOnboarding(r.Context(), docID, *body.ShowInOnboarding); err != nil {
			jsonError(w, "failed to update document", http.StatusInternalServerError)
			return
		}
		doc.ShowInOnboarding = *body.ShowInOnboarding
		jsonOK(w, doc)
		return
	}

	jsonError(w, "nothing to update", http.StatusBadRequest)
}

// serveDocumentFile opens and streams a document file to the HTTP response.
func (h *EEGDocumentHandler) serveDocumentFile(w http.ResponseWriter, r *http.Request, doc *domain.EEGDocument) {
	if doc.FilePath == "" {
		jsonError(w, "file not available", http.StatusNotFound)
		return
	}
	f, err := os.Open(doc.FilePath)
	if err != nil {
		slog.Error("failed to open document file", "path", doc.FilePath, "error", err)
		jsonError(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	mimeType := doc.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, doc.Filename))
	io.Copy(w, f) //nolint:errcheck
}
