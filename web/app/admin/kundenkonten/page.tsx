"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import {
  listEEGs,
  getEEG,
  listMembers,
  updateEEG,
  bulkAssignKundenkonto,
  type EEG,
  type UpdateEEGRequest,
} from "@/lib/api";
import { KundenkontenTable } from "./KundenkontenTable";

interface Props {
  searchParams: Promise<{ success?: string; error?: string }>;
}

export default async function KundenkontenAdminPage({ searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");
  if (session.role !== "admin") redirect("/eegs");

  const { success: spSuccess, error: spError } = await searchParams;

  let eegs: EEG[] = [];
  let missingCounts: Record<string, number> = {};
  let loadError: string | null = null;

  try {
    eegs = await listEEGs(session.accessToken!);
    const counts = await Promise.all(
      eegs.map((eeg) =>
        listMembers(session.accessToken!, eeg.id)
          .then((members) => members.filter((m) => !m.kundenkonto).length)
          .catch(() => 0)
      )
    );
    missingCounts = Object.fromEntries(eegs.map((eeg, i) => [eeg.id, counts[i]]));
  } catch (err: unknown) {
    loadError = (err as { message?: string }).message || "Fehler beim Laden der Energiegemeinschaften.";
  }

  // Sets the per-EEG Kundenkonto-Startnummer. Each EEG is numbered independently —
  // there is no shared/org-wide sequence. Round-trips the current EEG so other
  // fields (most are not optional on the update endpoint) stay unchanged.
  async function setStartAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session || session.role !== "admin") return;
    const eegId = formData.get("eeg_id") as string;
    const start = parseInt(formData.get("kundenkonto_start") as string) || 0;
    let errMsg: string | null = null;
    try {
      const current = await getEEG(session.accessToken!, eegId);
      // GET returns eda_transition_date/gruendungsdatum as full RFC3339 timestamps,
      // but the update endpoint parses them strictly as YYYY-MM-DD — truncate before
      // round-tripping (mirrors the main EEG-Einstellungen form's hidden-input mirror).
      await updateEEG(session.accessToken!, eegId, {
        ...current,
        eda_transition_date: current.eda_transition_date ? current.eda_transition_date.substring(0, 10) : undefined,
        gruendungsdatum: current.gruendungsdatum ? current.gruendungsdatum.substring(0, 10) : undefined,
        kundenkonto_start: start,
      } as UpdateEEGRequest);
    } catch (err: unknown) {
      errMsg = (err as { message?: string }).message || "Speichern fehlgeschlagen.";
    }
    if (errMsg) {
      redirect(`/admin/kundenkonten?error=${encodeURIComponent(errMsg)}`);
    }
    redirect(`/admin/kundenkonten?success=${encodeURIComponent("Startnummer gespeichert.")}`);
  }

  // Assigns Kundenkonten for each SELECTED EEG independently — this loops the
  // single-EEG bulk-assign endpoint once per EEG, each with its own counter/start
  // value. There is deliberately no cross-EEG/org-wide sequence.
  async function assignAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session || session.role !== "admin") return;
    const selectedIds = formData.getAll("eeg_id") as string[];
    if (selectedIds.length === 0) {
      redirect(`/admin/kundenkonten?error=${encodeURIComponent("Bitte mindestens eine Energiegemeinschaft auswählen.")}`);
    }
    const results: string[] = [];
    let anyError = false;
    for (const eegId of selectedIds) {
      const eeg = eegs.find((e) => e.id === eegId);
      const name = eeg?.display_name || eeg?.name || eegId;
      try {
        const res = await bulkAssignKundenkonto(session.accessToken!, eegId);
        results.push(`${name}: ${res.assigned_count} vergeben`);
      } catch (err: unknown) {
        anyError = true;
        results.push(`${name}: Fehler — ${(err as { message?: string }).message || "unbekannt"}`);
      }
    }
    const summary = results.join(" · ");
    if (anyError) {
      redirect(`/admin/kundenkonten?error=${encodeURIComponent(summary)}`);
    }
    redirect(`/admin/kundenkonten?success=${encodeURIComponent(summary)}`);
  }

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Kundenkonten</h1>
        <p className="text-slate-500 mt-1 max-w-2xl">
          Pro Energiegemeinschaft (EEG/BEG/GEA) eine Startnummer festlegen und Kundenkonto-Nummern für
          Mitglieder ohne eigene Nummer vergeben. Jede Energiegemeinschaft wird unabhängig durchnummeriert —
          es gibt keinen organisationsweiten Zähler. Um über mehrere Gemeinschaften hinweg lückenlose
          Nummern zu erhalten, selbst passende Startwerte setzen (z.&nbsp;B. EEG&nbsp;A ab 210004, EEG&nbsp;B ab 210050).
        </p>
      </div>

      {spSuccess && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
          {decodeURIComponent(spSuccess)}
        </div>
      )}
      {(loadError || spError) && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {loadError || decodeURIComponent(spError!)}
        </div>
      )}

      <KundenkontenTable
        eegs={eegs}
        missingCounts={missingCounts}
        setStartAction={setStartAction}
        assignAction={assignAction}
      />
    </div>
  );
}
