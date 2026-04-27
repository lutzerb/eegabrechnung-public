package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/invoice"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// campaignFile holds in-memory file data for a campaign attachment.
type campaignFile struct {
	meta domain.CampaignAttachment
	data []byte
}

// MemberEmailHandler handles bulk email campaign endpoints.
type MemberEmailHandler struct {
	campaignRepo *repository.MemberEmailRepository
	memberRepo   *repository.MemberRepository
	eegRepo      *repository.EEGRepository
}

func sanitizeAttachmentFilename(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = path.Base(name)
	if name == "." || name == "/" || name == "" {
		return "attachment"
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	var safeBase strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z':
			safeBase.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			safeBase.WriteRune(r)
		case r >= '0' && r <= '9':
			safeBase.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			safeBase.WriteRune(r)
		case r == ' ':
			safeBase.WriteByte('_')
		}
	}

	var safeExt strings.Builder
	for _, r := range ext {
		switch {
		case r >= 'a' && r <= 'z':
			safeExt.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			safeExt.WriteRune(r)
		case r >= '0' && r <= '9':
			safeExt.WriteRune(r)
		case r == '.':
			safeExt.WriteRune(r)
		}
	}

	filename := strings.Trim(safeBase.String(), "._")
	if filename == "" {
		filename = "attachment"
	}
	if safeExt.Len() == 0 {
		return filename
	}
	return filename + safeExt.String()
}

func campaignAttachmentPath(campaignID uuid.UUID, filename string) string {
	return filepath.Join("/data/campaigns", campaignID.String(), uuid.NewString()+filepath.Ext(filename))
}

func NewMemberEmailHandler(
	campaignRepo *repository.MemberEmailRepository,
	memberRepo *repository.MemberRepository,
	eegRepo *repository.EEGRepository,
) *MemberEmailHandler {
	return &MemberEmailHandler{
		campaignRepo: campaignRepo,
		memberRepo:   memberRepo,
		eegRepo:      eegRepo,
	}
}

// ListCampaigns handles GET /eegs/{eegID}/communications
func (h *MemberEmailHandler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	campaigns, err := h.campaignRepo.List(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to list campaigns", http.StatusInternalServerError)
		return
	}
	// Strip html_body for list view (can be large)
	for i := range campaigns {
		campaigns[i].HtmlBody = ""
	}
	jsonOK(w, campaigns)
}

// GetCampaign handles GET /eegs/{eegID}/communications/{id}
func (h *MemberEmailHandler) GetCampaign(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}
	var err error
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid campaign ID", http.StatusBadRequest)
		return
	}
	c, err := h.campaignRepo.GetByID(r.Context(), id, eeg.ID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, c)
}

// SendCampaign handles POST /eegs/{eegID}/communications
// Accepts multipart/form-data: subject, html_body, and optional file attachments.
func (h *MemberEmailHandler) SendCampaign(w http.ResponseWriter, r *http.Request) {
	_, eeg, ok := requireAdminEEGAccess(w, r, h.eegRepo)
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "invalid form data", http.StatusBadRequest)
		return
	}

	subject := strings.TrimSpace(r.FormValue("subject"))
	htmlBody := strings.TrimSpace(r.FormValue("html_body"))
	if subject == "" || htmlBody == "" {
		jsonError(w, "subject and html_body are required", http.StatusBadRequest)
		return
	}

	// Collect attachment files from the multipart form
	var attachments []campaignFile
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for _, fhs := range r.MultipartForm.File {
			for _, fh := range fhs {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				data, _ := io.ReadAll(f)
				f.Close()
				mimeType := fh.Header.Get("Content-Type")
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
				filename := sanitizeAttachmentFilename(fh.Filename)
				attachments = append(attachments, campaignFile{
					meta: domain.CampaignAttachment{
						Filename: filename,
						MimeType: mimeType,
						Size:     int64(len(data)),
					},
					data: data,
				})
			}
		}
	}

	// Optional member_ids filter (JSON array of UUIDs)
	var memberIDFilter map[uuid.UUID]struct{}
	if raw := strings.TrimSpace(r.FormValue("member_ids")); raw != "" {
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
			memberIDFilter = make(map[uuid.UUID]struct{}, len(ids))
			for _, s := range ids {
				if id, err := uuid.Parse(s); err == nil {
					memberIDFilter[id] = struct{}{}
				}
			}
		}
	}

	// Fetch all members for this EEG
	members, err := h.memberRepo.ListByEeg(r.Context(), eeg.ID)
	if err != nil {
		jsonError(w, "failed to load members", http.StatusInternalServerError)
		return
	}

	// Fetch EEG for sender name
	// Build attachment metas for storage (no file path yet)
	attMetas := make([]domain.CampaignAttachment, len(attachments))
	for i, a := range attachments {
		attMetas[i] = a.meta
	}

	// Store campaign record in DB first to get the ID
	campaign := &domain.MemberEmailCampaign{
		EegID:          eeg.ID,
		Subject:        subject,
		HtmlBody:       htmlBody,
		RecipientCount: 0,
		Attachments:    attMetas,
	}
	if err := h.campaignRepo.Create(r.Context(), campaign); err != nil {
		jsonError(w, "failed to store campaign", http.StatusInternalServerError)
		return
	}

	// Save attachment files to disk and update meta with file paths
	if len(attachments) > 0 {
		dir := filepath.Join("/data/campaigns", campaign.ID.String())
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("failed to create campaign dir", "error", err, "dir", dir)
		} else {
			for i, a := range attachments {
				path := campaignAttachmentPath(campaign.ID, a.meta.Filename)
				if err := os.WriteFile(path, a.data, 0644); err != nil {
					slog.Error("failed to write attachment", "error", err, "path", path)
					continue
				}
				campaign.Attachments[i].FilePath = path
				attachments[i].meta.FilePath = path
			}
		}
	}

	if eeg.IsDemo {
		jsonError(w, "email sending disabled in demo mode", http.StatusForbidden)
		return
	}

	// Send to selected (or all) active members with a valid email address
	sent := 0
	for _, m := range members {
		if m.Status == "INACTIVE" || m.Email == "" {
			continue
		}
		if memberIDFilter != nil {
			if _, ok := memberIDFilter[m.ID]; !ok {
				continue
			}
		}
		smtpCfg := invoice.SMTPConfig{Host: eeg.SMTPHost, From: eeg.SMTPFrom, Username: eeg.SMTPUser, Password: eeg.SMTPPassword}
		personalizedSubject := applyEmailPlaceholders(subject, m, eeg.Name)
		personalizedBody := applyEmailPlaceholders(htmlBody, m, eeg.Name)
		if err := h.sendHTMLEmail(smtpCfg, m.Email, eeg.Name, personalizedSubject, personalizedBody, attachments); err != nil {
			slog.Error("failed to send campaign email", "member_id", m.ID, "error", err)
		} else {
			sent++
		}
	}

	// Update the recipient count in DB
	h.campaignRepo.UpdateRecipientCount(r.Context(), campaign.ID, sent) //nolint:errcheck
	campaign.RecipientCount = sent

	jsonOK(w, campaign)
}

// sendHTMLEmail builds a multipart MIME message and sends it via SMTP.
func (h *MemberEmailHandler) sendHTMLEmail(smtpCfg invoice.SMTPConfig, toEmail, fromName, subject, htmlBody string, attachments []campaignFile) error {
	from := smtpCfg.From
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, smtpCfg.From)
	}

	var msgBuf bytes.Buffer

	mpWriter := multipart.NewWriter(&msgBuf)

	// Write headers before the multipart body
	var headerBuf bytes.Buffer
	headerBuf.WriteString("From: " + from + "\r\n")
	headerBuf.WriteString("To: " + toEmail + "\r\n")
	headerBuf.WriteString("Subject: " + subject + "\r\n")
	headerBuf.WriteString("MIME-Version: 1.0\r\n")
	headerBuf.WriteString("Content-Type: multipart/mixed; boundary=\"" + mpWriter.Boundary() + "\"\r\n")
	headerBuf.WriteString("\r\n")

	// HTML part
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlPart, err := mpWriter.CreatePart(htmlHeader)
	if err != nil {
		return fmt.Errorf("create html part: %w", err)
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return fmt.Errorf("write html part: %w", err)
	}

	// Attachment parts
	for _, att := range attachments {
		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", att.meta.MimeType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.meta.Filename))
		attPart, err := mpWriter.CreatePart(attHeader)
		if err != nil {
			slog.Error("failed to create attachment part", "filename", att.meta.Filename, "error", err)
			continue
		}
		encoded := base64.StdEncoding.EncodeToString(att.data)
		// Write in 76-char lines per MIME spec
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			attPart.Write([]byte(encoded[i:end] + "\r\n")) //nolint:errcheck
		}
	}

	if err := mpWriter.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	// Combine headers + multipart body
	result := append(headerBuf.Bytes(), msgBuf.Bytes()...)

	var smtpAuth smtp.Auth
	if smtpCfg.Username != "" {
		host := smtpCfg.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		smtpAuth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, host)
	}
	return smtp.SendMail(smtpCfg.Host, smtpAuth, smtpCfg.From, []string{toEmail}, result)
}

// applyEmailPlaceholders replaces known {{placeholder}} tokens with member-specific values.
// Supported: {{vorname}}, {{nachname}}, {{name}}, {{mitglieds_nr}}, {{eeg_name}}, {{email}}
func applyEmailPlaceholders(s string, m domain.Member, eegName string) string {
	fullName := strings.TrimSpace(m.Name1 + " " + m.Name2)
	return strings.NewReplacer(
		"{{vorname}}", m.Name1,
		"{{nachname}}", m.Name2,
		"{{name}}", fullName,
		"{{mitglieds_nr}}", m.MitgliedsNr,
		"{{eeg_name}}", eegName,
		"{{email}}", m.Email,
	).Replace(s)
}
