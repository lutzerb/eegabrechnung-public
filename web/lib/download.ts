// Parses a filename out of a Content-Disposition header value, preferring the
// RFC 5987/6266 filename*=UTF-8''... form (the real, non-ASCII-safe name the
// API sends for names with umlauts etc.) over the legacy ASCII filename="..."
// fallback, which is what the API always sends alongside it.
export function filenameFromContentDisposition(
  contentDisposition: string | null | undefined,
  fallback: string,
): string {
  if (!contentDisposition) return fallback;
  const extMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (extMatch) {
    try {
      return decodeURIComponent(extMatch[1]);
    } catch {
      // fall through to the plain filename= below
    }
  }
  const plainMatch = contentDisposition.match(/filename="([^"]+)"/);
  return plainMatch?.[1] ?? fallback;
}
