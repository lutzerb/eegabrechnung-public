package invoice

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// MonthlyKwh is an energy summary for one calendar month.
// EnergyPriceCt and ProducerPriceCt are set when used as invoice line items (multi-month billing)
// so each month can carry its own tariff price. Zero means "use period-level price".
//
// IMPORTANT: ConsumptionKwh/GenerationKwh here are already the COMMUNITY-COVERED
// amounts (wh_self / wh_community) — the same figures billed as "Bezug Strom"/
// "Einspeisung Strom" on the invoice — NOT the member's total physical
// consumption/feed-in. This matches MonthlySummaryForMember's query. See
// EnergyPeriodRow/GenerationPeriodRow for the analogous current-period fields
// where WhSelf/WhCommunity carry the same meaning.
//
// TotalConsumptionKwh/TotalGenerationKwh (wh_total) are only populated for the
// chart history fetch when the "individuell" design's percentage chart is
// selected (see billing.Service.chartHistory) — zero means "split not
// fetched", not "no consumption/generation that month". Netzbezug/
// Resteinspeisung are derived at draw time as TotalConsumptionKwh-ConsumptionKwh
// / TotalGenerationKwh-GenerationKwh.
type MonthlyKwh struct {
	Month               time.Time
	ConsumptionKwh      float64
	GenerationKwh       float64
	TotalConsumptionKwh float64
	TotalGenerationKwh  float64
	EnergyPriceCt       float64 // ct/kWh for this month's Bezug (0 = use period-level price)
	ProducerPriceCt     float64 // ct/kWh for this month's Einspeisung
}

// MeterPointKwh carries a meter point ID and its billed energy for the invoice period.
type MeterPointKwh struct {
	Zaehlpunkt string
	Kwh        float64
}

// ExtraMeterLine is one manually-read Zusatzzähler (submeter) billed on the invoice —
// the Kwh delta between two counter readings, priced at the same ct/kWh as the
// member's normal Bezug.
type ExtraMeterLine struct {
	Label   string
	Kwh     float64
	PriceCt float64
	Amount  float64
}

//go:embed fonts/DejaVuSans.ttf
var dejaVuSans []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var dejaVuSansBold []byte

// VATOptions carries per-invoice VAT details and breakdown data for PDF generation.
// Consumption (Bezug) and generation (Einspeisung) are treated independently:
//   - Consumption uses the EEG's own VAT status (UseVat / VatPct).
//   - Generation uses the member's individual VAT rule (§ 6, § 19, § 22 UStG).
type VATOptions struct {
	UseVat bool    // EEG is VAT-registered (Regelbesteuerung)
	VatPct float64 // EEG VAT rate (for consumption side)

	// Energy quantities and net prices
	ConsumptionKwh float64 // community kWh consumed (Bezug)
	GenerationKwh  float64 // community kWh generated/fed in (Einspeisung)
	ConsumptionNet float64 // pre-VAT consumption charge
	GenerationNet  float64 // pre-VAT generation credit (positive = credit amount)
	EnergyPrice    float64 // ct/kWh — Arbeitspreis
	ProducerPrice  float64 // ct/kWh — Einspeisetarif

	// Consumption VAT (EEG-level)
	ConsumptionVatPct    float64
	ConsumptionVatAmount float64
	ConsumptionGross     float64 // ConsumptionNet + ConsumptionVatAmount

	// Generation VAT (member-level)
	GenerationVatPct    float64
	GenerationVatAmount float64
	GenerationGross     float64 // GenerationNet + GenerationVatAmount
	GenerationVatText   string  // legal VAT notice text for generation side

	// GenerationReverseCharge: true when § 19 Abs. 1d UStG i.V.m. § 2 Z 2 UStBBKV applies to the generation side.
	// This happens when a VAT-registered Unternehmen delivers electricity to the EEG.
	// The EEG is the tax debtor; 20 % VAT is shown on the invoice and remitted by the EEG.
	GenerationReverseCharge bool

	// ConsumptionReverseCharge: true when § 19 Abs. 1d UStG i.V.m. § 2 Z 2 UStBBKV applies to the consumption side.
	// This happens when a Regelbesteuerung EEG delivers electricity to a VAT-registered Unternehmen.
	// The 20% VAT is still shown and remitted by the EEG, but the invoice must note that
	// the tax debt passes to the recipient (Steuerschuldübergang).
	ConsumptionReverseCharge bool

	// Meter points with per-ZP billed energy for invoice imprint.
	ConsumptionMeterPointKwh []MeterPointKwh
	GenerationMeterPointKwh  []MeterPointKwh

	// Monthly breakdown — populated when the billing period spans more than one month.
	// ConsumptionKwh is scaled to effectiveConsumption (post free-kWh/discount) so that
	// sum(monthly × EnergyPrice) equals ConsumptionNet minus fixed fees.
	// GenerationKwh is scaled so months sum to the billed generation total.
	MonthlyLineItems []MonthlyKwh

	// Fixed fees (shown as a separate line in multi-month invoices).
	// MeterFeeEur/ParticipationFeeEur/ZaehlpunktsGebuehrEur are per-month unit values;
	// FeeMonths is the multiplier (per_month fee billing: started calendar months of
	// the period; per_invoice: always 1). ZaehlpunktsGebuehrTotal already includes it.
	MeterFeeEur         float64
	ParticipationFeeEur float64
	FeeMonths           int

	// EnergyNet is the energy-only Bezug amount (excludes all fixed fees below) — used for the
	// "Bezug Strom" line item so it doesn't silently absorb fixed fees.
	EnergyNet float64

	// Zählpunktsgebühr (fee × number of active meter points) — always shown as its own line,
	// unlike MeterFeeEur/ParticipationFeeEur above which only get a separate line for
	// multi-month invoices.
	ZaehlpunktsGebuehrEur   float64
	ZaehlpunktsGebuehrCount int
	ZaehlpunktsGebuehrTotal float64

	// Zusatzzähler (manually-read submeters, e.g. Wärmepumpe) — one line each, only
	// shown when non-empty. Already folded into ConsumptionNet/NetAmount.
	ExtraMeterLines []ExtraMeterLine

	// Servicegebühr pro kWh Bezug/Einspeisung — optional per-kWh fee, separate from
	// EnergyPrice/ProducerPrice, always shown as its own line (like Zählpunktsgebühr
	// above). Both totals are already folded into ConsumptionNet: the Einspeisung fee
	// too, so it carries the EEG-level consumption VAT rather than the member-specific
	// generation VAT — see domain.EEG.ServicegebuehrEinspeisungCtKwh's doc comment.
	ServiceFeeBezugKwh         float64
	ServiceFeeBezugCtKwh       float64
	ServiceFeeBezugTotal       float64
	ServiceFeeEinspeisungKwh   float64
	ServiceFeeEinspeisungCtKwh float64
	ServiceFeeEinspeisungTotal float64

	// WerbebonusTotal (Migration 114) — flat "Mitglieder werben Mitglieder"
	// referral bonus, manually granted by an admin. Net amount (pre-VAT), already
	// folded into ConsumptionNet before this struct is built — shown here purely
	// as its own negative-signed row so the item table still reconciles to
	// ConsumptionNet (EnergyNet + fees − WerbebonusTotal). Only ever set on the
	// Bezug side: a pure-producer Gutschrift with a referral bonus is rendered by
	// GenerateCreditNotePDF/Themed instead, which take the bonus as a separate
	// parameter (never routed through VATOptions).
	WerbebonusTotal float64
}

// germanMonth returns the German name for a month (no umlauts needed — all ASCII here).
func germanMonth(m time.Month) string {
	months := map[time.Month]string{
		time.January:   "Jänner",
		time.February:  "Februar",
		time.March:     "März",
		time.April:     "April",
		time.May:       "Mai",
		time.June:      "Juni",
		time.July:      "Juli",
		time.August:    "August",
		time.September: "September",
		time.October:   "Oktober",
		time.November:  "November",
		time.December:  "Dezember",
	}
	if s, ok := months[m]; ok {
		return s
	}
	return m.String()
}

// periodLabel returns a human-readable period string like "Jänner 2026".
func periodLabel(periodStart time.Time) string {
	return fmt.Sprintf("%s %d", germanMonth(periodStart.Month()), periodStart.Year())
}

// periodSpansMultipleMonths reports whether an invoice's billing period covers
// more than one calendar month. This — not how many of those months happen to
// have billable data — is what decides the multi-month per-month-labeled table
// layout: a quarterly period with readings in only one calendar month (e.g. a
// producer just onboarded mid-quarter) must still label that row by its own
// month, not fall back to the single-month layout and mislabel it with the
// period's first month.
func periodSpansMultipleMonths(periodStart, periodEnd time.Time) bool {
	return periodStart.Year() != periodEnd.Year() || periodStart.Month() != periodEnd.Month()
}

// formatAmount formats a float as German locale currency string e.g. "1.234,56 €".
func formatAmount(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	decPart := parts[1]

	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}

	formatted := string(result) + "," + decPart
	if negative {
		formatted = "-" + formatted
	}
	return formatted + " €"
}

// renderPaymentNoticeTemplate substitutes {betrag}/{iban}/{eeg_iban}/{eeg_bic}/{datum}
// in a custom Zahlungshinweis/Auszahlung template (invoice_payment_notice_mode
// == "custom"), same {placeholder} convention as domain.EEG.OnboardingContractText's
// {iban}/{datum}. Pass a zero time.Time when there's no natural due/collection date
// for the caller (e.g. a Gutschrift payout) — {datum} is then substituted with "".
func renderPaymentNoticeTemplate(tmpl string, betrag float64, iban, eegIban, eegBic string, datum time.Time) string {
	text := tmpl
	text = strings.ReplaceAll(text, "{betrag}", formatAmount(betrag))
	text = strings.ReplaceAll(text, "{iban}", iban)
	text = strings.ReplaceAll(text, "{eeg_iban}", eegIban)
	text = strings.ReplaceAll(text, "{eeg_bic}", eegBic)
	datumStr := ""
	if !datum.IsZero() {
		datumStr = datum.Format("02.01.2006")
	}
	text = strings.ReplaceAll(text, "{datum}", datumStr)
	return text
}

// formatKwh formats kWh value with 3 decimal places and German decimal separator.
func formatKwh(v float64) string {
	s := fmt.Sprintf("%.3f", v)
	parts := strings.Split(s, ".")
	return parts[0] + "," + parts[1]
}

// shortID returns the first 8 characters of a UUID string.
func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// newPDF creates an fpdf instance with DejaVu Sans as the Unicode font.
func newPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("DejaVu", "", dejaVuSans)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", dejaVuSansBold)
	return pdf
}

// GenerationVATText returns the Austrian VAT notice text applicable to a generation
// credit (Einspeisung) for this member — used both in Gutschriften and in the
// generation section of combined invoices.
func GenerationVATText(member *domain.Member) string {
	switch member.BusinessRole {
	case "landwirt_pauschaliert":
		// § 22 UStG: Durchschnittssteuersatz 13 % — always applies, even if UID present
		return "Durchschnittssteuersatz 13 % gem. § 22 UStG"
	case "gemeinde_hoheitlich":
		return "USt. (0 %), steuerbefreit (hoheitliche Tätigkeit)"
	case "privat":
		return "USt. (0 %), kein Steuerausweis (Privatperson)"
	}
	// For all other roles: Reverse Charge if UID present, else Kleinunternehmer exemption
	if member.UidNummer != "" {
		return "Die Steuerschuld geht auf den Leistungsempfänger über (Reverse Charge gem. § 2 Z 2 UStBBKV)"
	}
	return "USt. (0 %), steuerbefreit gem. § 6 Abs. 1 Z 27 UStG"
}

// GenerationVATPct returns the VAT percentage applicable to the generation side.
//   - pauschalierter Landwirt (§ 22 UStG): 13 %
//   - Unternehmen / Gemeinde BgA with UID: 20 % (Reverse Charge — EEG is tax debtor)
//   - all others: 0 %
func GenerationVATPct(member *domain.Member) float64 {
	switch member.BusinessRole {
	case "landwirt_pauschaliert":
		return 13.0
	case "privat", "gemeinde_hoheitlich":
		return 0.0
	}
	if member.UidNummer != "" {
		return 20.0
	}
	return 0.0
}

// GenerationReverseCharge reports whether § 19 Abs. 1d UStG i.V.m. § 2 Z 2 UStBBKV (Reverse Charge) applies
// to the generation side for this member — i.e. the EEG is the tax debtor.
func GenerationReverseCharge(member *domain.Member) bool {
	if member.BusinessRole == "privat" || member.BusinessRole == "gemeinde_hoheitlich" || member.BusinessRole == "landwirt_pauschaliert" {
		return false
	}
	return member.UidNummer != ""
}

// shortMonth returns a 3-letter German month abbreviation.
func shortMonth(m time.Month) string {
	abbr := map[time.Month]string{
		time.January: "Jän", time.February: "Feb", time.March: "Mär",
		time.April: "Apr", time.May: "Mai", time.June: "Jun",
		time.July: "Jul", time.August: "Aug", time.September: "Sep",
		time.October: "Okt", time.November: "Nov", time.December: "Dez",
	}
	if s, ok := abbr[m]; ok {
		return s
	}
	return m.String()[:3]
}

// drawTotalRow renders a bold "Summe" row at the bottom of a multi-month kWh block,
// showing the cumulated kWh and amount across all months of the invoice period.
func drawTotalRow(pdf *fpdf.Fpdf, colDesc, colKwh, colPrice, colAmount, rowH float64, label string, totalKwh, totalAmount float64, negative bool) {
	amountStr := formatAmount(totalAmount)
	if negative {
		amountStr = "-" + amountStr
	}
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(colDesc, rowH, label, "1", 0, "L", false, 0, "")
	pdf.CellFormat(colKwh, rowH, formatKwh(totalKwh), "1", 0, "R", false, 0, "")
	pdf.CellFormat(colPrice, rowH, "", "1", 0, "R", false, 0, "")
	pdf.CellFormat(colAmount, rowH, amountStr, "1", 1, "R", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
}

// drawMpSubRow renders a small gray sub-row inside the invoice line table showing
// the meter point ZP identifiers. When len(mps) > 1 each ZP gets its own line with
// the per-ZP kWh shown in the kWh column.
func drawMpSubRow(pdf *fpdf.Fpdf, mps []MeterPointKwh) {
	const (
		colDesc  = 80.0
		colKwh   = 30.0
		colPrice = 40.0
		subH     = 4.5
	)
	if len(mps) == 0 {
		return
	}
	pdf.SetFont("DejaVu", "", 7.5)
	pdf.SetTextColor(120, 120, 120)
	if len(mps) == 1 {
		pdf.CellFormat(colDesc+colKwh+colPrice, subH, "ZP: "+mps[0].Zaehlpunkt, "LB", 0, "L", false, 0, "")
		pdf.CellFormat(0, subH, "", "RB", 1, "", false, 0, "")
	} else {
		for i, mp := range mps {
			isLast := i == len(mps)-1
			lBorder, bBorder, rBorder := "L", "", "R"
			if isLast {
				lBorder, bBorder, rBorder = "LB", "B", "RB"
			}
			pdf.CellFormat(colDesc, subH, "  "+mp.Zaehlpunkt, lBorder, 0, "L", false, 0, "")
			pdf.CellFormat(colKwh, subH, formatKwh(mp.Kwh), bBorder, 0, "R", false, 0, "")
			pdf.CellFormat(colPrice, subH, "", bBorder, 0, "", false, 0, "")
			pdf.CellFormat(0, subH, "", rBorder, 1, "", false, 0, "")
		}
	}
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(0, 0, 0)
}

// drawBarChart draws a grouped bar chart showing monthly consumption (blue) and/or
// generation (green) kWh for the last N months.  It advances the PDF cursor below
// the chart area.  If data is empty, nothing is drawn.
func drawBarChart(pdf *fpdf.Fpdf, data []MonthlyKwh) {
	if len(data) == 0 {
		return
	}

	// Ensure enough vertical space — add a page if < 52mm remain
	_, pageH := pdf.GetPageSize()
	_, _, mBottom, _ := pdf.GetMargins()
	bottomBound := pageH - mBottom
	if pdf.GetY()+60 > bottomBound {
		pdf.AddPage()
	}

	// Determine whether we have both consumption and generation data
	hasBoth := false
	for _, d := range data {
		if d.ConsumptionKwh > 0 && d.GenerationKwh > 0 {
			hasBoth = true
			break
		}
	}
	hasCons := false
	hasGen := false
	for _, d := range data {
		if d.ConsumptionKwh > 0 {
			hasCons = true
		}
		if d.GenerationKwh > 0 {
			hasGen = true
		}
	}

	// Find max kWh for y-scale
	maxKwh := 0.0
	for _, d := range data {
		if d.ConsumptionKwh > maxKwh {
			maxKwh = d.ConsumptionKwh
		}
		if d.GenerationKwh > maxKwh {
			maxKwh = d.GenerationKwh
		}
	}
	if maxKwh == 0 {
		return
	}

	// Round up max to a "nice" ceiling
	magnitude := math.Pow(10, math.Floor(math.Log10(maxKwh)))
	niceMax := math.Ceil(maxKwh/magnitude) * magnitude
	if niceMax == 0 {
		niceMax = 1
	}

	// Layout constants
	leftLabelW := 16.0 // space for y-axis labels
	startX := 20.0 + leftLabelW
	startY := pdf.GetY() + 5.0
	chartW := 170.0 - leftLabelW
	chartH := 34.0
	nTicks := 4

	// Title
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(60, 60, 60)
	pdf.SetXY(20, pdf.GetY())
	pdf.CellFormat(170, 5, "Wie hat sich Ihr Verbrauch / Ihre Einspeisung entwickelt?", "", 1, "L", false, 0, "")

	// Y-axis gridlines + labels
	pdf.SetFont("DejaVu", "", 6)
	for i := 0; i <= nTicks; i++ {
		frac := float64(i) / float64(nTicks)
		tickY := startY + chartH - frac*chartH
		val := niceMax * frac

		// Gridline (skip baseline)
		if i > 0 {
			pdf.SetDrawColor(210, 210, 210)
			pdf.SetLineWidth(0.2)
			pdf.Line(startX, tickY, startX+chartW, tickY)
		}

		// Label (right-align in the label column)
		pdf.SetTextColor(120, 120, 120)
		label := fmt.Sprintf("%.0f", val)
		pdf.SetXY(20, tickY-2.0)
		pdf.CellFormat(leftLabelW-2, 4, label, "", 0, "R", false, 0, "")
	}

	// Axes
	pdf.SetDrawColor(140, 140, 140)
	pdf.SetLineWidth(0.4)
	pdf.Line(startX, startY, startX, startY+chartH)               // Y-axis
	pdf.Line(startX, startY+chartH, startX+chartW, startY+chartH) // X-axis

	// Bars
	n := len(data)
	groupW := chartW / float64(n)
	gap := groupW * 0.12

	for i, d := range data {
		groupX := startX + float64(i)*groupW

		if hasBoth {
			// Two bars per month
			bw := (groupW - 3*gap) / 2

			if d.ConsumptionKwh > 0 {
				bh := d.ConsumptionKwh / niceMax * chartH
				pdf.SetFillColor(59, 130, 246)
				pdf.Rect(groupX+gap, startY+chartH-bh, bw, bh, "F")
			}
			if d.GenerationKwh > 0 {
				bh := d.GenerationKwh / niceMax * chartH
				pdf.SetFillColor(34, 197, 94)
				pdf.Rect(groupX+2*gap+bw, startY+chartH-bh, bw, bh, "F")
			}
		} else {
			bw := groupW - 2*gap
			if d.ConsumptionKwh > 0 {
				bh := d.ConsumptionKwh / niceMax * chartH
				pdf.SetFillColor(59, 130, 246)
				pdf.Rect(groupX+gap, startY+chartH-bh, bw, bh, "F")
			} else if d.GenerationKwh > 0 {
				bh := d.GenerationKwh / niceMax * chartH
				pdf.SetFillColor(34, 197, 94)
				pdf.Rect(groupX+gap, startY+chartH-bh, bw, bh, "F")
			}
		}

		// Month label (e.g. "Jän" + year suffix if January or first bar)
		pdf.SetFont("DejaVu", "", 6)
		pdf.SetTextColor(80, 80, 80)
		label := shortMonth(d.Month.Month())
		if d.Month.Month() == time.January || i == 0 {
			label += fmt.Sprintf(" %d", d.Month.Year())
		}
		pdf.SetXY(groupX, startY+chartH+1)
		pdf.CellFormat(groupW, 3.5, label, "", 0, "C", false, 0, "")
	}

	// Legend
	legendY := startY + chartH + 3.5
	pdf.SetFont("DejaVu", "", 7)
	pdf.SetTextColor(60, 60, 60)
	lx := startX
	if hasCons {
		pdf.SetFillColor(59, 130, 246)
		pdf.Rect(lx, legendY, 3.5, 2.5, "F")
		pdf.SetXY(lx+4.5, legendY-0.5)
		pdf.CellFormat(25, 3.5, "Bezug (kWh)", "", 0, "L", false, 0, "")
		lx += 31
	}
	if hasGen {
		pdf.SetFillColor(34, 197, 94)
		pdf.Rect(lx, legendY, 3.5, 2.5, "F")
		pdf.SetXY(lx+4.5, legendY-0.5)
		pdf.CellFormat(30, 3.5, "Einspeisung (kWh)", "", 0, "L", false, 0, "")
	}

	// Unit label top-right of chart
	pdf.SetFont("DejaVu", "", 6)
	pdf.SetTextColor(120, 120, 120)
	pdf.SetXY(startX+chartW-15, startY-3.5)
	pdf.CellFormat(15, 3.5, "kWh", "", 0, "R", false, 0, "")

	// Reset drawing state
	pdf.SetTextColor(0, 0, 0)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetLineWidth(0.2)
	pdf.SetXY(20, legendY+5)
}

// drawRechnungsstellerBlock draws the EEG address block (§11 UStG Rechnungssteller).
func drawRechnungsstellerBlock(pdf *fpdf.Fpdf, eeg *domain.EEG) {
	if eeg.Strasse == "" && eeg.Plz == "" && eeg.Ort == "" && eeg.UidNummer == "" {
		return
	}
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 5, "Rechnungssteller", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	if eeg.Strasse != "" {
		pdf.CellFormat(0, 4.5, eeg.Strasse, "", 1, "L", false, 0, "")
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(0, 4.5, strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 1, "L", false, 0, "")
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(0, 4.5, "UID: "+eeg.UidNummer, "", 1, "L", false, 0, "")
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)
}

// GenerateCreditNotePDF creates an A4 PDF Gutschrift for a producer member and returns raw bytes.
// werbebonusNet is an optional Migration-114 referral bonus (net, pre-VAT),
// already folded into inv.GenerationNetAmount/GenerationVatAmount by the
// caller — pass 0 when none applies. Shown as its own row; the Nettobetrag/
// Gutschriftbetrag summary below is read from inv's stored totals (not
// recomputed from generationKwh×producerPriceCt) so it stays consistent with
// that extra row.
func GenerateCreditNotePDF(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, producerPriceCt, generationKwh float64, generationMeterPoints []MeterPointKwh, monthlyItems []MonthlyKwh, history []MonthlyKwh, werbebonusNet float64) ([]byte, error) {
	pdf := newPDF()
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

	// ── Logo (optional) ──────────────────────────────────────────────────────
	embedLogo(pdf, eeg.LogoPath)

	// ── Header: title + EEG name + address ──────────────────────────────────
	pdf.SetFont("DejaVu", "B", 16)
	pdf.CellFormat(0, 9, "Gutschrift", "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 6, eeg.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	if eeg.Strasse != "" {
		pdf.CellFormat(0, 5, eeg.Strasse, "", 1, "L", false, 0, "")
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(0, 5, strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 1, "L", false, 0, "")
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(0, 5, "UID: "+eeg.UidNummer, "", 1, "L", false, 0, "")
	}
	pdf.Ln(6)

	// ── Recipient block ──────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 7, "Gutschrift an", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)

	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	pdf.CellFormat(0, 6, fullName, "", 1, "L", false, 0, "")
	if member.Strasse != "" {
		pdf.CellFormat(0, 6, member.Strasse, "", 1, "L", false, 0, "")
	}
	if member.Plz != "" || member.Ort != "" {
		pdf.CellFormat(0, 6, strings.TrimSpace(member.Plz+" "+member.Ort), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 6, "Mitgliedsnummer: "+member.MitgliedsNr, "", 1, "L", false, 0, "")
	if member.UidNummer != "" {
		pdf.CellFormat(0, 6, "UID-Nummer: "+member.UidNummer, "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// ── Gutschrift number, date & billing period ─────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Gutschrift-Nr.:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, creditNoteNr, "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Datum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, inv.CreatedAt.Format("02.01.2006"), "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Abrechnungszeitraum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, fmt.Sprintf("%s – %s",
		inv.PeriodStart.Format("02.01.2006"),
		inv.PeriodEnd.Format("02.01.2006"),
	), "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// ── Pre-text ────────────────────────────────────────────────────────────
	if eeg.InvoicePreText != "" {
		pdf.SetFont("DejaVu", "", 10)
		pdf.MultiCell(0, 6, eeg.InvoicePreText, "", "L", false)
		pdf.Ln(4)
	}

	// ── Line table ───────────────────────────────────────────────────────────
	colDesc := 80.0
	colKwh := 30.0
	colPrice := 40.0
	colAmount := 0.0
	rowH := 8.0
	vatH := 6.0

	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(colDesc, rowH, "Beschreibung", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colKwh, rowH, "kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colPrice, rowH, "Tarif je kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colAmount, rowH, "Betrag", "1", 1, "R", true, 0, "")

	pdf.SetFont("DejaVu", "", 10)
	pdf.SetFillColor(255, 255, 255)
	netAmount := generationKwh * producerPriceCt / 100

	if periodSpansMultipleMonths(inv.PeriodStart, inv.PeriodEnd) {
		// ── Multi-month credit note: one row per calendar month ───────────────
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
			pdf.CellFormat(colDesc, rowH, "Einspeisung Strom "+monthLabel, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colKwh, rowH, formatKwh(m.GenerationKwh), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", mPriceCt), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, rowH, formatAmount(genAmount), "1", 1, "R", false, 0, "")
		}
		drawTotalRow(pdf, colDesc, colKwh, colPrice, colAmount, rowH, "Summe Einspeisung", totalGenKwh, totalGenAmount, false)
		drawMpSubRow(pdf, generationMeterPoints)
	} else {
		// ── Single-month credit note: one row ─────────────────────────────────
		pdf.CellFormat(colDesc, rowH, "Einspeisung Strom "+period, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(generationKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", producerPriceCt), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(netAmount), "1", 1, "R", false, 0, "")
		drawMpSubRow(pdf, generationMeterPoints)
	}

	// Werbebonus — shown separately so the "Einspeisung Strom" row above stays a
	// pure kWh × Tarif figure; the Nettobetrag/Gutschriftbetrag below folds it in
	// via inv.GenerationNetAmount.
	if werbebonusNet > 0 {
		pdf.CellFormat(colDesc, rowH, "Werbebonus (Mitglied geworben)", "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, "", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, "", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(werbebonusNet), "1", 1, "R", false, 0, "")
	}

	// ── VAT section ──────────────────────────────────────────────────────────
	// For landwirt_pauschaliert: 13 % VAT is added on top (§ 22 UStG).
	// For all others: VAT is 0 (Reverse Charge or exempt — text-only notice).
	vatText := GenerationVATText(member)
	genVatPct := inv.GenerationVatPct
	genVatAmount := inv.GenerationVatAmount
	genRC := GenerationReverseCharge(member)
	// Nettobetrag reads from inv.GenerationNetAmount (not the locally recomputed
	// netAmount) so it includes any Werbebonus folded in by the caller.
	nettoDisplay := inv.GenerationNetAmount
	totalDisplay := nettoDisplay + genVatAmount
	// For RC: EEG remits VAT to Finanzamt — actual payout to producer is net only.
	// For Landwirt §22: producer receives gross (including 13 % Durchschnittssatz).
	payDisplay := totalDisplay
	if genRC {
		payDisplay = nettoDisplay
	}

	pdf.Ln(1)
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Nettobetrag", "0", 0, "R", false, 0, "")
	pdf.CellFormat(colAmount, vatH, formatAmount(nettoDisplay), "0", 1, "R", false, 0, "")

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

	pdf.Ln(2)
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 10, fmt.Sprintf("Gutschriftbetrag: %s", formatAmount(totalDisplay)), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	// ── Payment notice ───────────────────────────────────────────────────────
	// Respects the same invoice_payment_notice_mode as GeneratePDF's Zahlungshinweis
	// ("none" omits the section entirely; "custom" uses the free-text template).
	// sepa_lastschrift/ueberweisung (or unset) both fall back to the default payout
	// wording below — a Gutschrift payout doesn't have a debit-vs-transfer distinction
	// the way a consumer invoice does.
	if eeg.InvoicePaymentNoticeMode != "none" {
		pdf.SetFont("DejaVu", "B", 10)
		pdf.SetTextColor(40, 40, 40)
		pdf.CellFormat(0, 5, "Auszahlung", "", 1, "L", false, 0, "")
		pdf.SetFont("DejaVu", "", 10)
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
		pdf.MultiCell(0, 5, creditNotice, "", "L", false)
		pdf.Ln(4)
	}

	// ── Energy history bar chart ─────────────────────────────────────────────
	if len(history) > 0 {
		drawBarChart(pdf, history)
		pdf.Ln(2)
	}

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "", 8)
	pdf.SetTextColor(128, 128, 128)
	footerText := "Erstellt von eegabrechnung"
	if eeg.InvoiceFooterText != "" {
		footerText = eeg.InvoiceFooterText
	}
	pdf.CellFormat(0, 6, footerText, "", 1, "C", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("pdf generation error: %w", err)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

// embedLogo adds a company logo image to the PDF at the top-right if the path exists and is readable.
func embedLogo(pdf *fpdf.Fpdf, logoPath string) {
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
	// Place logo top-right, max 40mm wide, auto height
	pdf.ImageOptions("logo", 150, 15, 40, 0, false, opt, 0, "")
}

// GeneratePDF creates an A4 PDF invoice and returns the raw bytes.
func GeneratePDF(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member, vat VATOptions, history []MonthlyKwh) ([]byte, error) {
	pdf := newPDF()
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	period := periodLabel(inv.PeriodStart)

	// Format invoice number: use prefix + zero-padded number if configured
	invoiceNr := shortID(inv.ID.String())
	if inv.InvoiceNumber != nil {
		digits := eeg.InvoiceNumberDigits
		if digits <= 0 {
			digits = 4
		}
		invoiceNr = fmt.Sprintf("%s%0*d", eeg.InvoiceNumberPrefix, digits, *inv.InvoiceNumber)
	}

	// ── Logo (optional) ──────────────────────────────────────────────────────
	embedLogo(pdf, eeg.LogoPath)

	// ── Header: title + EEG name + address ──────────────────────────────────
	pdf.SetFont("DejaVu", "B", 16)
	pdf.CellFormat(0, 9, "Rechnung", "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 6, eeg.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	if eeg.Strasse != "" {
		pdf.CellFormat(0, 5, eeg.Strasse, "", 1, "L", false, 0, "")
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(0, 5, strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 1, "L", false, 0, "")
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(0, 5, "UID: "+eeg.UidNummer, "", 1, "L", false, 0, "")
	}
	pdf.Ln(6)

	// ── Recipient block ──────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 7, "Rechnungsempfänger", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)

	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	pdf.CellFormat(0, 6, fullName, "", 1, "L", false, 0, "")

	if member.Strasse != "" {
		pdf.CellFormat(0, 6, member.Strasse, "", 1, "L", false, 0, "")
	}
	if member.Plz != "" || member.Ort != "" {
		pdf.CellFormat(0, 6, strings.TrimSpace(member.Plz+" "+member.Ort), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 6, "Mitgliedsnummer: "+member.MitgliedsNr, "", 1, "L", false, 0, "")
	if member.UidNummer != "" {
		pdf.CellFormat(0, 6, "UID-Nummer: "+member.UidNummer, "", 1, "L", false, 0, "")
	}
	if member.Email != "" {
		pdf.CellFormat(0, 6, "E-Mail: "+member.Email, "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// ── Invoice number, date & billing period ────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Rechnungsnummer:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, invoiceNr, "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Rechnungsdatum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, inv.CreatedAt.Format("02.01.2006"), "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Abrechnungszeitraum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, fmt.Sprintf("%s – %s",
		inv.PeriodStart.Format("02.01.2006"),
		inv.PeriodEnd.Format("02.01.2006"),
	), "", 1, "L", false, 0, "")

	pdf.Ln(6)

	// ── Pre-text (optional) ──────────────────────────────────────────────────
	if eeg.InvoicePreText != "" {
		pdf.SetFont("DejaVu", "", 10)
		pdf.MultiCell(0, 6, eeg.InvoicePreText, "", "L", false)
		pdf.Ln(4)
	}

	// ── Line table ───────────────────────────────────────────────────────────
	colDesc := 80.0
	colKwh := 30.0
	colPrice := 40.0
	colAmount := 0.0 // fills remaining width

	rowH := 8.0 // main data row height
	vatH := 6.0 // VAT / summary row height

	// Header row
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(colDesc, rowH, "Beschreibung", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colKwh, rowH, "kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colPrice, rowH, "Preis je kWh", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colAmount, rowH, "Betrag", "1", 1, "R", true, 0, "")

	pdf.SetFont("DejaVu", "", 10)
	pdf.SetFillColor(255, 255, 255)

	multiMonth := periodSpansMultipleMonths(inv.PeriodStart, inv.PeriodEnd)
	if multiMonth {
		// ── Multi-month: one row per calendar month ───────────────────────────
		// All Bezug rows first, then all Einspeisung rows.
		totalConsKwh := 0.0
		totalConsAmount := 0.0
		for _, m := range vat.MonthlyLineItems {
			if m.ConsumptionKwh == 0 && vat.GenerationKwh > 0 {
				continue // pure producer — no consumption rows
			}
			monthLabel := germanMonth(m.Month.Month()) + " " + fmt.Sprintf("%d", m.Month.Year())
			priceCt := m.EnergyPriceCt
			if priceCt == 0 {
				priceCt = vat.EnergyPrice
			}
			energyAmount := m.ConsumptionKwh * priceCt / 100
			totalConsKwh += m.ConsumptionKwh
			totalConsAmount += energyAmount
			pdf.CellFormat(colDesc, rowH, "Bezug Strom "+monthLabel, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colKwh, rowH, formatKwh(m.ConsumptionKwh), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", priceCt), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, rowH, formatAmount(energyAmount), "1", 1, "R", false, 0, "")
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
			pdf.CellFormat(colDesc, rowH, "Einspeisung "+monthLabel, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colKwh, rowH, formatKwh(m.GenerationKwh), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", prodPriceCt), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, rowH, "-"+formatAmount(genAmount), "1", 1, "R", false, 0, "")
		}
		if vat.GenerationKwh > 0 {
			drawTotalRow(pdf, colDesc, colKwh, colPrice, colAmount, rowH, "Summe Einspeisung", totalGenKwh, totalGenAmount, true)
			drawMpSubRow(pdf, vat.GenerationMeterPointKwh)
		}
		// Fixed fees as separate line (were baked into ConsumptionNet for single-month)
		feeMonths := vat.FeeMonths
		if feeMonths < 1 {
			feeMonths = 1
		}
		feeTotal := (vat.MeterFeeEur + vat.ParticipationFeeEur) * float64(feeMonths)
		feeLabel := "Messstellengebühr / Teilnahmegebühr"
		if feeMonths > 1 {
			feeLabel = fmt.Sprintf("Messstellengebühr / Teilnahmegebühr (%d Monate)", feeMonths)
		}
		if feeTotal > 0 || eeg.InvoiceShowZeroFeeFixgebuehr {
			pdf.CellFormat(colDesc, rowH, feeLabel, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colKwh, rowH, "", "1", 0, "R", false, 0, "")
			pdf.CellFormat(colPrice, rowH, "", "1", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, rowH, formatAmount(feeTotal), "1", 1, "R", false, 0, "")
		}
	} else if vat.GenerationKwh > 0 {
		// ── Single-month prosumer: Bezug + Einspeisung ────────────────────────
		pdf.CellFormat(colDesc, rowH, "Bezug Strom "+period, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(vat.ConsumptionKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", vat.EnergyPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(vat.EnergyNet), "1", 1, "R", false, 0, "")
		drawMpSubRow(pdf, vat.ConsumptionMeterPointKwh)

		pdf.CellFormat(colDesc, rowH, "Einspeisung Strom "+period, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(vat.GenerationKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", vat.ProducerPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, "-"+formatAmount(vat.GenerationNet), "1", 1, "R", false, 0, "")
		drawMpSubRow(pdf, vat.GenerationMeterPointKwh)
	} else {
		// ── Single-month pure consumer ────────────────────────────────────────
		pdf.CellFormat(colDesc, rowH, "Bezug Strom "+period, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(inv.ConsumptionKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", vat.EnergyPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(vat.EnergyNet), "1", 1, "R", false, 0, "")
		drawMpSubRow(pdf, vat.ConsumptionMeterPointKwh)
	}

	// Zählpunktsgebühr — always its own line (single- and multi-month alike), unlike the
	// Messstellengebühr/Teilnahmegebühr line above which is only broken out for multi-month.
	if vat.ZaehlpunktsGebuehrTotal > 0 || eeg.InvoiceShowZeroFeeZaehlpunktsgebuehr {
		label := fmt.Sprintf("Zählpunktsgebühr (%d × %s)", vat.ZaehlpunktsGebuehrCount, formatAmount(vat.ZaehlpunktsGebuehrEur))
		if vat.FeeMonths > 1 {
			label = fmt.Sprintf("Zählpunktsgebühr (%d × %s × %d Monate)", vat.ZaehlpunktsGebuehrCount, formatAmount(vat.ZaehlpunktsGebuehrEur), vat.FeeMonths)
		}
		pdf.CellFormat(colDesc, rowH, label, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, "", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, "", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(vat.ZaehlpunktsGebuehrTotal), "1", 1, "R", false, 0, "")
	}

	// Servicegebühr Bezug — own line, own kWh × ct/kWh rate, like Zählpunktsgebühr above.
	if vat.ServiceFeeBezugTotal > 0 || eeg.InvoiceShowZeroFeeServicegebuehrBezug {
		pdf.CellFormat(colDesc, rowH, "Servicegebühr Bezug", "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(vat.ServiceFeeBezugKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", vat.ServiceFeeBezugCtKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(vat.ServiceFeeBezugTotal), "1", 1, "R", false, 0, "")
	}

	// Servicegebühr Einspeisung — folded into ConsumptionNet (not GenerationNet), so it is
	// shown as a positive charge here, same as Zählpunktsgebühr: the sum of all rows on this
	// table must equal the invoice total, and this fee already reduces the member's net
	// payout via a lower GenerationNet-relative balance, NOT via a negative row amount here
	// (a "-" sign would double-subtract it). See VATOptions doc comment for the VAT rationale.
	if vat.ServiceFeeEinspeisungTotal > 0 || eeg.InvoiceShowZeroFeeServicegebuehrEinspeisung {
		pdf.CellFormat(colDesc, rowH, "Servicegebühr Einspeisung", "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(vat.ServiceFeeEinspeisungKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", vat.ServiceFeeEinspeisungCtKwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(vat.ServiceFeeEinspeisungTotal), "1", 1, "R", false, 0, "")
	}

	// Werbebonus — Mitglieder-werben-Mitglieder-Prämie, already folded into
	// ConsumptionNet above; shown as its own negative row for transparency.
	if vat.WerbebonusTotal > 0 {
		pdf.CellFormat(colDesc, rowH, "Werbebonus (Mitglied geworben)", "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, "", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, "", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, "-"+formatAmount(vat.WerbebonusTotal), "1", 1, "R", false, 0, "")
	}

	// Zusatzzähler — manually-read submeters (e.g. Wärmepumpe), one line each.
	for _, em := range vat.ExtraMeterLines {
		label := fmt.Sprintf("%s (Zusatzzähler)", em.Label)
		pdf.CellFormat(colDesc, rowH, label, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colKwh, rowH, formatKwh(em.Kwh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPrice, rowH, fmt.Sprintf("%.4f ct", em.PriceCt), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(em.Amount), "1", 1, "R", false, 0, "")
	}

	// ── VAT breakdown — Bezug and Einspeisung treated independently (Austrian law) ──
	pdf.Ln(1)
	pdf.SetFont("DejaVu", "", 10)

	if vat.GenerationKwh > 0 {
		// ── Bezug VAT block ────────────────────────────────────────────────────
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
		pdf.SetFont("DejaVu", "B", 10)
		pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Bruttobetrag Bezug", "0", 0, "R", false, 0, "")
		pdf.CellFormat(colAmount, vatH, formatAmount(vat.ConsumptionGross), "0", 1, "R", false, 0, "")
		pdf.SetFont("DejaVu", "", 10)

		pdf.Ln(1)

		// ── Einspeisung VAT block ──────────────────────────────────────────────
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
			pdf.SetFont("DejaVu", "B", 10)
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Bruttobetrag Einspeisung", "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationGross), "0", 1, "R", false, 0, "")
			pdf.SetFont("DejaVu", "", 10)
		} else if vat.GenerationVatText != "" {
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, vat.GenerationVatText, "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, "0,00 €", "0", 1, "R", false, 0, "")
			pdf.SetFont("DejaVu", "B", 10)
			pdf.CellFormat(colDesc+colKwh+colPrice, vatH, "Bruttobetrag Einspeisung", "0", 0, "R", false, 0, "")
			pdf.CellFormat(colAmount, vatH, formatAmount(vat.GenerationGross), "0", 1, "R", false, 0, "")
			pdf.SetFont("DejaVu", "", 10)
		}

		pdf.Ln(2)
		pdf.SetFont("DejaVu", "B", 10)
		// For RC generation: EEG remits VAT to Finanzamt — only net is transferred to member.
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
		pdf.CellFormat(0, 10, totalLabel, "", 1, "R", false, 0, "")
	} else {
		// ── Pure consumer: single VAT block ───────────────────────────────────
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
		pdf.Ln(2)
		pdf.SetFont("DejaVu", "B", 10)
		pdf.CellFormat(0, 10, fmt.Sprintf("Bruttobetrag Bezug: %s", formatAmount(inv.TotalAmount)), "", 1, "R", false, 0, "")
	}

	pdf.Ln(4)

	// ── Payment notice ───────────────────────────────────────────────────────
	if eeg.InvoicePaymentNoticeMode != "none" {
		pdf.SetFont("DejaVu", "B", 10)
		pdf.SetTextColor(40, 40, 40)
		pdf.CellFormat(0, 5, "Zahlungshinweis", "", 1, "L", false, 0, "")
		pdf.SetFont("DejaVu", "", 10)
		pdf.SetTextColor(0, 0, 0)
		var notice string
		// For RC generation: EEG remits VAT to Finanzamt — only net is transferred to member.
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
			// Negative saldo — EEG owes the member → credit transfer
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
		pdf.MultiCell(0, 5, notice, "", "L", false)
	}

	if vat.GenerationReverseCharge && inv.GenerationVatAmount > 0 {
		pdf.Ln(3)
		pdf.SetFont("DejaVu", "", 9)
		pdf.SetTextColor(80, 80, 80)
		rcNote := fmt.Sprintf(
			"Die Umsatzsteuer auf die Einspeisevergütung in Höhe von %s wird gemäß § 2 Z 2 UStBBKV (Reverse Charge) von der Energiegemeinschaft selbst berechnet und an das Finanzamt abgeführt.",
			formatAmount(inv.GenerationVatAmount),
		)
		pdf.MultiCell(0, 5, rcNote, "", "L", false)
		pdf.SetFont("DejaVu", "", 10)
		pdf.SetTextColor(0, 0, 0)
	}

	pdf.Ln(4)

	// ── Energy history bar chart ─────────────────────────────────────────────
	if len(history) > 0 {
		drawBarChart(pdf, history)
		pdf.Ln(2)
	}

	// ── Post-text (optional) ─────────────────────────────────────────────────
	if eeg.InvoicePostText != "" {
		pdf.SetFont("DejaVu", "", 10)
		pdf.MultiCell(0, 6, eeg.InvoicePostText, "", "L", false)
		pdf.Ln(4)
	}

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "", 8)
	pdf.SetTextColor(128, 128, 128)
	footerText := "Erstellt von eegabrechnung"
	if eeg.InvoiceFooterText != "" {
		footerText = eeg.InvoiceFooterText
	}
	pdf.CellFormat(0, 6, footerText, "", 1, "C", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("pdf generation error: %w", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateStornorechnung creates a formal cancellation document (Stornorechnung) for a
// finalized invoice that is being cancelled. References the original invoice number and
// shows negated amounts as required by §11 UStG.
func GenerateStornorechnung(inv *domain.Invoice, eeg *domain.EEG, member *domain.Member) ([]byte, error) {
	pdf := newPDF()
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

	// ── Logo (optional) ──────────────────────────────────────────────────────
	embedLogo(pdf, eeg.LogoPath)

	// ── Header: title + EEG name + address ───────────────────────────────────
	pdf.SetFont("DejaVu", "B", 16)
	pdf.CellFormat(0, 9, "Stornorechnung", "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 6, eeg.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	if eeg.Strasse != "" {
		pdf.CellFormat(0, 5, eeg.Strasse, "", 1, "L", false, 0, "")
	}
	if eeg.Plz != "" || eeg.Ort != "" {
		pdf.CellFormat(0, 5, strings.TrimSpace(eeg.Plz+" "+eeg.Ort), "", 1, "L", false, 0, "")
	}
	if eeg.UidNummer != "" {
		pdf.CellFormat(0, 5, "UID: "+eeg.UidNummer, "", 1, "L", false, 0, "")
	}
	pdf.Ln(6)

	// ── Reference, date & billing period ─────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Storno zu Beleg:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, origNr, "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Stornodatum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, time.Now().Format("02.01.2006"), "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Ursprungsdatum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, inv.CreatedAt.Format("02.01.2006"), "", 1, "L", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(55, 7, "Abrechnungszeitraum:", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, fmt.Sprintf("%s – %s",
		inv.PeriodStart.Format("02.01.2006"),
		inv.PeriodEnd.Format("02.01.2006"),
	), "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// ── Member block ─────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "B", 10)
	if inv.DocumentType == "credit_note" {
		pdf.CellFormat(0, 7, "Storno Gutschrift an", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(0, 7, "Storno Rechnung an", "", 1, "L", false, 0, "")
	}
	pdf.SetFont("DejaVu", "", 10)
	fullName := strings.TrimSpace(member.Name1 + " " + member.Name2)
	pdf.CellFormat(0, 6, fullName, "", 1, "L", false, 0, "")
	if member.Strasse != "" {
		pdf.CellFormat(0, 6, member.Strasse, "", 1, "L", false, 0, "")
	}
	if member.Plz != "" || member.Ort != "" {
		pdf.CellFormat(0, 6, strings.TrimSpace(member.Plz+" "+member.Ort), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 6, "Mitgliedsnummer: "+member.MitgliedsNr, "", 1, "L", false, 0, "")
	if member.UidNummer != "" {
		pdf.CellFormat(0, 6, "UID-Nummer: "+member.UidNummer, "", 1, "L", false, 0, "")
	}
	pdf.Ln(8)

	// ── Storno notice ────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetFillColor(255, 245, 200)
	pdf.MultiCell(0, 6,
		fmt.Sprintf("Diese Stornorechnung hebt den Beleg %s vollständig auf. "+
			"Die nachfolgenden Beträge entsprechen den negativen Werten des Originalbelegs.", origNr),
		"1", "L", true,
	)
	pdf.SetFillColor(255, 255, 255)
	pdf.Ln(4)

	// ── Amount table ─────────────────────────────────────────────────────────
	colLabel := 130.0
	colAmount := 0.0
	rowH := 8.0

	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(colLabel, rowH, "Nettobetrag (storniert)", "0", 0, "L", false, 0, "")
	pdf.CellFormat(colAmount, rowH, formatAmount(-inv.NetAmount), "0", 1, "R", false, 0, "")

	if inv.VatPctApplied > 0 {
		pdf.CellFormat(colLabel, rowH, fmt.Sprintf("USt. (%.0f %%)", inv.VatPctApplied), "0", 0, "L", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(-inv.VatAmount), "0", 1, "R", false, 0, "")
	} else {
		pdf.CellFormat(colLabel, rowH, "USt. (0 %, steuerbefreit gem. § 6 UStG)", "0", 0, "L", false, 0, "")
		pdf.CellFormat(colAmount, rowH, formatAmount(0), "0", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(0, 10, fmt.Sprintf("Stornobetrag: %s", formatAmount(-inv.TotalAmount)), "", 1, "R", false, 0, "")
	pdf.Ln(10)

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetFont("DejaVu", "", 8)
	pdf.SetTextColor(128, 128, 128)
	footerText := "Erstellt von eegabrechnung"
	if eeg.InvoiceFooterText != "" {
		footerText = eeg.InvoiceFooterText
	}
	pdf.CellFormat(0, 6, footerText, "", 1, "C", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("storno pdf generation error: %w", err)
	}
	var stornoBuf bytes.Buffer
	if err := pdf.Output(&stornoBuf); err != nil {
		return nil, fmt.Errorf("storno pdf output error: %w", err)
	}
	return stornoBuf.Bytes(), nil
}

func pdfNextWorkday(t time.Time) time.Time {
	d := t.AddDate(0, 0, 1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}
