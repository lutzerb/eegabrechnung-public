package handler

import (
	"net/http"

	"github.com/lutzerb/eegabrechnung/internal/netzbetreiber"
)

type netzbetreiberPrefixesResponse struct {
	KnownPrefixes []string          `json:"known_prefixes"`
	Overrides     map[string]string `json:"overrides"`
}

// ListNetzbetreiberPrefixes handles GET /api/v1/public/netzbetreiber-prefixes.
//
// Returns the static Zählpunkt-prefix → Netzbetreiber reference data (not
// tenant-specific, not sensitive) so the frontend can mirror
// netzbetreiber.ResolveRoutingID client-side for immediate form feedback,
// without a round trip per keystroke.
//
//	@Summary		List known Netzbetreiber prefixes
//	@Description	Returns all known Zählpunkt-prefixes and historical prefix overrides, for client-side Zählpunkt validation.
//	@Tags			Onboarding
//	@Produce		json
//	@Success		200	{object}	netzbetreiberPrefixesResponse
//	@Router			/public/netzbetreiber-prefixes [get]
func ListNetzbetreiberPrefixes(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, netzbetreiberPrefixesResponse{
		KnownPrefixes: netzbetreiber.KnownPrefixes(),
		Overrides:     netzbetreiber.Overrides(),
	})
}
