// Client-side validation utilities for structured financial/utility identifiers

/**
 * Validates an IBAN using the MOD-97 checksum algorithm.
 * Returns an error message string, or null if valid (or empty).
 */
export function validateIBAN(raw: string): string | null {
  const iban = raw.replace(/\s+/g, "").toUpperCase();
  if (iban.length === 0) return null;

  if (iban.length < 15 || iban.length > 34) {
    return `Ungültige IBAN-Länge (${iban.length} Zeichen, erwartet 15–34).`;
  }

  if (!/^[A-Z]{2}[0-9]{2}[A-Z0-9]+$/.test(iban)) {
    return "IBAN enthält ungültige Zeichen.";
  }

  // Move first 4 chars to the end, then replace letters with digits (A=10 … Z=35)
  const rearranged = iban.slice(4) + iban.slice(0, 4);
  const numeric = rearranged
    .split("")
    .map((c) => {
      const code = c.charCodeAt(0);
      return code >= 65 && code <= 90 ? String(code - 55) : c;
    })
    .join("");

  // MOD-97 on arbitrary-length integer (process in 9-digit chunks)
  let remainder = 0;
  for (const ch of numeric) {
    remainder = (remainder * 10 + parseInt(ch, 10)) % 97;
  }

  if (remainder !== 1) {
    return "IBAN-Prüfziffer ungültig.";
  }
  return null;
}

/**
 * Validates a BIC/SWIFT code (8 or 11 characters).
 * Returns an error message string, or null if valid (or empty).
 */
export function validateBIC(raw: string): string | null {
  const bic = raw.replace(/\s+/g, "").toUpperCase();
  if (bic.length === 0) return null;

  // Format: AAAABBCC[DDD]
  //   AAAA = bank code (4 letters)
  //   BB   = country code (2 letters)
  //   CC   = location code (2 alphanumeric)
  //   DDD  = branch code (3 alphanumeric, optional)
  if (!/^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$/.test(bic)) {
    return "Ungültiger BIC (Beispiel: RLNWATWWXXX oder BKAUATWW).";
  }
  return null;
}

/**
 * Validates an Austrian Zählpunktnummer:
 *   - Exactly 33 characters
 *   - Uppercase alphanumeric only
 *   - Must start with "AT"
 * Returns an error message string, or null if valid (or empty).
 */
export function validateZaehlpunkt(raw: string): string | null {
  const zp = raw.trim().toUpperCase();
  if (zp.length === 0) return null;

  if (!/^AT/.test(zp)) {
    return "Zählpunktnummer muss mit AT beginnen.";
  }
  if (!/^[A-Z0-9]+$/.test(zp)) {
    return "Zählpunktnummer darf nur Großbuchstaben und Ziffern enthalten.";
  }
  if (zp.length !== 33) {
    return `Zählpunktnummer muss genau 33 Zeichen lang sein (aktuell: ${zp.length}).`;
  }
  return null;
}

/**
 * Validates a SEPA Gläubiger-ID (Creditor Identifier).
 * Format: 2-letter country code + 2 check digits + 3-char business code + national identifier.
 * Returns an error message string, or null if valid (or empty).
 */
export function validateSepaCreditorId(raw: string): string | null {
  const cid = raw.replace(/\s+/g, "").toUpperCase();
  if (cid.length === 0) return null;

  if (cid.length < 8 || cid.length > 35) {
    return `Ungültige Gläubiger-ID-Länge (${cid.length} Zeichen, erwartet 8–35).`;
  }
  if (!/^[A-Z]{2}[0-9]{2}[A-Z0-9]{3}[A-Z0-9]+$/.test(cid)) {
    return "Ungültige Gläubiger-ID (Beispiel: AT98ZZZ01234567890).";
  }
  return null;
}


/**
 * Validates an Austrian UID-Nummer (Umsatzsteuer-Identifikationsnummer).
 * Format: ATU followed by exactly 8 digits (e.g. ATU12345678).
 * Returns an error message string, or null if valid (or empty).
 */
export function validateUIDNummer(raw: string): string | null {
  const uid = raw.replace(/\s+/g, "").toUpperCase();
  if (uid.length === 0) return null;

  if (!/^ATU[0-9]{8}$/.test(uid)) {
    return "Ungültige UID-Nummer. Format: ATU gefolgt von 8 Ziffern (z.B. ATU12345678).";
  }
  return null;
}

/** Format an IBAN in groups of 4 characters (e.g. "AT61 1904 3002 3457 3201"). */
export function formatIBAN(iban: string): string {
  const clean = iban.replace(/\s/g, "").toUpperCase();
  return clean.match(/.{1,4}/g)?.join(" ") ?? clean;
}
