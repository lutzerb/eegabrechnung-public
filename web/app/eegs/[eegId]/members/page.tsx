import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { getEEG, listMembers, deleteMember, deleteMeterPoint, type Member } from "@/lib/api";
import Link from "next/link";
import { ConfirmDeleteButton } from "./ConfirmDeleteButton";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ success?: string; error?: string; q?: string; status?: string; stichtag?: string }>;
}

const STATUS_LABELS: Record<string, { label: string; color: string }> = {
  ACTIVE:     { label: "Aktiv",        color: "bg-green-100 text-green-700" },
  REGISTERED: { label: "Angemeldet",   color: "bg-blue-100 text-blue-700" },
  NEW:        { label: "Neu",          color: "bg-yellow-100 text-yellow-700" },
  INACTIVE:   { label: "Inaktiv",      color: "bg-slate-100 text-slate-500" },
};

export default async function MembersPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { q: qParam, status, stichtag: stichtagParam, success: successParam, error: errorParam } = await searchParams;
  const q = qParam || "";
  const statusFilter = status || "";
  const stichtag = stichtagParam || "";

  let eeg = null;
  let members: Member[] = [];
  let error: string | null = null;

  try {
    [eeg, members] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      listMembers(session.accessToken!, eegId, stichtag || undefined),
    ]);
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    error = apiError.message || "Fehler beim Laden der Mitglieder.";
  }

  // Client-side filter by q and status (avoids a second API call)
  const filtered = members.filter((m) => {
    const matchesStatus = !statusFilter || (m.status || "ACTIVE") === statusFilter;
    const search = q.toLowerCase();
    const matchesQ =
      !q ||
      (m.name1 || "").toLowerCase().includes(search) ||
      (m.name2 || "").toLowerCase().includes(search) ||
      (m.name || "").toLowerCase().includes(search) ||
      (m.email || "").toLowerCase().includes(search) ||
      (m.mitglieds_nr || "").toLowerCase().includes(search);
    return matchesStatus && matchesQ;
  });

  async function deleteMemberAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const memberId = formData.get("memberId") as string;
    try {
      await deleteMember(session.accessToken!, eegId, memberId);
      revalidatePath(`/eegs/${eegId}/members`);
    } catch (err: unknown) {
      const apiError = err as { message?: string };
      redirect(
        `/eegs/${eegId}/members?error=${encodeURIComponent(apiError.message || "Löschen fehlgeschlagen.")}`
      );
    }
  }

  async function deleteMeterPointAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const meterPointId = formData.get("meterPointId") as string;
    try {
      await deleteMeterPoint(session.accessToken!, eegId, meterPointId);
      revalidatePath(`/eegs/${eegId}/members`);
    } catch (err: unknown) {
      const apiError = err as { message?: string };
      redirect(
        `/eegs/${eegId}/members?error=${encodeURIComponent(apiError.message || "Zählpunkt löschen fehlgeschlagen.")}`
      );
    }
  }

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link
          href={`/eegs/${eegId}`}
          className="text-sm text-slate-500 hover:text-slate-700"
        >
          {eeg?.name || eegId}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Mitglieder</span>
      </div>

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Mitglieder</h1>
          <p className="text-slate-500 mt-1">
            {filtered.length} von {members.length} Mitglieder in {eeg?.name || "dieser Energiegemeinschaft"}
            {stichtag && (
              <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800">
                Stichtag: {new Date(stichtag + "T00:00:00").toLocaleDateString("de-AT")}
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-3">
          <a
            href={`/api/eegs/${eegId}/export/stammdaten`}
            className="px-4 py-2 text-sm font-medium border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors"
          >
            Exportieren
          </a>
          <Link
            href={`/eegs/${eegId}/import`}
            className="px-4 py-2 text-sm font-medium border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors"
          >
            Stammdaten importieren
          </Link>
          <Link
            href={`/eegs/${eegId}/members/new`}
            className="px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
          >
            + Mitglied hinzufügen
          </Link>
        </div>
      </div>

      {successParam && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
          <p className="font-medium">Gespeichert</p>
          <p className="text-sm mt-1">Die Änderungen wurden erfolgreich gespeichert.</p>
        </div>
      )}
      {(error || errorParam) && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">
            {error || (errorParam ? decodeURIComponent(errorParam) : "")}
          </p>
        </div>
      )}

      {/* Search + filter bar */}
      <form method="GET" className="mb-4 flex flex-wrap gap-3">
        <input
          type="text"
          name="q"
          defaultValue={q}
          placeholder="Suche nach Name, E-Mail, Mitgliedsnr…"
          className="flex-1 min-w-[200px] px-3 py-2 border border-slate-300 rounded-lg text-sm text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <select
          name="status"
          defaultValue={statusFilter}
          className="px-3 py-2 border border-slate-300 rounded-lg text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">Alle Status</option>
          <option value="ACTIVE">Aktiv</option>
          <option value="REGISTERED">Angemeldet</option>
          <option value="NEW">Neu</option>
          <option value="INACTIVE">Inaktiv</option>
        </select>
        <div className="flex items-center gap-2">
          <label className="text-sm text-slate-600 whitespace-nowrap">Stichtag</label>
          <input
            type="date"
            name="stichtag"
            defaultValue={stichtag}
            className="px-3 py-2 border border-slate-300 rounded-lg text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <button
          type="submit"
          className="px-4 py-2 text-sm font-medium bg-slate-100 text-slate-700 rounded-lg hover:bg-slate-200 transition-colors"
        >
          Suchen
        </button>
        {(q || statusFilter || stichtag) && (
          <a
            href={`/eegs/${eegId}/members`}
            className="px-4 py-2 text-sm font-medium text-slate-500 rounded-lg hover:bg-slate-100 transition-colors"
          >
            Zurücksetzen
          </a>
        )}
      </form>

      {filtered.length === 0 && !error ? (
        <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
          <p className="text-slate-600 font-medium">Keine Mitglieder gefunden.</p>
          {members.length === 0 && (
            <>
              <p className="text-slate-400 text-sm mt-1">
                Fügen Sie ein Mitglied hinzu oder importieren Sie Stammdaten.
              </p>
              <Link
                href={`/eegs/${eegId}/members/new`}
                className="mt-4 inline-block px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
              >
                + Mitglied hinzufügen
              </Link>
            </>
          )}
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="overflow-x-auto">
          <table className="w-full text-sm min-w-[600px]">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50">
                <th className="text-left px-6 py-3.5 font-medium text-slate-600">Mitglied</th>
                <th className="text-left px-6 py-3.5 font-medium text-slate-600">Mitgliedsnr.</th>
                <th className="text-left px-6 py-3.5 font-medium text-slate-600">Status</th>
                <th className="text-left px-6 py-3.5 font-medium text-slate-600">E-Mail</th>
                <th className="text-left px-6 py-3.5 font-medium text-slate-600">Zählpunkte</th>
                <th className="text-right px-6 py-3.5 font-medium text-slate-600">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {filtered.map((member) => {
                const st = member.status || "ACTIVE";
                const statusMeta = STATUS_LABELS[st] || { label: st, color: "bg-slate-100 text-slate-600" };
                return (
                  <tr key={member.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-6 py-4">
                      <Link
                        href={`/eegs/${eegId}/members/${member.id}`}
                        className="font-medium text-slate-900 hover:text-blue-700 transition-colors"
                      >
                        {member.name}
                      </Link>
                    </td>
                    <td className="px-6 py-4 text-slate-600 font-mono text-xs">
                      {member.member_number || member.mitglieds_nr || "—"}
                    </td>
                    <td className="px-6 py-4">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${statusMeta.color}`}>
                        {statusMeta.label}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-slate-600">
                      {member.email || "—"}
                    </td>
                    <td className="px-6 py-4">
                      {member.meter_points && member.meter_points.length > 0 ? (
                        <div className="space-y-1.5">
                          {member.meter_points.map((mp) => (
                            <div key={mp.id} className="flex items-center gap-2 flex-wrap">
                              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">
                                {mp.direction === "CONSUMPTION"
                                  ? "Bezug"
                                  : mp.direction === "GENERATION"
                                  ? "Einspeisung"
                                  : mp.direction}
                              </span>
                              {mp.direction === "GENERATION" && mp.generation_type && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-50 text-yellow-700">
                                  {mp.generation_type}
                                </span>
                              )}
                              <span className="font-mono text-xs text-slate-600">
                                {mp.meter_id}
                              </span>
                              {mp.participation_factor !== undefined && mp.participation_factor !== 100 && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-50 text-orange-700" title="Teilnahmefaktor">
                                  {mp.participation_factor}%
                                </span>
                              )}
                              <div className="flex items-center gap-1 ml-auto">
                                <Link
                                  href={`/eegs/${eegId}/members/${member.id}/meter-points/${mp.id}/edit`}
                                  className="px-2 py-0.5 text-xs text-blue-700 hover:bg-blue-50 rounded transition-colors"
                                >
                                  Bearbeiten
                                </Link>
                                <ConfirmDeleteButton
                                  action={deleteMeterPointAction}
                                  hiddenName="meterPointId"
                                  hiddenValue={mp.id}
                                  confirmMessage={`Zählpunkt "${mp.meter_id}" wirklich löschen?`}
                                  className="px-2 py-0.5 text-xs text-red-600 hover:bg-red-50 rounded transition-colors"
                                />
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <span className="text-slate-400 text-xs">Keine Zählpunkte</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Link
                          href={`/eegs/${eegId}/members/${member.id}/meter-points/new`}
                          className="px-3 py-1.5 text-xs font-medium text-slate-700 bg-slate-100 rounded-md hover:bg-slate-200 transition-colors"
                        >
                          + Zählpunkt
                        </Link>
                        <Link
                          href={`/eegs/${eegId}/members/${member.id}/edit`}
                          className="px-3 py-1.5 text-xs font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                        >
                          Bearbeiten
                        </Link>
                        <ConfirmDeleteButton
                          action={deleteMemberAction}
                          hiddenName="memberId"
                          hiddenValue={member.id}
                          confirmMessage={`Mitglied "${member.name}" wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.`}
                          className="px-3 py-1.5 text-xs font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                        />
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          </div>
          <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
            <p className="text-xs text-slate-500">
              {filtered.length} Mitglied{filtered.length !== 1 ? "er" : ""} &middot;{" "}
              {filtered.reduce((sum, m) => sum + (m.meter_points?.length || 0), 0)}{" "}
              Zählpunkt(e) gesamt
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
