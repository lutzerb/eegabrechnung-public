package invoice

import (
	"time"

	"github.com/google/uuid"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// SampleFixtureData returns fully fictional EEG/member/invoice/VAT/energy data
// for design previews — used by cmd/pdfsample (standalone CLI) and the
// invoice-design preview HTTP endpoint, so there's a single source of truth for
// "Beispieldaten" instead of two independently-maintained fixtures. LogoPath is
// left empty; callers set it (a placeholder for the CLI, the real per-EEG logo
// for the preview endpoint). The member is a prosumer: two consumption meter
// points (zp1/zp2) so the preview also demonstrates the multi-Zählpunkt
// grouping (see drawEnergyPeriodTable / drawMpSubRow) — not just the common
// single-ZP case — plus one generation meter point (zp3) so both the Bezug
// and the Einspeisung branch of the pricing table render in the preview, and
// both the consumption and the generation breakdown table at the top of the
// invoice (see drawEnergyPeriodTable / drawGenerationPeriodTable) are shown.
// The returned history covers 6 months with a varying (not perfectly flat)
// community share so both drawBarChartThemed and drawPercentBarChartThemed
// render something representative in the design preview.
func SampleFixtureData() (*domain.EEG, *domain.Member, *domain.Invoice, VATOptions, []EnergyPeriodRow, []GenerationPeriodRow, []MonthlyKwh) {
	const zp1 = "AT0010000000000000000000000000001"
	const zp2 = "AT0010000000000000000000000000002"
	const zp3 = "AT0010000000000000000000000000003"

	eeg := &domain.EEG{
		ID:                       uuid.New(),
		Name:                     "Sonnenschein Energiegemeinschaft",
		GemeinschaftTyp:          "EEG",
		Strasse:                  "Musterstraße 1",
		Plz:                      "1010",
		Ort:                      "Wien",
		UidNummer:                "ATU00000000",
		EnergyPrice:              6.0,
		UseVat:                   true,
		VatPct:                   20,
		InvoiceNumberPrefix:      "2026_",
		InvoiceNumberDigits:      4,
		InvoicePaymentNoticeMode: "sepa_lastschrift",
		SepaPreNotificationDays:  14,
	}

	member := &domain.Member{
		ID:          uuid.New(),
		EegID:       eeg.ID,
		MitgliedsNr: "M009",
		Name1:       "Maria",
		Name2:       "Mustermann",
		Email:       "maria.mustermann@beispiel.at",
		Strasse:     "Musterweg 5",
		Plz:         "2100",
		Ort:         "Korneuburg",
		IBAN:        "AT00 0000 0000 0000 0000",
	}

	// Fee line items: Fixgebühr (MeterFeeEur) + Teilnahmegebühr (ParticipationFeeEur)
	// are non-zero so their "Messstellengebühr / Teilnahmegebühr" line always shows
	// in the preview; Zählpunktsgebühr is deliberately left at 0 (count > 0, rate
	// 0) so the preview also demonstrates invoice_show_zero_fees's effect — with
	// it off (default) the Zählpunktsgebühr line is omitted, with it on it shows
	// at 0,00 €.
	const (
		energyNet   = 15.14
		meterFee    = 2.50
		participFee = 1.00
		feeMonths   = 2
		feeTotal    = (meterFee + participFee) * feeMonths
		zpGebuehr   = 0.0
		zpCount     = 2
		zpTotal     = zpGebuehr * zpCount * feeMonths
		consNet     = energyNet + feeTotal + zpTotal
		consVatPct  = 20.0
		consVatAmt  = consNet * consVatPct / 100
		consGross   = consNet + consVatAmt

		// Einspeisung (generation credit) — modest feed-in on top of the consumption
		// above, so the preview also demonstrates the prosumer/Einspeisung branch of
		// the pricing table (see the vat.GenerationKwh > 0 checks in GeneratePDFThemed)
		// instead of only the pure-consumer case.
		producerPrice = 8.0
		genAprKwh     = 40.00
		genMayKwh     = 55.00
		genKwh        = genAprKwh + genMayKwh
		genAprAmt     = genAprKwh * producerPrice / 100
		genMayAmt     = genMayKwh * producerPrice / 100
		genNet        = genAprAmt + genMayAmt
		genVatPct     = 0.0
		genVatAmt     = genNet * genVatPct / 100
		genGross      = genNet + genVatAmt

		// Gesamteinspeisung (meter's total feed-in) is somewhat higher than the
		// Abnahme durch Energiegemeinschaft above — the difference is Resteinspeisung
		// (residual fed into the public grid), so the top-of-invoice generation
		// breakdown table (drawGenerationPeriodTable) shows a non-zero Rest column too.
		genAprTotal = 52.00
		genMayTotal = 70.00
	)

	invoiceNumber := 9
	inv := &domain.Invoice{
		ID:                   uuid.New(),
		MemberID:             member.ID,
		EegID:                eeg.ID,
		PeriodStart:          time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		ConsumptionKwh:       252.41,
		GenerationKwh:        genKwh,
		TotalKwh:             252.41,
		ConsumptionNetAmount: consNet,
		GenerationNetAmount:  genNet,
		NetAmount:            consNet - genNet,
		VatAmount:            consVatAmt + genVatAmt,
		VatPctApplied:        consVatPct,
		GenerationVatPct:     genVatPct,
		GenerationVatAmount:  genVatAmt,
		TotalAmount:          consGross - genGross,
		InvoiceNumber:        &invoiceNumber,
		Status:               "finalized",
		DocumentType:         "invoice",
		CreatedAt:            time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
	}

	// Split evenly-ish across the two fictional Zählpunkte; sums match the
	// monthly line items below (April 106.57, May 145.84). Generation (zp3) uses
	// the default Kleinunternehmer/private-producer VAT treatment (0 %, § 6 Abs. 1
	// Z 27 UStG) — matches invoice.GenerationVATText for a member with no UID and
	// no BusinessRole set, same as the fictional sample member above.
	vat := VATOptions{
		UseVat:               true,
		VatPct:               20,
		ConsumptionKwh:       252.41,
		GenerationKwh:        genKwh,
		ConsumptionNet:       consNet,
		GenerationNet:        genNet,
		EnergyPrice:          6.0,
		ProducerPrice:        producerPrice,
		ConsumptionVatPct:    consVatPct,
		ConsumptionVatAmount: consVatAmt,
		ConsumptionGross:     consGross,
		GenerationVatPct:     genVatPct,
		GenerationVatAmount:  genVatAmt,
		GenerationGross:      genGross,
		GenerationVatText:    "USt. (0 %), steuerbefreit gem. § 6 Abs. 1 Z 27 UStG",
		EnergyNet:            energyNet,
		MonthlyLineItems: []MonthlyKwh{
			{Month: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 106.57, EnergyPriceCt: 6.0, GenerationKwh: genAprKwh, ProducerPriceCt: producerPrice},
			{Month: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 145.84, EnergyPriceCt: 6.0, GenerationKwh: genMayKwh, ProducerPriceCt: producerPrice},
		},
		ConsumptionMeterPointKwh: []MeterPointKwh{
			{Zaehlpunkt: zp1, Kwh: 142.00},
			{Zaehlpunkt: zp2, Kwh: 110.41},
		},
		GenerationMeterPointKwh: []MeterPointKwh{
			{Zaehlpunkt: zp3, Kwh: genKwh},
		},
		MeterFeeEur:             meterFee,
		ParticipationFeeEur:     participFee,
		FeeMonths:               feeMonths,
		ZaehlpunktsGebuehrEur:   zpGebuehr,
		ZaehlpunktsGebuehrCount: zpCount,
		ZaehlpunktsGebuehrTotal: zpTotal,
	}

	energyRows := []EnergyPeriodRow{
		{
			Zaehlpunkt:            zp1,
			ZeitraumVon:           time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			ZeitraumBis:           time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			GesamtverbrauchKwh:    86.00,
			NetzbezugKwh:          26.00,
			CommunityVerbrauchKwh: 60.00,
		},
		{
			Zaehlpunkt:            zp1,
			ZeitraumVon:           time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			ZeitraumBis:           time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			GesamtverbrauchKwh:    118.00,
			NetzbezugKwh:          36.00,
			CommunityVerbrauchKwh: 82.00,
		},
		{
			Zaehlpunkt:            zp2,
			ZeitraumVon:           time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			ZeitraumBis:           time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			GesamtverbrauchKwh:    66.00,
			NetzbezugKwh:          19.43,
			CommunityVerbrauchKwh: 46.57,
		},
		{
			Zaehlpunkt:            zp2,
			ZeitraumVon:           time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			ZeitraumBis:           time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			GesamtverbrauchKwh:    88.80,
			NetzbezugKwh:          24.96,
			CommunityVerbrauchKwh: 63.84,
		},
	}

	generationRows := []GenerationPeriodRow{
		{
			Zaehlpunkt:           zp3,
			ZeitraumVon:          time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			ZeitraumBis:          time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			GesamteinspeisungKwh: genAprTotal,
			AbnahmeKwh:           genAprKwh,
			ResteinspeisungKwh:   genAprTotal - genAprKwh,
		},
		{
			Zaehlpunkt:           zp3,
			ZeitraumVon:          time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			ZeitraumBis:          time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			GesamteinspeisungKwh: genMayTotal,
			AbnahmeKwh:           genMayKwh,
			ResteinspeisungKwh:   genMayTotal - genMayKwh,
		},
	}

	// 6-month chart history (Dez 2025 - Mai 2026), varying community share so
	// both the absolute and percentage chart variants show something
	// representative rather than flat/static-looking bars. ConsumptionKwh/
	// GenerationKwh are the community-covered (billed) amounts — see
	// MonthlyKwh's doc comment — TotalConsumptionKwh/TotalGenerationKwh are
	// the member's physical totals, always >= the community amount.
	history := []MonthlyKwh{
		{Month: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 145.00, TotalConsumptionKwh: 210.00},
		{Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 150.00, TotalConsumptionKwh: 240.00},
		{Month: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 140.00, TotalConsumptionKwh: 190.00},
		{Month: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 130.00, TotalConsumptionKwh: 175.00, GenerationKwh: 28.00, TotalGenerationKwh: 35.00},
		{Month: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 76.00, TotalConsumptionKwh: 106.57, GenerationKwh: genAprKwh * 0.8, TotalGenerationKwh: genAprKwh},
		{Month: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), ConsumptionKwh: 106.00, TotalConsumptionKwh: 145.84, GenerationKwh: genMayKwh * 0.75, TotalGenerationKwh: genMayKwh},
	}

	return eeg, member, inv, vat, energyRows, generationRows, history
}
