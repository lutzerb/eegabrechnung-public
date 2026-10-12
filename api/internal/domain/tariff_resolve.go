package domain

import "github.com/google/uuid"

// BillingTariffs holds every tariff schedule relevant for one billing run (entries limited
// to the billing period) and resolves which one applies to a member.
type BillingTariffs struct {
	Default    *TariffSchedule               // EEG Standardtarif
	Shared     map[uuid.UUID]*TariffSchedule // active EEG-wide plans by schedule ID
	Individual map[uuid.UUID]*TariffSchedule // active Individualtarife by member ID
	Assignment map[uuid.UUID]uuid.UUID       // member ID → assigned EEG-wide schedule ID
}

// ResolvedTariff is the outcome of resolving the tariff for one member. Override pointers are
// nil when the EEG-level value applies.
type ResolvedTariff struct {
	// PriceTariff supplies the time-of-use energy prices; nil = flat EEG prices.
	PriceTariff *TariffSchedule

	FreeKwh                   *float64
	DiscountPct               *float64
	MeterFeeEur               *float64
	ParticipationFeeEur       *float64
	ZaehlpunktsGebuehrEur     *float64
	ServicegebuehrBezugCtKwh  *float64
	ServicegebuehrEinspeisung *float64
}

// Resolve applies the precedence Individualtarif > assigned plan > Standardtarif.
// Prices: the first level that has its own entries wins. Fee overrides: per field, the first
// level with a non-nil value wins. Overrides of the Standardtarif are ignored — the EEG-wide
// fee values live in the EEG settings.
func (b *BillingTariffs) Resolve(memberID uuid.UUID) ResolvedTariff {
	var levels []*TariffSchedule
	if s, ok := b.Individual[memberID]; ok && s != nil {
		levels = append(levels, s)
	}
	if id, ok := b.Assignment[memberID]; ok {
		if s, ok := b.Shared[id]; ok && s != nil {
			levels = append(levels, s)
		}
	}

	r := ResolvedTariff{PriceTariff: b.Default}
	for _, s := range levels {
		if len(s.Entries) > 0 {
			r.PriceTariff = s
			break
		}
	}
	for _, s := range levels {
		if s.IsDefault {
			continue
		}
		r.FreeKwh = firstNonNil(r.FreeKwh, s.FreeKwhOverride)
		r.DiscountPct = firstNonNil(r.DiscountPct, s.DiscountPctOverride)
		r.MeterFeeEur = firstNonNil(r.MeterFeeEur, s.MeterFeeEurOverride)
		r.ParticipationFeeEur = firstNonNil(r.ParticipationFeeEur, s.ParticipationFeeEurOverride)
		r.ZaehlpunktsGebuehrEur = firstNonNil(r.ZaehlpunktsGebuehrEur, s.ZaehlpunktsGebuehrOverride)
		r.ServicegebuehrBezugCtKwh = firstNonNil(r.ServicegebuehrBezugCtKwh, s.ServicegebuehrBezugCtKwhOverride)
		r.ServicegebuehrEinspeisung = firstNonNil(r.ServicegebuehrEinspeisung, s.ServicegebuehrEinspeisungCtKwhOverride)
	}
	return r
}

func firstNonNil(cur, next *float64) *float64 {
	if cur != nil {
		return cur
	}
	return next
}
