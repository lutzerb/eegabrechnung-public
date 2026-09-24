// Package edaversion tracks ebutilities.at Marktprozesse version cutover dates.
//
// edanet periodically republishes its "Marktprozesse" interface list with new
// Prozess-Id subject values and/or new XML schema versions, effective from a
// fixed calendar date. Because this service is deployed continuously (not
// only on the cutover date itself), the old and new values must both live in
// the code at the same time, with the switch happening at runtime based on
// the current date — not by editing values in place right before the cutover.
//
// Add a new named Cutover constant here for each future release, then
// reference it from an Entry in the relevant History table.
package edaversion

import "time"

// Vienna is shared so cutover dates compare in local Austrian time, matching
// edanet's own "gültig ab" dates (calendar dates, not UTC instants).
var Vienna = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		return time.FixedZone("CET", 3600)
	}
	return loc
}()

// Named ebutilities.at Marktprozesse cutover dates. Add one per release and
// reference it from the affected History tables in mail.go / ecmplist_builder.go etc.
var (
	// Cutover2026_10_05 is "Marktprozesse Version 04.10" (Oktober 2026).
	Cutover2026_10_05 = time.Date(2026, 10, 5, 0, 0, 0, 0, Vienna)
)

// Entry is one historical value, effective from a given date.
type Entry[T any] struct {
	From  time.Time // Vienna-local effective date; zero value = "seit jeher gültig"
	Value T
}

// Resolve returns the value of the latest Entry whose From is on or before now.
// history must be sorted ascending by From. Returns the zero value of T if
// history is empty or now is before every entry's From.
func Resolve[T any](history []Entry[T], now time.Time) T {
	var current T
	for _, e := range history {
		if e.From.After(now) {
			break
		}
		current = e.Value
	}
	return current
}
