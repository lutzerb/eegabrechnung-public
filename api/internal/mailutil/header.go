// Package mailutil provides safe construction of RFC 5322 email headers.
//
// Every outbound mail in this codebase assembles its headers by string
// concatenation. Any dynamic value placed into a header (recipient address,
// subject containing a member/applicant name, …) must have CR/LF removed first,
// otherwise an attacker who controls that value can inject additional headers or
// a forged body ("email header injection"). These helpers centralise that
// defense so no call site can forget it.
package mailutil

import (
	"mime"
	"strings"
)

// SanitizeHeaderValue removes CR and LF (and other C0 control characters except
// tab) from a header value, collapsing them to nothing. This neutralises header
// injection while leaving ordinary text and addresses intact.
func SanitizeHeaderValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		if r < 0x20 && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// EncodeSubject sanitises then RFC 2047 encodes a subject line. Q-encoding makes
// non-ASCII subjects (German umlauts etc.) display correctly and, because the
// value is sanitised first, guarantees no CR/LF survives into the header.
func EncodeSubject(s string) string {
	return mime.QEncoding.Encode("utf-8", SanitizeHeaderValue(s))
}

// Headers returns the standard From/To/Subject header block (each line CRLF
// terminated) with all values sanitised and the subject RFC 2047 encoded.
// Call sites append their remaining headers (MIME-Version, Content-Type, …) and
// the body after this.
func Headers(from, to, subject string) string {
	return "From: " + SanitizeHeaderValue(from) + "\r\n" +
		"To: " + SanitizeHeaderValue(to) + "\r\n" +
		"Subject: " + EncodeSubject(subject) + "\r\n"
}
