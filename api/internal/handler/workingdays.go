package handler

import "time"

// easterSunday returns Easter Sunday for the given year using the Anonymous
// Gregorian algorithm.
func easterSunday(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h+l-7*m+114)%31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// isAustrianHoliday returns true if t is an Austrian national public holiday.
func isAustrianHoliday(t time.Time) bool {
	y, mo, d := t.Date()
	// Fixed holidays
	switch {
	case mo == time.January && d == 1:   return true // Neujahr
	case mo == time.January && d == 6:   return true // Heilige Drei Könige
	case mo == time.May && d == 1:       return true // Staatsfeiertag
	case mo == time.August && d == 15:   return true // Mariä Himmelfahrt
	case mo == time.October && d == 26:  return true // Nationalfeiertag
	case mo == time.November && d == 1:  return true // Allerheiligen
	case mo == time.December && d == 8:  return true // Mariä Empfängnis
	case mo == time.December && d == 25: return true // Christtag
	case mo == time.December && d == 26: return true // Stefanitag
	}
	// Easter-dependent (offsets from Easter Sunday)
	easter := easterSunday(y)
	for _, offset := range []int{1, 39, 49, 60} { // Ostermontag, Himmelfahrt, Pfingstmontag, Fronleichnam
		h := easter.AddDate(0, 0, offset)
		if h.Month() == mo && h.Day() == d {
			return true
		}
	}
	return false
}

// isAustrianWorkingDay returns true if t is a working day (Mon–Fri, not a
// public holiday) in Austria.
func isAustrianWorkingDay(t time.Time) bool {
	wd := t.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !isAustrianHoliday(t)
}

// addAustrianWorkingDays returns the date that is exactly n Austrian working
// days after start (start itself is not counted).
func addAustrianWorkingDays(start time.Time, n int) time.Time {
	d := start.Truncate(24 * time.Hour)
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if isAustrianWorkingDay(d) {
			n--
		}
	}
	return d
}
