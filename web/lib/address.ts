export function formatAddress(a: { strasse?: string; plz?: string; ort?: string }): string {
  const plzOrt = [a.plz, a.ort].filter(Boolean).join(" ");
  return [a.strasse, plzOrt].filter(Boolean).join(", ");
}
