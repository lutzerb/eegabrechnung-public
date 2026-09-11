// Canonical German labels/colors for EDA process types and eda_processes.status values.
// Single source of truth so the processes table, messages table, meter-point badges and
// registration timeline don't drift into different wording/colors for the same concept.

export const EDA_PROCESS_TYPE_LABELS: Record<string, string> = {
  EC_REQ_ONL:     "Anmeldung",
  EC_PRTFACT_CHG: "Teilnahmefaktor",
  CM_REV_SP:      "Widerruf",
  CR_REQ_PT:      "Zählerstandsgang",
  EC_PODLIST:     "Zählpunktliste",
};

export const EDA_PROCESS_STATUS_LABELS: Record<string, string> = {
  pending:         "Ausstehend",
  sent:            "Gesendet",
  first_confirmed: "Erst-Bestätigt",
  confirmed:       "Bestätigt",
  completed:       "Abgeschlossen",
  rejected:        "Abgelehnt",
  error:           "Fehler",
};

export const EDA_PROCESS_STATUS_STYLES: Record<string, string> = {
  pending:         "bg-yellow-50 text-yellow-700",
  sent:            "bg-blue-50 text-blue-700",
  first_confirmed: "bg-indigo-50 text-indigo-700",
  confirmed:       "bg-green-50 text-green-700",
  completed:       "bg-green-100 text-green-800",
  rejected:        "bg-red-50 text-red-700",
  error:           "bg-red-100 text-red-800",
};

// Message/wire codes are a superset of the process types (also includes response codes
// like ZUSTIMMUNG_ECON that never appear as an eda_processes.process_type). CM_REV_SP is
// overridden to disambiguate from the customer-/NB-initiated revoke variants below.
// Shared by the Nachrichten tab (eda-messages-table.tsx) and the per-process message
// accordion (eda-processes-table.tsx) so message-type wording never drifts between them.
export const EDA_MESSAGE_TYPE_LABELS: Record<string, string> = {
  ...EDA_PROCESS_TYPE_LABELS,
  DATEN_CRMSG:      "Energiedaten (Antwort)",
  ANTWORT_PT:       "Edanet-Eingangsbestätigung",
  CM_REV_SP:        "Widerruf (EEG)",
  CM_REV_CUS:       "Widerruf durch Kunde",
  CM_REV_IMP:       "Widerruf durch NB (Unmöglichkeit)",
  ZUSTIMMUNG_ECON:  "Zustimmung",
  ABLEHNUNG_ECON:   "Ablehnung",
  ANTWORT_ECON:     "Zwischenbestätigung",
  ABSCHLUSS_ECON:   "Abschluss",
  SENDEN_ECP:       "Zählpunktliste",
  ERSTE_ANM:        "Erst-Bestätigung",
  FINALE_ANM:       "Final-Bestätigung",
  ABLEHNUNG_ANM:    "Ablehnung",
  ANFORDERUNG_ECON: "Zustimmungsanfrage",
  ANFORDERUNG_ECP:  "Listanforderung",
  ECMPList:         "Zählpunktliste",
};

export function edaMessageTypeLabel(process: string, messageType: string): string {
  const code = process || messageType;
  return EDA_MESSAGE_TYPE_LABELS[code] ?? code;
}

// pending/sent/error are the same concept as on eda_processes.status, so they share
// label + color with EDA_PROCESS_STATUS_LABELS/STYLES; ack/processed only exist on messages.
export function edaMessageStatusLabelAndStyle(status: string): { label: string; cls: string } {
  if (status === "ack") return { label: "Quittiert", cls: "bg-green-50 text-green-700" };
  if (status === "processed") return { label: "Verarbeitet", cls: "bg-green-50 text-green-700" };
  return {
    label: EDA_PROCESS_STATUS_LABELS[status] ?? EDA_PROCESS_STATUS_LABELS.pending,
    cls: EDA_PROCESS_STATUS_STYLES[status] ?? "bg-yellow-50 text-yellow-700",
  };
}
