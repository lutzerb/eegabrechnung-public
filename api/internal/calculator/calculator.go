package calculator

import (
	"time"

	"github.com/google/uuid"
)

// Reading is a single energy reading for a meter point.
type Reading struct {
	MeterPointID    uuid.UUID
	Energierichtung string // "CONSUMPTION" or "GENERATION"
	Ts              time.Time
	WhTotal         float64
	WhCommunity     float64
	WhSelf          float64
}

// AllocationResult holds the calculated community allocation for a meter point at a timestamp.
type AllocationResult struct {
	MeterPointID uuid.UUID
	Ts           time.Time
	Allocated    float64 // wh allocated from community generation
}

// Calculate performs DYNAMIC allocation from raw generation/consumption readings.
// For each timestamp: distribute generation proportionally to consumption.
func Calculate(readings []Reading) []AllocationResult {
	// Group by timestamp
	type tsKey = time.Time
	byTs := map[tsKey][]Reading{}
	for _, r := range readings {
		byTs[r.Ts] = append(byTs[r.Ts], r)
	}

	var results []AllocationResult
	for ts, rds := range byTs {
		var totalGeneration float64
		var totalConsumption float64

		for _, r := range rds {
			switch r.Energierichtung {
			case "GENERATION":
				totalGeneration += r.WhTotal
			case "CONSUMPTION":
				totalConsumption += r.WhTotal
			}
		}

		for _, r := range rds {
			if r.Energierichtung != "CONSUMPTION" {
				continue
			}
			var allocated float64
			if totalConsumption > 0 {
				shareRatio := r.WhTotal / totalConsumption
				allocated = totalGeneration * shareRatio
				if allocated > r.WhTotal {
					allocated = r.WhTotal
				}
			}
			results = append(results, AllocationResult{
				MeterPointID: r.MeterPointID,
				Ts:           ts,
				Allocated:    allocated,
			})
		}
	}
	return results
}

// MemberBilling holds aggregated billing amounts for a member.
type MemberBilling struct {
	MemberID    uuid.UUID
	TotalKwh    float64 // sum of wh_community across all meter points for the period
	TotalAmount float64 // TotalKwh * EEG.EnergyPrice
}

// AggregateMemberBilling sums community energy per member and multiplies by price.
// memberMeterPoints maps memberID → list of meter point IDs.
// readings are all readings for the EEG in the billing period.
func AggregateMemberBilling(
	readings []Reading,
	memberMeterPoints map[uuid.UUID][]uuid.UUID,
	energyPrice float64,
) []MemberBilling {
	// Sum wh_community per meter point
	whByMeterPoint := map[uuid.UUID]float64{}
	for _, r := range readings {
		whByMeterPoint[r.MeterPointID] += r.WhCommunity
	}

	var result []MemberBilling
	for memberID, meterPointIDs := range memberMeterPoints {
		var totalWh float64
		for _, mpID := range meterPointIDs {
			totalWh += whByMeterPoint[mpID]
		}
		result = append(result, MemberBilling{
			MemberID:    memberID,
			TotalKwh:    totalWh,
			TotalAmount: totalWh * energyPrice,
		})
	}
	return result
}
