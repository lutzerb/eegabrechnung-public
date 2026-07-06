import { EDAProcessEvent, MeterPointRegistrationPeriod } from "@/lib/api";
import { parseEdaErrorCodes } from "@/lib/eda-error-codes";
import { EDA_PROCESS_STATUS_STYLES } from "@/lib/eda-status-labels";

export interface RegistrationTimelineEntry {
  at: string;
  label: string;
  cls: string;
  detail?: string;
}

// "completed" means different things depending on process type: for CM_REV_SP it's the
// normal successful terminal state (Abmeldung bestätigt). For EC_REQ_ONL it's only ever
// set when a *later* revoke retroactively closes an already-confirmed registration
// (see worker.go processCMRevoke) — it is not a fresh confirmation, so it must not be
// labeled the same as "confirmed" or it reads as if the Anmeldung succeeded again on
// the day it was actually revoked.
function processOutcome(
  processType: string,
  status: string,
  errorMsg: string
): { label: string; cls: string; detail?: string } | null {
  const isAnmeldung = processType === "EC_REQ_ONL";
  if (status === "rejected" || status === "error") {
    // rejected/error carries one or more gateway response codes (e.g. "ABLEHNUNG_ECON
    // response_code=104" or "response_codes=[182 99]") — translate them to the same
    // German descriptions shown on the EDA-processes page instead of the raw code(s).
    const codes = parseEdaErrorCodes(errorMsg);
    const detail = codes.length > 0
      ? codes.map((p) => `Code ${p.code}: ${p.label}`).join("; ")
      : errorMsg || undefined;
    return {
      label:
        status === "rejected"
          ? isAnmeldung ? "Anmeldung abgelehnt" : "Abmeldung abgelehnt"
          : isAnmeldung ? "Anmeldung fehlgeschlagen" : "Abmeldung fehlgeschlagen",
      cls: EDA_PROCESS_STATUS_STYLES[status],
      detail,
    };
  }
  if (isAnmeldung && status === "completed")
    return { label: "Anmeldung beendet (durch Widerruf)", cls: "bg-slate-600 text-white", detail: errorMsg || undefined };
  if (status === "confirmed" || status === "completed")
    return { label: isAnmeldung ? "Anmeldung bestätigt" : "Abmeldung bestätigt", cls: EDA_PROCESS_STATUS_STYLES[status], detail: errorMsg || undefined };
  return null;
}

// Merges confirmed registration periods with the individual EDA process attempts
// (sent/confirmed/rejected/error) that produced them into one chronological timeline —
// so e.g. a rejected attempt, the eventual confirmed Anmeldung, an unsolicited
// Netzbetreiber-side Abmeldung, and a subsequent re-registration all show up as
// distinct events instead of only the resulting confirmed period.
export function buildRegistrationTimeline(
  periods: MeterPointRegistrationPeriod[],
  processes: EDAProcessEvent[]
): RegistrationTimelineEntry[] {
  const entries: RegistrationTimelineEntry[] = [];

  for (const p of periods) {
    entries.push({
      at: p.registriert_seit,
      label: "Registriert",
      cls: "bg-green-100 text-green-700",
      detail: p.consent_id ? `Consent-ID ${p.consent_id}` : undefined,
    });
    if (p.abgemeldet_am) {
      entries.push({
        at: p.abgemeldet_am,
        label: "Abgemeldet",
        cls: "bg-slate-600 text-white",
        detail: p.reason || undefined,
      });
    }
  }

  for (const proc of processes) {
    const isAnmeldung = proc.process_type === "EC_REQ_ONL";
    entries.push({
      at: proc.initiated_at,
      label: isAnmeldung ? "Anmeldung gesendet" : "Abmeldung gesendet",
      cls: "bg-yellow-100 text-yellow-700",
    });
    if (proc.completed_at) {
      const outcome = processOutcome(proc.process_type, proc.status, proc.error_msg);
      if (outcome) {
        entries.push({
          at: proc.completed_at,
          label: outcome.label,
          cls: outcome.cls,
          detail: outcome.detail,
        });
      }
    }
  }

  entries.sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime());
  return entries;
}
