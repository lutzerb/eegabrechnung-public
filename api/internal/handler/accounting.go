package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	"github.com/xuri/excelize/v2"
)

type AccountingHandler struct {
	eegRepo     *repository.EEGRepository
	invoiceRepo *repository.InvoiceRepository
	memberRepo  *repository.MemberRepository
}

func NewAccountingHandler(
	eegRepo *repository.EEGRepository,
	invoiceRepo *repository.InvoiceRepository,
	memberRepo *repository.MemberRepository,
) *AccountingHandler {
	return &AccountingHandler{eegRepo: eegRepo, invoiceRepo: invoiceRepo, memberRepo: memberRepo}
}

// accountingRow is a flattened row combining invoice + member data for export.
type accountingRow struct {
	Invoice *domain.Invoice
	Member  *domain.Member
}

// Export godoc
//
//	@Summary		Export accounting data
//	@Description	Exports invoices for the given date range as an XLSX Buchungsjournal or a DATEV EXTF Buchungsstapel CSV. Cancelled invoices are excluded. Both from and to are required.
//	@Tags			Buchhaltung
//	@Produce		application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,text/csv
//	@Param			eegID	path		string	true	"EEG ID (UUID)"
//	@Param			from	query		string	true	"Start date (YYYY-MM-DD, inclusive)"
//	@Param			to		query		string	true	"End date (YYYY-MM-DD, inclusive)"
//	@Param			format	query		string	false	"Export format: xlsx (default) or datev"	Enums(xlsx,datev)
//	@Success		200		{file}		application/octet-stream	"File attachment (XLSX or CSV)"
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/accounting/export [get]
func (h *AccountingHandler) Export(w http.ResponseWriter, r *http.Request) {
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "xlsx"
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		jsonError(w, "invalid 'from' date (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		jsonError(w, "invalid 'to' date (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	// Include full last day
	to = to.Add(24*time.Hour - time.Second)

	claims := auth.ClaimsFromContext(r.Context())
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	invoices, err := h.invoiceRepo.ListByEegAndPeriod(r.Context(), eegID, from, to)
	if err != nil {
		jsonError(w, "failed to load invoices", http.StatusInternalServerError)
		return
	}

	// Load member map for all involved members
	memberMap := map[uuid.UUID]*domain.Member{}
	for i := range invoices {
		inv := &invoices[i]
		if _, ok := memberMap[inv.MemberID]; !ok {
			m, err := h.memberRepo.GetByID(r.Context(), inv.MemberID)
			if err == nil {
				memberMap[inv.MemberID] = m
			}
		}
	}

	rows := make([]accountingRow, 0, len(invoices))
	for i := range invoices {
		inv := &invoices[i]
		if inv.Status == "cancelled" {
			continue
		}
		rows = append(rows, accountingRow{
			Invoice: inv,
			Member:  memberMap[inv.MemberID],
		})
	}

	switch format {
	case "datev":
		data, err := generateDatevExport(eeg, rows, from, to)
		if err != nil {
			jsonError(w, "failed to generate DATEV export: "+err.Error(), http.StatusInternalServerError)
			return
		}
		filename := fmt.Sprintf("buchungen_%s_%s.csv", from.Format("2006-01"), to.Format("2006-01"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Write(data)
	default: // xlsx
		data, err := generateXLSXExport(rows)
		if err != nil {
			jsonError(w, "failed to generate XLSX export: "+err.Error(), http.StatusInternalServerError)
			return
		}
		filename := fmt.Sprintf("buchungsjournal_%s_%s.xlsx", from.Format("2006-01"), to.Format("2006-01"))
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Write(data)
	}
}

// memberDebitorAccount computes the debitor account number for a member.
// Uses numeric part of MitgliedsNr added to the prefix (e.g. "0042" + 10000 = 10042).
func memberDebitorAccount(prefix int, mitgliedsNr string) int {
	re := regexp.MustCompile(`[0-9]+`)
	numStr := re.FindString(mitgliedsNr)
	if numStr != "" {
		n, err := strconv.Atoi(numStr)
		if err == nil {
			return prefix + n
		}
	}
	return prefix
}

// formatDE formats a float as German decimal (comma as separator, no thousand sep).
func formatDE(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	return strings.Replace(s, ".", ",", 1)
}

func memberFullName(m *domain.Member) string {
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m.Name1 + " " + m.Name2)
}

// generateXLSXExport produces a comprehensive accounting XLSX.
func generateXLSXExport(rows []accountingRow) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Buchungsjournal"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Belegdatum", "Belegnummer", "Belegtyp",
		"Mitglied", "UID-Nummer",
		"Zeitraum von", "Zeitraum bis",
		"Bezug kWh", "Einspeisung kWh",
		"Nettobetrag EUR", "MwSt-Satz %", "MwSt-Betrag EUR", "Bruttobetrag EUR",
		"Status",
	}

	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	styleAmount, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4, // #,##0.00
	})

	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	for i, row := range rows {
		r := i + 2
		inv := row.Invoice
		m := row.Member

		docTypeLabel := "Rechnung"
		if inv.DocumentType == "credit_note" {
			docTypeLabel = "Gutschrift"
		}

		uid := ""
		if m != nil {
			uid = m.UidNummer
		}

		cells := []interface{}{
			inv.CreatedAt.Format("02.01.2006"),
			formatInvoiceNumber(inv),
			docTypeLabel,
			memberFullName(m),
			uid,
			inv.PeriodStart.Format("02.01.2006"),
			inv.PeriodEnd.Format("02.01.2006"),
			inv.ConsumptionKwh,
			inv.GenerationKwh,
			inv.NetAmount,
			inv.VatPctApplied,
			inv.VatAmount,
			inv.TotalAmount,
			inv.Status,
		}

		for col, val := range cells {
			cell, _ := excelize.CoordinatesToCellName(col+1, r)
			f.SetCellValue(sheet, cell, val)
			// Apply number format to EUR and kWh columns
			if col >= 7 && col <= 12 {
				f.SetCellStyle(sheet, cell, cell, styleAmount)
			}
		}
	}

	// Auto-fit columns
	colWidths := []float64{12, 14, 10, 25, 16, 12, 12, 12, 12, 14, 10, 14, 14, 10}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// generateDatevExport produces a DATEV EXTF Buchungsstapel CSV.
func generateDatevExport(eeg *domain.EEG, rows []accountingRow, from, to time.Time) ([]byte, error) {
	var buf bytes.Buffer

	// DATEV EXTF uses semicolons and Windows-style line endings
	berater := eeg.DatevConsultantNr
	if berater == "" {
		berater = "70000"
	}
	mandant := eeg.DatevClientNr
	if mandant == "" {
		mandant = "1"
	}

	now := time.Now()
	ts := now.Format("20060102150405")
	wjBeginn := fmt.Sprintf("%d0101", from.Year())
	vonStr := from.Format("20060102")
	bisStr := to.Format("20060102")
	bezeichnung := fmt.Sprintf("eegabrechnung Export %s bis %s", from.Format("01.2006"), to.Format("01.2006"))

	// Row 1: Vorlaufsatz (30 semicolon-separated fields)
	vorlauf := strings.Join([]string{
		`"EXTF"`, "700", "21", `"Buchungsstapel"`, "13",
		ts, `""`,
		berater, `""`,
		mandant, `""`,
		wjBeginn, `""`,
		"0", `""`,
		vonStr, bisStr,
		`"` + bezeichnung + `"`,
		`""`, "0", `"EUR"`, `""`, `""`, `""`, `""`, `""`, `""`, `""`, `""`, `""`,
	}, ";")
	buf.WriteString(vorlauf + "\r\n")

	// Row 2: Column headers (minimum required)
	headerCols := []string{
		"Umsatz (ohne Soll/Haben-Kz)", "Soll/Haben-Kennzeichen", "WKZ Umsatz",
		"Kurs", "Basis-Umsatz", "WKZ Basis-Umsatz",
		"Konto", "Gegenkonto (ohne BU-Schlüssel)", "BU-Schlüssel",
		"Belegdatum", "Belegfeld 1", "Belegfeld 2", "Skonto", "Buchungstext",
	}
	buf.WriteString(strings.Join(headerCols, ";") + "\r\n")

	// Data rows
	revAccount := eeg.AccountingRevenueAccount
	if revAccount == 0 {
		revAccount = 4000
	}
	expAccount := eeg.AccountingExpenseAccount
	if expAccount == 0 {
		expAccount = 5000
	}
	debPrefix := eeg.AccountingDebitorPrefix
	if debPrefix == 0 {
		debPrefix = 10000
	}

	w := csv.NewWriter(&buf)
	w.Comma = ';'

	for _, row := range rows {
		inv := row.Invoice
		m := row.Member

		// Absolute amount (always positive in DATEV)
		amount := inv.TotalAmount
		if amount < 0 {
			amount = -amount
		}

		// S/H Kennzeichen
		sh := "S"
		if inv.TotalAmount < 0 {
			sh = "H"
		}

		// Konto / Gegenkonto
		debitor := memberDebitorAccount(debPrefix, "")
		if m != nil {
			debitor = memberDebitorAccount(debPrefix, m.MitgliedsNr)
		}

		var konto, gegenkonto int
		if inv.DocumentType == "credit_note" {
			// Gutschrift: Aufwand an Debitor
			konto = expAccount
			gegenkonto = debitor
		} else {
			// Rechnung: Debitor an Erlös
			konto = debitor
			gegenkonto = revAccount
		}

		// BU-Schlüssel (tax code)
		buSchluessel := ""
		if inv.VatPctApplied == 20 {
			buSchluessel = "9"
		} else if inv.VatPctApplied == 10 {
			buSchluessel = "8"
		}

		// Belegdatum: DDMM format
		belegdatum := inv.CreatedAt.Format("0201")

		// Belegfeld 1: invoice number, max 12 chars
		belegfeld1 := formatInvoiceNumber(inv)
		if len(belegfeld1) > 12 {
			belegfeld1 = belegfeld1[:12]
		}

		// Buchungstext: member name, max 60 chars
		buchungstext := memberFullName(m)
		if len(buchungstext) > 60 {
			buchungstext = buchungstext[:60]
		}

		record := []string{
			formatDE(amount),
			sh,
			"EUR",
			"", "", "", // Kurs, Basis-Umsatz, WKZ
			strconv.Itoa(konto),
			strconv.Itoa(gegenkonto),
			buSchluessel,
			belegdatum,
			belegfeld1,
			"",            // Belegfeld 2
			"",            // Skonto
			buchungstext,
		}
		w.Write(record)
	}
	w.Flush()

	return buf.Bytes(), nil
}

// formatInvoiceNumber returns the formatted invoice number string.
func formatInvoiceNumber(inv *domain.Invoice) string {
	if inv.InvoiceNumber == nil {
		return inv.ID.String()[:8]
	}
	return strconv.Itoa(*inv.InvoiceNumber)
}
