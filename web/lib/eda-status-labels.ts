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
