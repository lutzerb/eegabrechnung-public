// Package processes defines EDA process codes and their metadata.
package processes

import "fmt"

// Process code constants as defined by the Austrian MaKo standard.
const (
	AnforderungECON  = "ANFORDERUNG_ECON"  // Request energy data online
	AnforderungECOF  = "ANFORDERUNG_ECOF"  // Request energy data offline
	AnforderungCPF   = "ANFORDERUNG_CPF"   // Consumer participation factor
	AnforderungECP   = "ANFORDERUNG_ECP"   // Energy community participation
	AnforderungECC   = "ANFORDERUNG_ECC"   // Energy community contribution
	AnforderungCCMO  = "ANFORDERUNG_CCMO"  // Community contribution monthly overview
	AnforderungGN    = "ANFORDERUNG_GN"    // Grid notification
	AufhebungCCMS    = "AUFHEBUNG_CCMS"    // Cancellation
)

// All returns all known process codes.
var All = []string{
	AnforderungECON,
	AnforderungECOF,
	AnforderungCPF,
	AnforderungECP,
	AnforderungECC,
	AnforderungCCMO,
	AnforderungGN,
	AufhebungCCMS,
}

// versions maps process code to its XML schema version string.
var versions = map[string]string{
	AnforderungECON: "01.00",
	AnforderungECOF: "01.00",
	AnforderungCPF:  "01.00",
	AnforderungECP:  "01.00",
	AnforderungECC:  "01.00",
	AnforderungCCMO: "01.00",
	AnforderungGN:   "01.00",
	AufhebungCCMS:   "01.00",
}

// Version returns the XML schema version for a given process code.
// Returns an error if the process code is unknown.
func Version(process string) (string, error) {
	v, ok := versions[process]
	if !ok {
		return "", fmt.Errorf("unknown process code: %s", process)
	}
	return v, nil
}

// IsKnown returns true if the process code is known.
func IsKnown(process string) bool {
	_, ok := versions[process]
	return ok
}
