import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { getEEG, listMembers, deleteMember, type Member } from "@/lib/api";
import Link from "next/link";
import { ConfirmDeleteButton } from "./ConfirmDeleteButton";
import { MeterPointStatusBadge } from "@/components/meter-point-status-badge";
import { MeterPointAddressToggle } from "@/components/mp-address-toggle";
import { formatAddress } from "@/lib/address";
import { EffectiveTariffCell } from "@/components/effective-tariff";

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
        <MeterPointAddressToggle />
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
        <div id="members-table" data-show-address="false" className="group bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="overflow-x-auto">
          <table className="w-full text-sm table-fixed min-w-[900px]">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50">
                <th className="text-left px-3 py-3.5 font-medium text-slate-600 w-[13%]">Mitglied</th>
                <th className="text-left px-1.5 py-3.5 font-medium text-slate-600 w-[56px]">Nr.</th>
                <th className="text-left px-1.5 py-3.5 font-medium text-slate-600 w-[92px]">Status</th>
                <th className="text-left px-3 py-3.5 font-medium text-slate-600 w-[13%]">E-Mail</th>
                <th className="text-left px-3 py-3.5 font-medium text-slate-600 w-[10%]">Tarifplan</th>
                <th className="text-left px-3 py-3.5 font-medium text-slate-600 w-[39%]">Zählpunkte</th>
                <th className="text-right px-3 py-3.5 font-medium text-slate-600 w-[13%]">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {filtered.map((member) => {
                const st = member.status || "ACTIVE";
                const statusMeta = STATUS_LABELS[st] || { label: st, color: "bg-slate-100 text-slate-600" };
                return (
                  <tr key={member.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-4 truncate">
                      <Link
                        href={`/eegs/${eegId}/members/${member.id}`}
                        className="font-medium text-slate-900 hover:text-blue-700 transition-colors"
                      >
                        {member.name}
                      </Link>
                    </td>
                    <td className="px-1.5 py-4 text-slate-600 font-mono text-xs truncate">
                      <div className="truncate">{member.member_number || member.mitglieds_nr || "—"}</div>
                      {member.kundenkonto && (
                        <div
                          className="truncate text-slate-400 text-[10px]"
                          title={`Kundenkonto ${member.kundenkonto}`}
                        >
                          KK {member.kundenkonto}
                        </div>
                      )}
                    </td>
                    <td className="px-1.5 py-4">
                      <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${statusMeta.color}`}>
                        {statusMeta.label}
                      </span>
                    </td>
                    <td className="px-3 py-4 text-slate-600 truncate" title={member.email}>
                      {member.email || "—"}
                    </td>
                    <td className="px-3 py-4 truncate">
                      <EffectiveTariffCell tariff={member.effective_tariff} />
                    </td>
                    <td className="px-3 py-4">
                      <div className="space-y-1.5">
                        {member.meter_points && member.meter_points.length > 0 ? (
                          member.meter_points.map((mp) => (
                            <div key={mp.id}>
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span className="inline-flex items-center justify-center w-[5.5rem] flex-shrink-0 px-1.5 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">
                                {mp.direction === "CONSUMPTION" ? "Bezug" : "Einspeisung"}
                              </span>
                              <div className="w-[5.5rem] flex-shrink-0">
                                <MeterPointStatusBadge
                                  registriertSeit={mp.registriert_seit}
                                  abgemeldetAm={mp.abgemeldet_am}
                                  anmeldungStatus={mp.anmeldung_status}
                                  abmeldungStatus={mp.abmeldung_status}
                                  fullWidth
                                />
                              </div>
                              <Link
                                href={`/eegs/${eegId}/members/${member.id}/meter-points/${mp.id}/edit`}
                                title="Zählpunkt bearbeiten"
                                className="font-mono text-xs text-slate-500 hover:text-blue-700 hover:underline transition-colors"
                              >
                                {mp.meter_id}
                              </Link>
                              {mp.participation_factor !== undefined && mp.participation_factor !== 100 && (
                                <span className="inline-flex items-center px-1 py-0.5 rounded text-xs font-medium bg-orange-50 text-orange-700 flex-shrink-0" title="Teilnahmefaktor">
                                  {mp.participation_factor}%
                                </span>
                              )}
                              {mp.direction === "GENERATION" && mp.generation_type && (
                                <span className="inline-flex items-center px-1 py-0.5 rounded text-xs font-medium bg-yellow-50 text-yellow-700 flex-shrink-0">
                                  {mp.generation_type}
                                </span>
                              )}
                            </div>
                            <div className="hidden group-data-[show-address=true]:block text-xs text-slate-500 mt-0.5 pl-[5.5rem] ml-2">
                              {formatAddress(mp) || <span className="text-slate-300">keine Adresse</span>}
                            </div>
                            </div>
                          ))
                        ) : (
                          <span className="text-slate-400 text-xs">Keine Zählpunkte</span>
                        )}
                        {eeg?.extra_meters_enabled && (
                          <div>
                            <Link
                              href={`/eegs/${eegId}/members/${member.id}/extra-meters`}
                              className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium transition-colors ${
                                (member.extra_meters_count || 0) > 0
                                  ? "bg-amber-50 text-amber-700 hover:bg-amber-100"
                                  : "text-slate-400 hover:bg-slate-100"
                              }`}
                            >
                              {(member.extra_meters_count || 0) > 0
                                ? `${member.extra_meters_count} Zusatzzähler`
                                : "+ Zusatzzähler"}
                            </Link>
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-3 py-4 text-right">
                      <div className="flex flex-wrap items-center justify-end gap-1">
                        <Link
                          href={`/eegs/${eegId}/members/${member.id}/meter-points/new`}
                          className="px-2 py-1 text-xs font-medium text-slate-700 bg-slate-100 rounded-md hover:bg-slate-200 transition-colors whitespace-nowrap"
                        >
                          + Zählpunkt
                        </Link>
                        <Link
                          href={`/eegs/${eegId}/members/${member.id}/edit`}
                          className="px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors whitespace-nowrap"
                        >
                          Bearbeiten
                        </Link>
                        <ConfirmDeleteButton
                          action={deleteMemberAction}
                          hiddenName="memberId"
                          hiddenValue={member.id}
                          confirmMessage={`Mitglied "${member.name}" wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.`}
                          className="px-2 py-1 text-xs font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors whitespace-nowrap"
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
