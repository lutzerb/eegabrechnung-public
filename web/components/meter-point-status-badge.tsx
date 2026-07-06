// Derived status badge for meter points.
// Priority order matters: an ongoing EDA process overrides both registriert_seit and
// abgemeldet_am — e.g. a re-registration in flight for a previously deregistered
// Zählpunkt should show "Anmeldung läuft", not "Abgemeldet" (abgemeldet_am only
// clears once that EC_REQ_ONL is actually confirmed).
export function MeterPointStatusBadge({
  registriertSeit,
  abgemeldetAm,
  anmeldungStatus,
  abmeldungStatus,
  fullWidth = false,
}: {
  registriertSeit?: string;
  abgemeldetAm?: string;
  anmeldungStatus?: string;
  abmeldungStatus?: string;
  fullWidth?: boolean;
}) {
  const badge = deriveBadge({ registriertSeit, abgemeldetAm, anmeldungStatus, abmeldungStatus });
  return (
    <span className={`inline-flex items-center justify-center px-2 py-0.5 rounded text-xs font-medium ${fullWidth ? "w-full" : ""} ${badge.cls}`}>
      {badge.label}
    </span>
  );
}

export function deriveBadge({
  registriertSeit,
  abgemeldetAm,
  anmeldungStatus,
  abmeldungStatus,
}: {
  registriertSeit?: string;
  abgemeldetAm?: string;
  anmeldungStatus?: string;
  abmeldungStatus?: string;
}): { label: string; cls: string } {
  const laufend = ["pending", "sent", "first_confirmed"];

  // Ongoing EDA processes reflect live reality more accurately than the scalar
  // abgemeldet_am/registriert_seit fields, which can be stale mid-flight (e.g.
  // abgemeldet_am is still set while a re-registration is pending confirmation).
  if (abmeldungStatus && laufend.includes(abmeldungStatus))
    return { label: "Abmeldung läuft", cls: "bg-orange-100 text-orange-700" };
  if (anmeldungStatus && laufend.includes(anmeldungStatus))
    return { label: "Anmeldung läuft", cls: "bg-yellow-100 text-yellow-700" };
  if (abgemeldetAm)
    return { label: "Abgemeldet", cls: "bg-slate-600 text-white" };
  if (abmeldungStatus === "rejected" || abmeldungStatus === "error")
    return { label: "Abmeldung Fehler", cls: "bg-red-100 text-red-700" };
  if (anmeldungStatus === "rejected" || anmeldungStatus === "error")
    return { label: "Anmeldung Fehler", cls: "bg-red-100 text-red-700" };
  if (registriertSeit)
    return { label: "Aktiv", cls: "bg-green-100 text-green-700" };
  return { label: "Ausstehend", cls: "bg-slate-100 text-slate-500" };
}
