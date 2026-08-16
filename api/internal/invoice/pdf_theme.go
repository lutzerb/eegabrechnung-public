package invoice

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// pdf_theme.go implements the alternate "individuell" invoice visual design
// (per-EEG opt-in, see eeg.InvoiceDesign), inspired by a customer-supplied
// reference PDF ("Oikos" layout). The production GeneratePDF / GenerateCreditNotePDF
// / GenerateStornorechnung / embedLogo in pdf.go are untouched — helpers here are
// intentionally duplicated rather than shared, so a bug in the themed layout can
// never affect the default rendering path. Only small, pure, already-shared
// helpers (drawTotalRow, drawMpSubRow, drawBarChart, formatKwh, formatAmount,
// germanMonth, periodLabel) are reused directly from pdf.go — those always render
// in DejaVu Sans regardless of the chosen theme font (see newThemedPDF), since
// changing them would touch the production rendering path.

//go:embed fonts/Roboto-Regular.ttf
var robotoRegular []byte

//go:embed fonts/Roboto-Bold.ttf
var robotoBold []byte

//go:embed fonts/OpenSans-Regular.ttf
var openSansRegular []byte

//go:embed fonts/OpenSans-Bold.ttf
var openSansBold []byte

//go:embed fonts/PTSerif-Regular.ttf
var ptSerifRegular []byte

//go:embed fonts/PTSerif-Bold.ttf
var ptSerifBold []byte

type themeFontAsset struct{ Regular, Bold []byte }

// themeFonts maps the selectable invoice_font_family values to their embedded
// TTF bytes. "dejavu" reuses the same font pdf.go already embeds (dejaVuSans /
// dejaVuSansBold) — no separate copy needed.
var themeFonts = map[string]themeFontAsset{
	"dejavu":   {dejaVuSans, dejaVuSansBold},
	"roboto":   {robotoRegular, robotoBold},
	"opensans": {openSansRegular, openSansBold},
	"ptserif":  {ptSerifRegular, ptSerifBold},
}

// newThemedPDF creates an fpdf instance for the themed renderers. It registers
// the chosen theme font family under the name "Theme" (falling back to "dejavu"
// for an unknown/empty value), PLUS "DejaVu" unconditionally so the shared
// low-level helpers borrowed from pdf.go (drawTotalRow, drawMpSubRow,
// drawBarChart) keep working — those always render in DejaVu Sans, independent
// of the chosen theme font.
func newThemedPDF(theme InvoiceTheme) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("DejaVu", "", dejaVuSans)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", dejaVuSansBold)
	asset, ok := themeFonts[theme.FontFamily]
	if !ok {
		asset = themeFonts["dejavu"]
	}
	pdf.AddUTF8FontFromBytes("Theme", "", asset.Regular)
	pdf.AddUTF8FontFromBytes("Theme", "B", asset.Bold)
	return pdf
}

// InvoiceTheme carries the visual knobs for the themed renderers
// (GeneratePDFThemed, GenerateCreditNotePDFThemed, GenerateStornorechnungThemed).
// Persisted per EEG as eeg.Invoice{Design,AccentColor,LogoLeft,FontFamily,FontSize}
// and only applied when eeg.InvoiceDesign == "individuell".
type InvoiceTheme struct {
	AccentR, AccentG, AccentB int
	LogoLeft                  bool
	FontFamily                string  // dejavu | roboto | opensans | ptserif
	BaseFontSize              float64 // pt, 8-12 — body text size; other sizes are relative offsets
	RowSpacing                float64 // scale factor, 0.7-1.3 — multiplies every line/row height and inter-section gap
}

// DefaultOikosTheme approximates the accent color and logo placement of the
// customer-supplied reference PDF used as the initial design inspiration.
func DefaultOikosTheme() InvoiceTheme {
	return InvoiceTheme{AccentR: 201, AccentG: 184, AccentB: 154, LogoLeft: true, FontFamily: "dejavu", BaseFontSize: 10, RowSpacing: 1.0}
}

func (t InvoiceTheme) apply(pdf *fpdf.Fpdf) {
	pdf.SetFillColor(t.AccentR, t.AccentG, t.AccentB)
}

// size returns theme.BaseFontSize + delta, falling back to a sane default (10)
// when BaseFontSize wasn't set (e.g. a zero-value InvoiceTheme).
func (t InvoiceTheme) size(delta float64) float64 {
	base := t.BaseFontSize
	if base <= 0 {
		base = 10
	}
	return base + delta
}

// h scales a line/row height or gap by RowSpacing, falling back to 1.0 (no
// scaling) when unset (e.g. a zero-value InvoiceTheme). Used for every
// CellFormat/MultiCell height and pdf.Ln() gap in the themed renderers so a
// single "Zeilenabstand" knob controls line spacing and table row padding
// together — matches how compact/spacious the whole document reads.
func (t InvoiceTheme) h(base float64) float64 {
	scale := t.RowSpacing
	if scale <= 0 {
		scale = 1.0
	}
	return base * scale
}

// ln is pdf.Ln(base) with base scaled by theme.h — shorthand for the many
// inter-section vertical gaps in the themed renderers.
func (t InvoiceTheme) ln(pdf *fpdf.Fpdf, base float64) {
	pdf.Ln(t.h(base))
}

// ThemeFromEEG builds an InvoiceTheme from the persisted per-EEG design fields
// (see domain.EEG / migration 093). Falls back to DefaultOikosTheme's accent
// color when eeg.InvoiceAccentColor isn't a valid "#rrggbb" hex string (handler
// validation should already reject that, but PDF generation shouldn't panic on
// stale/bad data either).
func ThemeFromEEG(eeg *domain.EEG) InvoiceTheme {
	theme := DefaultOikosTheme()
	if r, g, b, ok := parseHexColor(eeg.InvoiceAccentColor); ok {
		theme.AccentR, theme.AccentG, theme.AccentB = r, g, b
	}
	theme.LogoLeft = eeg.InvoiceLogoLeft
	if eeg.InvoiceFontFamily != "" {
		theme.FontFamily = eeg.InvoiceFontFamily
	}
	if eeg.InvoiceFontSize > 0 {
		theme.BaseFontSize = float64(eeg.InvoiceFontSize)
	}
	if eeg.InvoiceRowSpacing > 0 {
		theme.RowSpacing = eeg.InvoiceRowSpacing
	}
	return theme
}

func parseHexColor(hex string) (r, g, b int, ok bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	var vals [3]int64
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseInt(hex[1+i*2:3+i*2], 16, 32)
		if err != nil {
			return 0, 0, 0, false
		}
		vals[i] = v
	}
	return int(vals[0]), int(vals[1]), int(vals[2]), true
}

// EnergyPeriodRow shows the Netzbezug (grid draw) vs. community-covered-
// consumption split for one billing period. Zaehlpunkt is only rendered as a
// group sub-heading when a member has more than one meter point (see
// drawEnergyPeriodTable) — matches the reference PDF's flat layout for the
// (common) single-ZP case.
type EnergyPeriodRow struct {
	Zaehlpunkt            string
	ZeitraumVon           time.Time
	ZeitraumBis           time.Time
	GesamtverbrauchKwh    float64
	NetzbezugKwh          float64
	CommunityVerbrauchKwh float64
}

// GenerationPeriodRow is the generation-side counterpart of EnergyPeriodRow:
// GesamteinspeisungKwh is the meter's total feed-in, AbnahmeKwh is the portion
// absorbed/purchased by the energy community (the billed Einspeisung amount),
// and ResteinspeisungKwh is the remainder fed into the public grid
// (GesamteinspeisungKwh - AbnahmeKwh). Rendered as a second table by
// drawGenerationPeriodTable, right after the consumption table, whenever the
// member has generation meter points.
type GenerationPeriodRow struct {
	Zaehlpunkt           string
	ZeitraumVon          time.Time
	ZeitraumBis          time.Time
	GesamteinspeisungKwh float64
	AbnahmeKwh           float64
	ResteinspeisungKwh   float64
}

// addressBlockLineHeight is the row height used for every line of the EEG
// header address block (Name/Strasse/PLZ Ort/UID-Nr) in the themed renderers.
const addressBlockLineHeight = 5.0

// addressBlockHeight returns the total height of the EEG header address block
// (Name is always shown; Strasse/PLZ+Ort/UID-Nr only when set) — used to scale
// the logo to the same height so the two sit flush next to each other instead
// of the logo using an arbitrary fixed size.
func addressBlockHeight(eeg *domain.EEG, theme InvoiceTheme) float64 {
	lines := 1.0 // Name always shown
	if eeg.Strasse != "" {
		lines++
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		lines++
	}
	if eeg.UidNummer != "" {
		lines++
	}
	return lines * theme.h(addressBlockLineHeight)
}

// embedLogoAt places a logo image at an arbitrary position/size. Pass w=0 to
// scale width automatically from h (aspect ratio preserved) or vice versa —
// same semantics as fpdf.ImageOptions. Duplicated from embedLogo (pdf.go)
// instead of parameterizing it, so the production embedLogo — and everything
// that calls it — stays byte-for-byte unchanged.
func embedLogoAt(pdf *fpdf.Fpdf, logoPath string, x, y, w, h float64) {
	if logoPath == "" {
		return
	}
	data, err := os.ReadFile(logoPath)
	if err != nil {
		return
	}
	imgType := "JPG"
	if strings.HasSuffix(strings.ToLower(logoPath), ".png") {
		imgType = "PNG"
	}
	opt := fpdf.ImageOptions{ImageType: imgType, ReadDpi: true}
	pdf.RegisterImageOptionsReader("logo", opt, bytes.NewReader(data))
	pdf.ImageOptions("logo", x, y, w, h, false, opt, 0, "")
}

// orDefault returns s, or def when s is empty.
func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// printableWidth returns the usable page width between the left and right margins.
func printableWidth(pdf *fpdf.Fpdf) float64 {
	pw, _ := pdf.GetPageSize()
	lMargin, _, rMargin, _ := pdf.GetMargins()
	return pw - lMargin - rMargin
}

// wrapLines splits text into lines that fit width w using the pdf's current
// font, always returning at least one (possibly empty) line.
func wrapLines(pdf *fpdf.Fpdf, text string, w float64) []string {
	raw := pdf.SplitLines([]byte(text), w)
	if len(raw) == 0 {
		return []string{""}
	}
	lines := make([]string, len(raw))
	for i, l := range raw {
		lines[i] = string(l)
	}
	return lines
}

// drawWrappingHeaderRow renders one header row across multiple columns, each
// wrapping onto as many lines as needed, growing the whole row to the tallest
// wrapped column so every header cell stays aligned. Uses the pdf's current
// font (caller sets family/size/style before calling). Labels here are
// admin-editable (see drawEnergyPeriodTable) and can be longer than the fixed
// column widths were tuned for.
func drawWrappingHeaderRow(pdf *fpdf.Fpdf, theme InvoiceTheme, cols []float64, aligns []string, labels []string) {
	lineH := theme.h(4.2)
	x0, y0 := pdf.GetXY()
	linesPerCol := make([][]string, len(labels))
	maxLines := 1
	for i, label := range labels {
		lines := wrapLines(pdf, label, cols[i]-2)
		linesPerCol[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	rowH := float64(maxLines) * lineH
	theme.apply(pdf)
	x := x0
	for i, w := range cols {
		pdf.Rect(x, y0, w, rowH, "FD")
		colY := y0 + (rowH-float64(len(linesPerCol[i]))*lineH)/2.0
		for j, line := range linesPerCol[i] {
			pdf.SetXY(x, colY+float64(j)*lineH)
			pdf.CellFormat(w, lineH, line, "", 0, aligns[i], false, 0, "")
		}
		x += w
	}
	pdf.SetXY(x0, y0+rowH)
}

// drawWrappingLineRow draws one 4-column line-item row (Beschreibung/kWh/Preis/
// Betrag), wrapping the Beschreibung text onto multiple lines when it doesn't
// fit colDesc and growing the row so every cell in the row stays aligned —
// description text can be long (fee labels, multi-Zählpunkt notes) and would
// otherwise overflow past the column border, since fpdf's CellFormat never wraps.
func drawWrappingLineRow(pdf *fpdf.Fpdf, theme InvoiceTheme, colDesc, colKwh, colPrice, colAmount, baseRowH float64, desc, kwh, price, amount string) {
	lineH := theme.h(5.0)
	x, y := pdf.GetXY()
	lines := wrapLines(pdf, desc, colDesc-2)
	rowH := baseRowH
	if wrapped := float64(len(lines)) * lineH; wrapped > rowH {
		rowH = wrapped
	}
	pdf.Rect(x, y, colDesc, rowH, "D")
	descY := y + (rowH-float64(len(lines))*lineH)/2.0
	for i, line := range lines {
		pdf.SetXY(x, descY+float64(i)*lineH)
		pdf.CellFormat(colDesc, lineH, line, "", 0, "L", false, 0, "")
	}
	pdf.SetXY(x+colDesc, y)
	pdf.CellFormat(colKwh, rowH, kwh, "1", 0, "R", false, 0, "")
	pdf.CellFormat(colPrice, rowH, price, "1", 0, "R", false, 0, "")
	pdf.CellFormat(colAmount, rowH, amount, "1", 1, "R", false, 0, "")
}

// drawEnergyPeriodTable renders the Oikos-style energy breakdown table (Zeitraum
// von/bis, Gesamtverbrauch, Netzbezug, Community-Verbrauch) with an accent-filled
// header and an unbordered total row, matching the reference PDF's look. When
// rows span more than one distinct Zählpunkt, each ZP's rows are preceded by a
// small "Zählpunkt: {ZP}" sub-heading (mirrors drawMpSubRow's convention in the
// pricing table) — the reference PDF only ever shows one ZP and has no such
// heading, so the single-ZP case stays visually identical to it.
func drawEnergyPeriodTable(pdf *fpdf.Fpdf, rows []EnergyPeriodRow, theme InvoiceTheme, eeg *domain.EEG) {
	if len(rows) == 0 {
		return
	}
	fullW := printableWidth(pdf)
	colVon := 30.0
	colBis := 30.0
	colGesamt := 38.0
	colNetz := 36.0
	colCommunity := fullW - colVon - colBis - colGesamt - colNetz
	rowH := theme.h(7.0)
	headerStartSize := theme.size(-1)

	distinctZPs := map[string]bool{}
	for _, r := range rows {
		if r.Zaehlpunkt != "" {
			distinctZPs[r.Zaehlpunkt] = true
		}
	}
	showZPHeadings := len(distinctZPs) > 1

	labelVon := orDefault(eeg.InvoiceEnergyLabelZeitraumVon, "Zeitraum von")
	labelBis := orDefault(eeg.InvoiceEnergyLabelZeitraumBis, "Zeitraum bis")
	labelGesamt := orDefault(eeg.InvoiceEnergyLabelGesamtverbrauch, "Gesamtverbrauch kWh")
	labelNetz := orDefault(eeg.InvoiceEnergyLabelNetzbezug, "Netzbezug kWh")
	labelCommunity := orDefault(eeg.InvoiceEnergyLabelCommunityVerbrauch, "Community-Verbrauch kWh")

	pdf.SetFont("Theme", "B", headerStartSize)
	drawWrappingHeaderRow(pdf, theme,
		[]float64{colVon, colBis, colGesamt, colNetz, colCommunity},
		[]string{"L", "L", "R", "R", "R"},
		[]string{labelVon, labelBis, labelGesamt, labelNetz, labelCommunity},
	)

	pdf.SetFont("Theme", "", theme.size(-1))
	pdf.SetFillColor(255, 255, 255)
	var totalGesamt, totalNetz, totalCommunity float64
	lastZP := ""
	first := true
	for _, r := range rows {
		if showZPHeadings && (first || r.Zaehlpunkt != lastZP) {
			pdf.SetFont("Theme", "B", theme.size(-2.5))
			pdf.SetTextColor(120, 120, 120)
			pdf.CellFormat(fullW, theme.h(4.5), "Zählpunkt: "+r.Zaehlpunkt, "", 1, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Theme", "", theme.size(-1))
			lastZP = r.Zaehlpunkt
		}
		first = false
		pdf.CellFormat(colVon, rowH, r.ZeitraumVon.Format("02.01.2006"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colBis, rowH, r.ZeitraumBis.Format("02.01.2006"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colGesamt, rowH, formatKwh(r.GesamtverbrauchKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colNetz, rowH, formatKwh(r.NetzbezugKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colCommunity, rowH, formatKwh(r.CommunityVerbrauchKwh), "1", 1, "R", false, 0, "")
		totalGesamt += r.GesamtverbrauchKwh
		totalNetz += r.NetzbezugKwh
		totalCommunity += r.CommunityVerbrauchKwh
	}
	// Unbordered total row, right-aligned under the last column — matches the
	// reference PDF, which shows only the summed community-consumption figure.
	pdf.SetFont("Theme", "B", theme.size(-1))
	pdf.CellFormat(colVon+colBis+colGesamt+colNetz, rowH, "", "", 0, "", false, 0, "")
	pdf.CellFormat(colCommunity, rowH, formatKwh(totalCommunity), "", 1, "R", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	theme.ln(pdf, 4)
}

// drawGenerationPeriodTable renders the Einspeisung counterpart of
// drawEnergyPeriodTable: Gesamteinspeisung (total feed-in) / Abnahme durch
// Energiegemeinschaft (community-absorbed, the billed amount) / Resteinspeisung
// (residual fed into the public grid). Uses the same column widths as the
// consumption table so the two tables line up visually when both are shown.
// Unlike drawEnergyPeriodTable's total row (which only totals the single
// billed column), both Abnahme and Resteinspeisung totals are shown here since
// both are meaningful to a producer.
func drawGenerationPeriodTable(pdf *fpdf.Fpdf, rows []GenerationPeriodRow, theme InvoiceTheme, eeg *domain.EEG) {
	if len(rows) == 0 {
		return
	}
	fullW := printableWidth(pdf)
	colVon := 30.0
	colBis := 30.0
	colGesamt := 38.0
	colAbnahme := 36.0
	colRest := fullW - colVon - colBis - colGesamt - colAbnahme
	rowH := theme.h(7.0)
	headerStartSize := theme.size(-1)

	distinctZPs := map[string]bool{}
	for _, r := range rows {
		if r.Zaehlpunkt != "" {
			distinctZPs[r.Zaehlpunkt] = true
		}
	}
	showZPHeadings := len(distinctZPs) > 1

	labelVon := orDefault(eeg.InvoiceEnergyLabelZeitraumVon, "Zeitraum von")
	labelBis := orDefault(eeg.InvoiceEnergyLabelZeitraumBis, "Zeitraum bis")
	labelGesamt := orDefault(eeg.InvoiceEnergyLabelGesamteinspeisung, "Gesamteinspeisung kWh")
	labelAbnahme := orDefault(eeg.InvoiceEnergyLabelAbnahmeEnergiegemeinschaft, "Abnahme durch Energiegemeinschaft kWh")
	labelRest := orDefault(eeg.InvoiceEnergyLabelResteinspeisung, "Resteinspeisung kWh")

	pdf.SetFont("Theme", "B", headerStartSize)
	drawWrappingHeaderRow(pdf, theme,
		[]float64{colVon, colBis, colGesamt, colAbnahme, colRest},
		[]string{"L", "L", "R", "R", "R"},
		[]string{labelVon, labelBis, labelGesamt, labelAbnahme, labelRest},
	)

	pdf.SetFont("Theme", "", theme.size(-1))
	pdf.SetFillColor(255, 255, 255)
	var totalAbnahme, totalRest float64
	lastZP := ""
	first := true
	for _, r := range rows {
		if showZPHeadings && (first || r.Zaehlpunkt != lastZP) {
			pdf.SetFont("Theme", "B", theme.size(-2.5))
			pdf.SetTextColor(120, 120, 120)
			pdf.CellFormat(fullW, theme.h(4.5), "Zählpunkt: "+r.Zaehlpunkt, "", 1, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Theme", "", theme.size(-1))
			lastZP = r.Zaehlpunkt
		}
		first = false
		pdf.CellFormat(colVon, rowH, r.ZeitraumVon.Format("02.01.2006"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colBis, rowH, r.ZeitraumBis.Format("02.01.2006"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colGesamt, rowH, formatKwh(r.GesamteinspeisungKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAbnahme, rowH, formatKwh(r.AbnahmeKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colRest, rowH, formatKwh(r.ResteinspeisungKwh), "1", 1, "R", false, 0, "")
		totalAbnahme += r.AbnahmeKwh
		totalRest += r.ResteinspeisungKwh
	}
	pdf.SetFont("Theme", "B", theme.size(-1))
	pdf.CellFormat(colVon+colBis+colGesamt, rowH, "", "", 0, "", false, 0, "")
	pdf.CellFormat(colAbnahme, rowH, formatKwh(totalAbnahme), "", 0, "R", false, 0, "")
	pdf.CellFormat(colRest, rowH, formatKwh(totalRest), "", 1, "R", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	theme.ln(pdf, 4)
}

// GeneratePDFThemed renders a consumer/prosumer "Rechnung" with the Oikos-style
// visual theme. It mirrors GeneratePDF's data logic (multi-month/single-month,
// prosumer/consumer branches) but restyles the header, logo, and pricing table.
// energyRows/generationRows are optional (nil-safe) — pass nil to omit the
// respective breakdown table (e.g. energyRows for a pure producer, generationRows
// for a pure consumer).
func GeneratePDFThemed(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, vat VATOptions, history []MonthlyKwh, energyRows []EnergyPeriodRow, generationRows []GenerationPeriodRow, theme InvoiceTheme) ([]byte, error) {
	pdf := newThemedPDF(theme)
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	period := periodLabel(inv.PeriodStart)

	invoiceNr := shortID(inv.ID.String())
	if inv.InvoiceNumber != nil {
		digits := eeg.InvoiceNumberDigits
		if digits <= 0 {
			digits = 4
		}
		invoiceNr = fmt.Sprintf("%s%0*d", eeg.InvoiceNumberPrefix, digits, *inv.InvoiceNumber)
	}

	// ── Header: logo + EEG address block side by side ────────────────────────
	logoX, addrX, addrW := 150.0, 20.0, 90.0
	if theme.LogoLeft {
		logoX, addrX, addrW = 20.0, 130.0, 60.0
	}
	embedLogoAt(pdf, eeg.LogoPath, logoX, 15, 0, addressBlockHeight(eeg, theme))

	pdf.SetXY(addrX, 15)
	pdf.SetFont("Theme", "B", theme.size(1))
	pdf.CellFormat(addrW, theme.h(5), eeg.Name, "", 2, "R", false, 0, "")
	pdf.SetX(addrX)
	pdf.SetFont("Theme", "", theme.size(-1))
	if eeg.Strasse != "" {
		pdf.CellFormat(addrW, theme.h(5), eeg.Strasse, "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(addrW, theme.h(5), strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(addrW, theme.h(5), "UID-Nr.: "+eeg.UidNummer, "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}

	pdf.SetXY(20, 45)

	// ── Recipient block ──────────────────────────────────────────────────────
	pdf.SetFont("Theme", "", theme.size(0))
	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	pdf.CellFormat(0, theme.h(6), fullName, "", 1, "L", false, 0, "")
	if member.Strasse != "" {
		pdf.CellFormat(0, theme.h(6), member.Strasse, "", 1, "L", false, 0, "")
	}
	if member.Plz != "" || member.Ort != "" {
		pdf.CellFormat(0, theme.h(6), strings.TrimSpace(member.Plz+" "+member.Ort), "", 1, "L", false, 0, "")
	}
	if member.Email != "" {
		pdf.CellFormat(0, theme.h(6), member.Email, "", 1, "L", false, 0, "")
	}
	theme.ln(pdf, 6)

	// ── Invoice number / date, right-aligned ─────────────────────────────────
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(6), "Rechnungsnummer: "+invoiceNr, "", 1, "R", false, 0, "")
	pdf.CellFormat(0, theme.h(6), "Rechnungsdatum: "+inv.CreatedAt.Format("02.01.2006"), "", 1, "R", false, 0, "")
	theme.ln(pdf, 4)

	// ── Title ─────────────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "B", theme.size(3))
	pdf.CellFormat(0, theme.h(8), "Rechnung - "+eeg.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(6), fmt.Sprintf("Abrechnungszeitraum: %s – %s", inv.PeriodStart.Format("02.01.2006"), inv.PeriodEnd.Format("02.01.2006")), "", 1, "L", false, 0, "")
	theme.ln(pdf, 4)

	// ── Pre-text (optional) ──────────────────────────────────────────────────
	if eeg.InvoicePreText != "" {
		pdf.SetFont("Theme", "", theme.size(0))
		pdf.MultiCell(0, theme.h(6), eeg.InvoicePreText, "", "L", false)
		theme.ln(pdf, 4)
	}

	// ── Energy breakdown table (Netzbezug/Community-Verbrauch, see EnergyPeriodRow) ─
	drawEnergyPeriodTable(pdf, energyRows, theme, eeg)

	// ── Generation breakdown table (Gesamteinspeisung/Abnahme/Resteinspeisung) ─
	drawGenerationPeriodTable(pdf, generationRows, theme, eeg)

	// ── Pricing table: same content/branches as GeneratePDF, restyled ────────
	colDesc := 80.0
	colKwh := 30.0
	colPrice := 40.0
	colAmount := 0.0
	rowH := theme.h(8.0)
	vatH := theme.h(6.0)

	// Banner title row spanning full width — matches printableWidth exactly so it
	// aligns with the column row below (colAmount=0 auto-fills to the same edge).
	pdf.SetFont("Theme", "B", theme.size(0))
	theme.apply(pdf)
	pdf.CellFormat(printableWidth(pdf), rowH, "Monatsabrechnung", "1", 1, "C", true, 0, "")

	pdf.SetFont("Theme", "B", theme.size(0))
	theme.apply(pdf)
	pdf.CellFormat(colDesc, rowH, "Beschreibung", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colKwh, rowH, "kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colPrice, rowH, "Preis je kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colAmount, rowH, "Betrag", "1", 1, "R", true, 0, "")

	pdf.SetFont("Theme", "", theme.size(0))
	pdf.SetFillColor(255, 255, 255)

	multiMonth := len(vat.MonthlyLineItems) > 1
	if multiMonth {
		totalConsKwh := 0.0
		totalConsAmount := 0.0
		for _, m := range vat.MonthlyLineItems {
			if m.ConsumptionKwh == 0 && vat.GenerationKwh > 0 {
				continue
			}
			monthLabel := germanMonth(m.Month.Month()) + " " + fmt.Sprintf("%d", m.Month.Year())
			priceCt := m.EnergyPriceCt
			if priceCt == 0 {
				priceCt = vat.EnergyPrice
			}
			energyAmount := m.ConsumptionKwh * priceCt / 100
			totalConsKwh += m.ConsumptionKwh
			totalConsAmount += energyAmount
			drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
				"Bezug Strom "+monthLabel, formatKwh(m.ConsumptionKwh), fmt.Sprintf("%.4f ct", priceCt), formatAmount(energyAmount))
		}
		if vat.ConsumptionKwh > 0 || vat.GenerationKwh == 0 {
			drawTotalRow(pdf, colDesc, colKwh, colPrice, colAmount, rowH, "Summe Bezug", totalConsKwh, totalConsAmount, false)
			drawMpSubRow(pdf, vat.ConsumptionMeterPointKwh)
		}
		totalGenKwh := 0.0
		totalGenAmount := 0.0
		for _, m := range vat.MonthlyLineItems {
			if m.GenerationKwh == 0 {
				continue
			}
			monthLabel := germanMonth(m.Month.Month()) + " " + fmt.Sprintf("%d", m.Month.Year())
			prodPriceCt := m.ProducerPriceCt
			if prodPriceCt == 0 {
				prodPriceCt = vat.ProducerPrice
			}
			genAmount := m.GenerationKwh * prodPriceCt / 100
			totalGenKwh += m.GenerationKwh
			totalGenAmount += genAmount
			drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
				"Einspeisung "+monthLabel, formatKwh(m.GenerationKwh), fmt.Sprintf("%.4f ct", prodPriceCt), "-"+formatAmount(genAmount))
		}
		if vat.GenerationKwh > 0 {
			drawTotalRow(pdf, colDesc, colKwh, colPrice, colAmount, rowH, "Summe Einspeisung", totalGenKwh, totalGenAmount, true)
			drawMpSubRow(pdf, vat.GenerationMeterPointKwh)
		}
		feeMonths := vat.FeeMonths
		if feeMonths < 1 {
			feeMonths = 1
		}
		feeTotal := (vat.MeterFeeEur + vat.ParticipationFeeEur) * float64(feeMonths)
		feeLabel := "Messstellengebühr / Teilnahmegebühr"
		if feeMonths > 1 {
			feeLabel = fmt.Sprintf("Messstellengebühr / Teilnahmegebühr (%d Monate)", feeMonths)
		}
		if feeTotal > 0 || eeg.InvoiceShowZeroFees {
			drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH, feeLabel, "", "", formatAmount(feeTotal))
		}
	} else if vat.GenerationKwh > 0 {
		drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
			"Bezug Strom "+period, formatKwh(vat.ConsumptionKwh), fmt.Sprintf("%.4f ct", vat.EnergyPrice), formatAmount(vat.EnergyNet))
		drawMpSubRow(pdf, vat.ConsumptionMeterPointKwh)

		drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
			"Einspeisung Strom "+period, formatKwh(vat.GenerationKwh), fmt.Sprintf("%.4f ct", vat.ProducerPrice), "-"+formatAmount(vat.GenerationNet))
		drawMpSubRow(pdf, vat.GenerationMeterPointKwh)
	} else {
		drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
			"Bezug Strom "+period, formatKwh(inv.ConsumptionKwh), fmt.Sprintf("%.4f ct", vat.EnergyPrice), formatAmount(vat.EnergyNet))
		drawMpSubRow(pdf, vat.ConsumptionMeterPointKwh)
	}

	if vat.ZaehlpunktsGebuehrTotal > 0 || eeg.InvoiceShowZeroFees {
		label := fmt.Sprintf("Zählpunktsgebühr (%d × %s)", vat.ZaehlpunktsGebuehrCount, formatAmount(vat.ZaehlpunktsGebuehrEur))
		if vat.FeeMonths > 1 {
			label = fmt.Sprintf("Zählpunktsgebühr (%d × %s × %d Monate)", vat.ZaehlpunktsGebuehrCount, formatAmount(vat.ZaehlpunktsGebuehrEur), vat.FeeMonths)
		}
		drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH, label, "", "", formatAmount(vat.ZaehlpunktsGebuehrTotal))
	}

	// ── VAT breakdown ─────────────────────────────────────────────────────────
	theme.ln(pdf, 1)
	pdf.SetFont("Theme", "", theme.size(0))

	if vat.GenerationKwh > 0 {
		pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Nettobetrag Bezug", "0", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, vatH, formatAmount(vat.ConsumptionNet), "0", 1, "R", false, 0, "")
		if vat.ConsumptionVatPct > 0 {
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, fmt.Sprintf("USt. (%.0f %%) auf Bezug", vat.ConsumptionVatPct), "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(vat.ConsumptionVatAmount), "0", 1, "R", false, 0, "")
		} else {
			if vat.UseVat {
				pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "USt. (0 %) auf Bezug", "0", 0, "R", false, 0, "")
			} else {
				pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "USt. (0 %, steuerbefreit gem. § 6 Abs. 1 Z 27 UStG)", "0", 0, "R", false, 0, "")
			}
			pdf.CellFormat(colAmount, vatH, "0,00 €", "0", 1, "R", false, 0, "")
		}
		pdf.SetFont("Theme", "B", theme.size(0))
		pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Bruttobetrag Bezug", "0", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, vatH, formatAmount(vat.ConsumptionGross), "0", 1, "R", false, 0, "")
		pdf.SetFont("Theme", "", theme.size(0))

		theme.ln(pdf, 1)

		pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Nettobetrag Einspeisung", "0", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationNet), "0", 1, "R", false, 0, "")
		if vat.GenerationVatPct > 0 {
			if vat.GenerationReverseCharge {
				pdf.CellFormat(colDesc+colKwh+colPrice, vatH, fmt.Sprintf("USt. (%.0f %%), Reverse Charge § 2 Z 2 UStBBKV", vat.GenerationVatPct), "0", 0, "R", false, 0, "")
				pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationVatAmount), "0", 1, "R", false, 0, "")
			} else {
				label := vat.GenerationVatText
				if label == "" {
					label = fmt.Sprintf("USt. (%.0f %%) auf Einspeisung", vat.GenerationVatPct)
				}
				pdf.CellFormat(colDesc+colKwh+colPrice, vatH, label, "0", 0, "R", false, 0, "")
				pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationVatAmount), "0", 1, "R", false, 0, "")
			}
			pdf.SetFont("Theme", "B", theme.size(0))
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Bruttobetrag Einspeisung", "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationGross), "0", 1, "R", false, 0, "")
			pdf.SetFont("Theme", "", theme.size(0))
		} else if vat.GenerationVatText != "" {
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, vat.GenerationVatText, "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, "0,00 €", "0", 1, "R", false, 0, "")
			pdf.SetFont("Theme", "B", theme.size(0))
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Bruttobetrag Einspeisung", "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationGross), "0", 1, "R", false, 0, "")
			pdf.SetFont("Theme", "", theme.size(0))
		}

		theme.ln(pdf, 2)
		drawThemedSaldoRow(pdf, colDesc+colKwh+colPrice+colAmount, inv, vat, theme)
	} else {
		pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Nettobetrag Bezug", "0", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, vatH, formatAmount(vat.ConsumptionNet), "0", 1, "R", false, 0, "")
		if vat.ConsumptionVatPct > 0 {
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, fmt.Sprintf("USt. (%.0f %%)", vat.ConsumptionVatPct), "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(vat.ConsumptionVatAmount), "0", 1, "R", false, 0, "")
		} else {
			if vat.UseVat {
				pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "USt. (0 %)", "0", 0, "R", false, 0, "")
			} else {
				pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "USt. (0 %, steuerbefreit gem. § 6 Abs. 1 Z 27 UStG)", "0", 0, "R", false, 0, "")
			}
			pdf.CellFormat(colAmount, vatH, "0,00 €", "0", 1, "R", false, 0, "")
		}
		theme.ln(pdf, 2)
		drawThemedTotalLine(pdf, fmt.Sprintf("Bruttobetrag Bezug: %s", formatAmount(inv.TotalAmount)), theme)
	}

	theme.ln(pdf, 4)

	// ── Payment notice ───────────────────────────────────────────────────────
	if eeg.InvoicePaymentNoticeMode != "none" {
		pdf.SetFont("Theme", "B", theme.size(0))
		pdf.SetTextColor(40, 40, 40)
		pdf.CellFormat(0, theme.h(5), "Zahlungshinweis", "", 1, "L", false, 0, "")
		pdf.SetFont("Theme", "", theme.size(0))
		pdf.SetTextColor(0, 0, 0)
		var notice string
		zahlBetrag := inv.TotalAmount
		if vat.GenerationReverseCharge {
			zahlBetrag += inv.GenerationVatAmount
		}
		if eeg.InvoicePaymentNoticeMode == "custom" {
			noticeDays := eeg.SepaPreNotificationDays
			if noticeDays <= 0 {
				noticeDays = 14
			}
			datum := inv.CreatedAt.AddDate(0, 0, noticeDays)
			notice = renderPaymentNoticeTemplate(eeg.InvoicePaymentNoticeText, math.Abs(zahlBetrag), member.IBAN, eeg.IBAN, eeg.BIC, datum)
		} else if zahlBetrag < 0 {
			credit := formatAmount(-zahlBetrag)
			noticeDays := eeg.SepaPreNotificationDays
			if noticeDays <= 0 {
				noticeDays = 14
			}
			transferDate := pdfNextWorkday(inv.CreatedAt.AddDate(0, 0, noticeDays))
			transferDateStr := transferDate.Format("02.01.2006")
			if member.IBAN != "" {
				notice = fmt.Sprintf("Der Guthabenbetrag von %s wird bis zum %s automatisch auf Ihr Konto (IBAN: %s) überwiesen.", credit, transferDateStr, member.IBAN)
			} else {
				notice = fmt.Sprintf("Der Guthabenbetrag von %s wird bis zum %s automatisch auf Ihr Konto überwiesen.", credit, transferDateStr)
			}
		} else if zahlBetrag < 0.005 {
			notice = "Der Rechnungsbetrag beträgt 0,00 €. Es wird kein Betrag eingezogen."
		} else if eeg.InvoicePaymentNoticeMode == "ueberweisung" {
			noticeDays := eeg.SepaPreNotificationDays
			if noticeDays <= 0 {
				noticeDays = 14
			}
			dueDate := inv.CreatedAt.AddDate(0, 0, noticeDays)
			dueDateStr := dueDate.Format("02.01.2006")
			notice = fmt.Sprintf("Bitte überweisen Sie den Rechnungsbetrag von %s bis zum %s auf folgendes Konto: IBAN: %s", formatAmount(zahlBetrag), dueDateStr, eeg.IBAN)
			if eeg.BIC != "" {
				notice += fmt.Sprintf(" (BIC: %s)", eeg.BIC)
			}
		} else {
			noticeDays := eeg.SepaPreNotificationDays
			if noticeDays <= 0 {
				noticeDays = 14
			}
			collectionDate := inv.CreatedAt.AddDate(0, 0, noticeDays)
			collectionDateStr := collectionDate.Format("02.01.2006")
			if member.IBAN != "" {
				notice = fmt.Sprintf("Der Rechnungsbetrag von %s wird per SEPA-Lastschrift von Ihrem Konto (IBAN: %s) eingezogen. Der Betrag wird frühestens am %s fällig.", formatAmount(zahlBetrag), member.IBAN, collectionDateStr)
			} else {
				notice = fmt.Sprintf("Der Rechnungsbetrag von %s wird per SEPA-Lastschrift von Ihrem Konto eingezogen. Der Betrag wird frühestens am %s fällig.", formatAmount(zahlBetrag), collectionDateStr)
			}
		}
		pdf.MultiCell(0, theme.h(5), notice, "", "L", false)
	}

	if vat.GenerationReverseCharge && inv.GenerationVatAmount > 0 {
		theme.ln(pdf, 3)
		pdf.SetFont("Theme", "", theme.size(-1))
		pdf.SetTextColor(80, 80, 80)
		rcNote := fmt.Sprintf(
			"Die Umsatzsteuer auf die Einspeisevergütung in Höhe von %s wird gemäß § 2 Z 2 UStBBKV (Reverse Charge) von der Energiegemeinschaft selbst berechnet und an das Finanzamt abgeführt.",
			formatAmount(inv.GenerationVatAmount),
		)
		pdf.MultiCell(0, theme.h(5), rcNote, "", "L", false)
		pdf.SetFont("Theme", "", theme.size(0))
		pdf.SetTextColor(0, 0, 0)
	}

	theme.ln(pdf, 4)

	if len(history) > 0 {
		drawBarChart(pdf, history)
		theme.ln(pdf, 2)
	}

	if eeg.InvoicePostText != "" {
		pdf.SetFont("Theme", "", theme.size(0))
		pdf.MultiCell(0, theme.h(6), eeg.InvoicePostText, "", "L", false)
		theme.ln(pdf, 4)
	}

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "", theme.size(-2))
	pdf.SetTextColor(128, 128, 128)
	footerText := "Erstellt von eegabrechnung"
	if eeg.InvoiceFooterText != "" {
		footerText = eeg.InvoiceFooterText
	}
	pdf.CellFormat(0, theme.h(6), footerText, "", 1, "C", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("pdf generation error: %w", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

// drawThemedSaldoRow renders the bold Saldo line with a top rule instead of the
// full-box style used by GeneratePDF's drawTotalRow — matches the reference
// PDF's "Gesamtkosten inkl. USt." row.
func drawThemedSaldoRow(pdf *fpdf.Fpdf, width float64, inv *domain.Invoice, vat VATOptions, theme InvoiceTheme) {
	saldoBetrag := inv.TotalAmount
	if vat.GenerationReverseCharge {
		saldoBetrag += inv.GenerationVatAmount
	}
	var totalLabel string
	if saldoBetrag < 0 {
		totalLabel = fmt.Sprintf("Saldo (EEG zahlt an Sie): %s", formatAmount(-saldoBetrag))
	} else {
		totalLabel = fmt.Sprintf("Saldo (Zahlungsbetrag): %s", formatAmount(saldoBetrag))
	}
	drawThemedTotalLine(pdf, totalLabel, theme)
}

func drawThemedTotalLine(pdf *fpdf.Fpdf, label string, theme InvoiceTheme) {
	x, y := pdf.GetXY()
	lMargin, _, rMargin, _ := pdf.GetMargins()
	pw, _ := pdf.GetPageSize()
	lineW := pw - lMargin - rMargin
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.4)
	pdf.Line(x, y, x+lineW, y)
	theme.ln(pdf, 1)
	pdf.SetFont("Theme", "B", theme.size(1))
	pdf.CellFormat(0, theme.h(8), label, "", 1, "R", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
}

// GenerateCreditNotePDFThemed renders a producer "Gutschrift" with the Oikos-style
// visual theme — mirrors GenerateCreditNotePDF's data logic (multi-month/single-
// month branches, VAT/Reverse-Charge handling) exactly, restyling header, logo,
// and pricing table the same way GeneratePDFThemed does. No energy breakdown
// table: Netzbezug is a consumption concept, not applicable to a producer credit
// note.
func GenerateCreditNotePDFThemed(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, producerPriceCt, generationKwh float64, generationMeterPoints []MeterPointKwh, monthlyItems []MonthlyKwh, history []MonthlyKwh, theme InvoiceTheme) ([]byte, error) {
	pdf := newThemedPDF(theme)
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	period := periodLabel(inv.PeriodStart)

	creditNoteNr := shortID(inv.ID.String())
	if inv.InvoiceNumber != nil {
		digits := eeg.CreditNoteNumberDigits
		if digits <= 0 {
			digits = 5
		}
		creditNoteNr = fmt.Sprintf("%s%0*d", eeg.CreditNoteNumberPrefix, digits, *inv.InvoiceNumber)
	}

	// ── Header: logo + EEG address block side by side ────────────────────────
	logoX, addrX, addrW := 150.0, 20.0, 90.0
	if theme.LogoLeft {
		logoX, addrX, addrW = 20.0, 130.0, 60.0
	}
	embedLogoAt(pdf, eeg.LogoPath, logoX, 15, 0, addressBlockHeight(eeg, theme))

	pdf.SetXY(addrX, 15)
	pdf.SetFont("Theme", "B", theme.size(1))
	pdf.CellFormat(addrW, theme.h(5), eeg.Name, "", 2, "R", false, 0, "")
	pdf.SetX(addrX)
	pdf.SetFont("Theme", "", theme.size(-1))
	if eeg.Strasse != "" {
		pdf.CellFormat(addrW, theme.h(5), eeg.Strasse, "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(addrW, theme.h(5), strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(addrW, theme.h(5), "UID-Nr.: "+eeg.UidNummer, "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}

	pdf.SetXY(20, 45)

	// ── Recipient block ──────────────────────────────────────────────────────
	pdf.SetFont("Theme", "", theme.size(0))
	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	pdf.CellFormat(0, theme.h(6), fullName, "", 1, "L", false, 0, "")
	if member.Strasse != "" {
		pdf.CellFormat(0, theme.h(6), member.Strasse, "", 1, "L", false, 0, "")
	}
	if member.Plz != "" || member.Ort != "" {
		pdf.CellFormat(0, theme.h(6), strings.TrimSpace(member.Plz+" "+member.Ort), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, theme.h(6), "Mitgliedsnummer: "+member.MitgliedsNr, "", 1, "L", false, 0, "")
	if member.UidNummer != "" {
		pdf.CellFormat(0, theme.h(6), "UID-Nummer: "+member.UidNummer, "", 1, "L", false, 0, "")
	}
	theme.ln(pdf, 6)

	// ── Gutschrift number / date, right-aligned ──────────────────────────────
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(6), "Gutschrift-Nr.: "+creditNoteNr, "", 1, "R", false, 0, "")
	pdf.CellFormat(0, theme.h(6), "Datum: "+inv.CreatedAt.Format("02.01.2006"), "", 1, "R", false, 0, "")
	theme.ln(pdf, 4)

	// ── Title ─────────────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "B", theme.size(3))
	pdf.CellFormat(0, theme.h(8), "Gutschrift - "+eeg.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(6), fmt.Sprintf("Abrechnungszeitraum: %s – %s", inv.PeriodStart.Format("02.01.2006"), inv.PeriodEnd.Format("02.01.2006")), "", 1, "L", false, 0, "")
	theme.ln(pdf, 4)

	// ── Pre-text ────────────────────────────────────────────────────────────
	if eeg.InvoicePreText != "" {
		pdf.SetFont("Theme", "", theme.size(0))
		pdf.MultiCell(0, theme.h(6), eeg.InvoicePreText, "", "L", false)
		theme.ln(pdf, 4)
	}

	// ── Pricing table: same content/branches as GenerateCreditNotePDF, restyled ─
	colDesc := 80.0
	colKwh := 30.0
	colPrice := 40.0
	colAmount := 0.0
	rowH := theme.h(8.0)
	vatH := theme.h(6.0)

	pdf.SetFont("Theme", "B", theme.size(0))
	theme.apply(pdf)
	pdf.CellFormat(printableWidth(pdf), rowH, "Gutschriftsabrechnung", "1", 1, "C", true, 0, "")

	pdf.SetFont("Theme", "B", theme.size(0))
	theme.apply(pdf)
	pdf.CellFormat(colDesc, rowH, "Beschreibung", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colKwh, rowH, "kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colPrice, rowH, "Tarif je kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colAmount, rowH, "Betrag", "1", 1, "R", true, 0, "")

	pdf.SetFont("Theme", "", theme.size(0))
	pdf.SetFillColor(255, 255, 255)
	netAmount := generationKwh * producerPriceCt / 100

	if len(monthlyItems) > 1 {
		totalGenKwh := 0.0
		totalGenAmount := 0.0
		for _, m := range monthlyItems {
			if m.GenerationKwh == 0 {
				continue
			}
			monthLabel := germanMonth(m.Month.Month()) + " " + fmt.Sprintf("%d", m.Month.Year())
			mPriceCt := m.ProducerPriceCt
			if mPriceCt == 0 {
				mPriceCt = producerPriceCt
			}
			genAmount := m.GenerationKwh * mPriceCt / 100
			totalGenKwh += m.GenerationKwh
			totalGenAmount += genAmount
			drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
				"Einspeisung Strom "+monthLabel, formatKwh(m.GenerationKwh), fmt.Sprintf("%.4f ct", mPriceCt), formatAmount(genAmount))
		}
		drawTotalRow(pdf, colDesc, colKwh, colPrice, colAmount, rowH, "Summe Einspeisung", totalGenKwh, totalGenAmount, false)
		drawMpSubRow(pdf, generationMeterPoints)
	} else {
		drawWrappingLineRow(pdf, theme, colDesc, colKwh, colPrice, colAmount, rowH,
			"Einspeisung Strom "+period, formatKwh(generationKwh), fmt.Sprintf("%.4f ct", producerPriceCt), formatAmount(netAmount))
		drawMpSubRow(pdf, generationMeterPoints)
	}

	// ── VAT section ──────────────────────────────────────────────────────────
	vatText := GenerationVATText(member)
	genVatPct := inv.GenerationVatPct
	genVatAmount := inv.GenerationVatAmount
	genRC := GenerationReverseCharge(member)
	totalDisplay := netAmount + genVatAmount
	payDisplay := totalDisplay
	if genRC {
		payDisplay = netAmount
	}

	theme.ln(pdf, 1)
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Nettobetrag", "0", 0, "R", false, 0, "")
	pdf.CellFormat(colAmount, vatH, formatAmount(netAmount), "0", 1, "R", false, 0, "")

	if genVatPct > 0 {
		if genRC {
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, fmt.Sprintf("USt. (%.0f %%), Reverse Charge § 2 Z 2 UStBBKV", genVatPct), "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(genVatAmount), "0", 1, "R", false, 0, "")
		} else {
			label := vatText
			if label == "" {
				label = fmt.Sprintf("USt. (%.0f %%) auf Einspeisung", genVatPct)
			}
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, label, "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(genVatAmount), "0", 1, "R", false, 0, "")
		}
	} else if vatText != "" {
		pdf.CellFormat(colDesc+colKwh+colPrice, vatH, vatText, "0", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, vatH, "0,00 €", "0", 1, "R", false, 0, "")
	}

	theme.ln(pdf, 2)
	drawThemedTotalLine(pdf, fmt.Sprintf("Gutschriftbetrag: %s", formatAmount(totalDisplay)), theme)
	theme.ln(pdf, 2)

	// ── Payment notice ───────────────────────────────────────────────────────
	if eeg.InvoicePaymentNoticeMode != "none" {
		pdf.SetFont("Theme", "B", theme.size(0))
		pdf.SetTextColor(40, 40, 40)
		pdf.CellFormat(0, theme.h(5), "Auszahlung", "", 1, "L", false, 0, "")
		pdf.SetFont("Theme", "", theme.size(0))
		pdf.SetTextColor(0, 0, 0)
		var creditNotice string
		if eeg.InvoicePaymentNoticeMode == "custom" {
			creditNotice = renderPaymentNoticeTemplate(eeg.InvoicePaymentNoticeText, payDisplay, member.IBAN, eeg.IBAN, eeg.BIC, time.Time{})
		} else {
			creditNotice = fmt.Sprintf("Der Gutschriftbetrag von %s wird automatisch auf Ihr Konto überwiesen.", formatAmount(payDisplay))
			if member.IBAN != "" {
				creditNotice = fmt.Sprintf("Der Gutschriftbetrag von %s wird automatisch auf Ihr Konto (IBAN: %s) überwiesen.", formatAmount(payDisplay), member.IBAN)
			}
		}
		pdf.MultiCell(0, theme.h(5), creditNotice, "", "L", false)
		theme.ln(pdf, 4)
	}

	if len(history) > 0 {
		drawBarChart(pdf, history)
		theme.ln(pdf, 2)
	}

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "", theme.size(-2))
	pdf.SetTextColor(128, 128, 128)
	footerText := "Erstellt von eegabrechnung"
	if eeg.InvoiceFooterText != "" {
		footerText = eeg.InvoiceFooterText
	}
	pdf.CellFormat(0, theme.h(6), footerText, "", 1, "C", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("pdf generation error: %w", err)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateStornorechnungThemed renders a Storno document with the Oikos-style
// visual theme — mirrors GenerateStornorechnung's data logic exactly (it has no
// pricing/energy table to restyle, just header/logo/notice-box/total).
func GenerateStornorechnungThemed(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, theme InvoiceTheme) ([]byte, error) {
	pdf := newThemedPDF(theme)
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	origNr := shortID(inv.ID.String())
	if inv.InvoiceNumber != nil {
		if inv.DocumentType == "credit_note" {
			digits := eeg.CreditNoteNumberDigits
			if digits <= 0 {
				digits = 5
			}
			origNr = fmt.Sprintf("%s%0*d", eeg.CreditNoteNumberPrefix, digits, *inv.InvoiceNumber)
		} else {
			digits := eeg.InvoiceNumberDigits
			if digits <= 0 {
				digits = 4
			}
			origNr = fmt.Sprintf("%s%0*d", eeg.InvoiceNumberPrefix, digits, *inv.InvoiceNumber)
		}
	}

	// ── Header: logo + EEG address block side by side ────────────────────────
	logoX, addrX, addrW := 150.0, 20.0, 90.0
	if theme.LogoLeft {
		logoX, addrX, addrW = 20.0, 130.0, 60.0
	}
	embedLogoAt(pdf, eeg.LogoPath, logoX, 15, 0, addressBlockHeight(eeg, theme))

	pdf.SetXY(addrX, 15)
	pdf.SetFont("Theme", "B", theme.size(1))
	pdf.CellFormat(addrW, theme.h(5), eeg.Name, "", 2, "R", false, 0, "")
	pdf.SetX(addrX)
	pdf.SetFont("Theme", "", theme.size(-1))
	if eeg.Strasse != "" {
		pdf.CellFormat(addrW, theme.h(5), eeg.Strasse, "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(addrW, theme.h(5), strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(addrW, theme.h(5), "UID-Nr.: "+eeg.UidNummer, "", 2, "R", false, 0, "")
		pdf.SetX(addrX)
	}

	pdf.SetXY(20, 45)

	// ── Title ─────────────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "B", theme.size(3))
	pdf.CellFormat(0, theme.h(8), "Stornorechnung - "+eeg.Name, "", 1, "L", false, 0, "")
	theme.ln(pdf, 4)

	// ── Reference, date & billing period ─────────────────────────────────────
	pdf.SetFont("Theme", "B", theme.size(0))
	pdf.CellFormat(55, theme.h(7), "Storno zu Beleg:", "", 0, "L", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(7), origNr, "", 1, "L", false, 0, "")

	pdf.SetFont("Theme", "B", theme.size(0))
	pdf.CellFormat(55, theme.h(7), "Stornodatum:", "", 0, "L", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(7), time.Now().Format("02.01.2006"), "", 1, "L", false, 0, "")

	pdf.SetFont("Theme", "B", theme.size(0))
	pdf.CellFormat(55, theme.h(7), "Ursprungsdatum:", "", 0, "L", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(7), inv.CreatedAt.Format("02.01.2006"), "", 1, "L", false, 0, "")

	pdf.SetFont("Theme", "B", theme.size(0))
	pdf.CellFormat(55, theme.h(7), "Abrechnungszeitraum:", "", 0, "L", false, 0, "")
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(0, theme.h(7), fmt.Sprintf("%s – %s",
		inv.PeriodStart.Format("02.01.2006"),
		inv.PeriodEnd.Format("02.01.2006"),
	), "", 1, "L", false, 0, "")
	theme.ln(pdf, 6)

	// ── Member block ─────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "B", theme.size(0))
	if inv.DocumentType == "credit_note" {
		pdf.CellFormat(0, theme.h(7), "Storno Gutschrift an", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(0, theme.h(7), "Storno Rechnung an", "", 1, "L", false, 0, "")
	}
	pdf.SetFont("Theme", "", theme.size(0))
	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	pdf.CellFormat(0, theme.h(6), fullName, "", 1, "L", false, 0, "")
	if member.Strasse != "" {
		pdf.CellFormat(0, theme.h(6), member.Strasse, "", 1, "L", false, 0, "")
	}
	if member.Plz != "" || member.Ort != "" {
		pdf.CellFormat(0, theme.h(6), strings.TrimSpace(member.Plz+" "+member.Ort), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, theme.h(6), "Mitgliedsnummer: "+member.MitgliedsNr, "", 1, "L", false, 0, "")
	if member.UidNummer != "" {
		pdf.CellFormat(0, theme.h(6), "UID-Nummer: "+member.UidNummer, "", 1, "L", false, 0, "")
	}
	theme.ln(pdf, 8)

	// ── Storno notice — accent-tinted instead of the fixed yellow highlight ──
	pdf.SetFont("Theme", "", theme.size(0))
	pdf.SetFillColor(theme.AccentR, theme.AccentG, theme.AccentB)
	pdf.MultiCell(0, theme.h(6),
		fmt.Sprintf("Diese Stornorechnung hebt den Beleg %s vollständig auf. "+
			"Die nachfolgenden Beträge entsprechen den negativen Werten des Originalbelegs.", origNr),
		"1", "L", true,
	)
	pdf.SetFillColor(255, 255, 255)
	theme.ln(pdf, 4)

	// ── Amount table ─────────────────────────────────────────────────────────
	colLabel := 130.0
	colAmount := 0.0
	rowH := theme.h(8.0)

	pdf.SetFont("Theme", "", theme.size(0))
	pdf.CellFormat(colLabel, rowH, "Nettobetrag (storniert)", "0", 0, "L", false, 0, "")
	pdf.CellFormat(colAmount, rowH, formatAmount(-inv.NetAmount), "0", 1, "R", false, 0, "")

	if inv.VatPctApplied > 0 {
		pdf.CellFormat(colLabel, rowH, fmt.Sprintf("USt. (%.0f %%)", inv.VatPctApplied), "0", 0, "L", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(-inv.VatAmount), "0", 1, "R", false, 0, "")
	} else {
		pdf.CellFormat(colLabel, rowH, "USt. (0 %, steuerbefreit gem. § 6 UStG)", "0", 0, "L", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(0), "0", 1, "R", false, 0, "")
	}

	theme.ln(pdf, 4)
	drawThemedTotalLine(pdf, fmt.Sprintf("Stornobetrag: %s", formatAmount(-inv.TotalAmount)), theme)
	theme.ln(pdf, 6)

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetFont("Theme", "", theme.size(-2))
	pdf.SetTextColor(128, 128, 128)
	footerText := "Erstellt von eegabrechnung"
	if eeg.InvoiceFooterText != "" {
		footerText = eeg.InvoiceFooterText
	}
	pdf.CellFormat(0, theme.h(6), footerText, "", 1, "C", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("storno pdf generation error: %w", err)
	}
	var stornoBuf bytes.Buffer
	if err := pdf.Output(&stornoBuf); err != nil {
		return nil, fmt.Errorf("storno pdf output error: %w", err)
	}
	return stornoBuf.Bytes(), nil
}
