package domain

import (
	"testing"

	"github.com/google/uuid"
)

func f(v float64) *float64 { return &v }

func plan(name string, withEntries bool) *TariffSchedule {
	s := &TariffSchedule{ID: uuid.New(), Name: name}
	if withEntries {
		s.Entries = []TariffEntry{{EnergyPrice: 10}}
	}
	return s
}

func TestResolve(t *testing.T) {
	member := uuid.New()

	def := plan("default", true)
	def.IsDefault = true
	def.FreeKwhOverride = f(999) // must be ignored on the Standardtarif

	t.Run("no assignment uses Standardtarif and EEG fees", func(t *testing.T) {
		b := &BillingTariffs{Default: def, Shared: map[uuid.UUID]*TariffSchedule{def.ID: def}}
		r := b.Resolve(member)
		if r.PriceTariff != def {
			t.Fatalf("PriceTariff = %v, want default", r.PriceTariff)
		}
		if r.FreeKwh != nil {
			t.Fatalf("FreeKwh = %v, want nil (default overrides ignored)", *r.FreeKwh)
		}
	})

	t.Run("assigned plan with entries wins for price and fees", func(t *testing.T) {
		a := plan("gewerbe", true)
		a.MeterFeeEurOverride = f(5)
		b := &BillingTariffs{
			Default:    def,
			Shared:     map[uuid.UUID]*TariffSchedule{def.ID: def, a.ID: a},
			Assignment: map[uuid.UUID]uuid.UUID{member: a.ID},
		}
		r := b.Resolve(member)
		if r.PriceTariff != a {
			t.Fatalf("PriceTariff = %v, want assigned", r.PriceTariff)
		}
		if r.MeterFeeEur == nil || *r.MeterFeeEur != 5 {
			t.Fatalf("MeterFeeEur = %v, want 5", r.MeterFeeEur)
		}
	})

	t.Run("fee-only assigned plan keeps Standardtarif prices", func(t *testing.T) {
		a := plan("fee-only", false)
		a.DiscountPctOverride = f(10)
		b := &BillingTariffs{
			Default:    def,
			Shared:     map[uuid.UUID]*TariffSchedule{def.ID: def, a.ID: a},
			Assignment: map[uuid.UUID]uuid.UUID{member: a.ID},
		}
		r := b.Resolve(member)
		if r.PriceTariff != def {
			t.Fatalf("PriceTariff = %v, want default", r.PriceTariff)
		}
		if r.DiscountPct == nil || *r.DiscountPct != 10 {
			t.Fatalf("DiscountPct = %v, want 10", r.DiscountPct)
		}
	})

	t.Run("Individualtarif beats assigned plan, fees cascade per field", func(t *testing.T) {
		a := plan("assigned", true)
		a.MeterFeeEurOverride = f(5)
		a.ParticipationFeeEurOverride = f(7)
		ind := plan("individual", true)
		ind.MemberID = &member
		ind.MeterFeeEurOverride = f(0) // explicit 0 = exempt, must beat the assigned plan's 5
		b := &BillingTariffs{
			Default:    def,
			Shared:     map[uuid.UUID]*TariffSchedule{def.ID: def, a.ID: a},
			Individual: map[uuid.UUID]*TariffSchedule{member: ind},
			Assignment: map[uuid.UUID]uuid.UUID{member: a.ID},
		}
		r := b.Resolve(member)
		if r.PriceTariff != ind {
			t.Fatalf("PriceTariff = %v, want individual", r.PriceTariff)
		}
		if r.MeterFeeEur == nil || *r.MeterFeeEur != 0 {
			t.Fatalf("MeterFeeEur = %v, want 0", r.MeterFeeEur)
		}
		if r.ParticipationFeeEur == nil || *r.ParticipationFeeEur != 7 {
			t.Fatalf("ParticipationFeeEur = %v, want 7 (from assigned plan)", r.ParticipationFeeEur)
		}
	})

	t.Run("fee-only Individualtarif falls back to assigned plan prices", func(t *testing.T) {
		a := plan("assigned", true)
		ind := plan("individual", false)
		ind.MemberID = &member
		b := &BillingTariffs{
			Default:    def,
			Shared:     map[uuid.UUID]*TariffSchedule{def.ID: def, a.ID: a},
			Individual: map[uuid.UUID]*TariffSchedule{member: ind},
			Assignment: map[uuid.UUID]uuid.UUID{member: a.ID},
		}
		if r := b.Resolve(member); r.PriceTariff != a {
			t.Fatalf("PriceTariff = %v, want assigned", r.PriceTariff)
		}
	})

	t.Run("assignment to a plan that is no longer active falls back to Standardtarif", func(t *testing.T) {
		b := &BillingTariffs{
			Default:    def,
			Shared:     map[uuid.UUID]*TariffSchedule{def.ID: def},
			Assignment: map[uuid.UUID]uuid.UUID{member: uuid.New()},
		}
		if r := b.Resolve(member); r.PriceTariff != def {
			t.Fatalf("PriceTariff = %v, want default", r.PriceTariff)
		}
	})

	t.Run("no Standardtarif at all means flat EEG prices", func(t *testing.T) {
		b := &BillingTariffs{}
		if r := b.Resolve(member); r.PriceTariff != nil {
			t.Fatalf("PriceTariff = %v, want nil", r.PriceTariff)
		}
	})
}
