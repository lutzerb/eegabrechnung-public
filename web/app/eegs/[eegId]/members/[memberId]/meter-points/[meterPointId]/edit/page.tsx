"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getMember, getMeterPoint, getMeterPointHistory, updateMeterPoint, MeterPointHistory } from "@/lib/api";
import { buildRegistrationTimeline } from "./registration-timeline";
import Link from "next/link";
import { TeilnahmefaktorSection } from "./TeilnahmefaktorSection";
import { ZaehlpunktAbmeldenSection } from "./ZaehlpunktAbmeldenSection";
import { MeterPointStatusBadge } from "@/components/meter-point-status-badge";

interface Props {
  params: Promise<{ eegId: string; memberId: string; meterPointId: string }>;
  searchParams: Promise<{ error?: string }>;
}

export default async function EditMeterPointPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId, memberId, meterPointId } = await params;
  const { error: spError } = await searchParams;

  let eeg = null;
  let member = null;
  let mp = null;
  let loadError: string | null = null;

  try {
    [eeg, member, mp] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      getMember(session.accessToken!, eegId, memberId),
      getMeterPoint(session.accessToken!, eegId, meterPointId),
    ]);
  } catch (err: unknown) {
    loadError = (err as { message?: string }).message || "Fehler beim Laden.";
  }

  if (loadError || !eeg || !member || !mp) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{loadError}</p>
        </div>
      </div>
    );
  }

  let history: MeterPointHistory = { periods: [], processes: [] };
  try {
    history = await getMeterPointHistory(session.accessToken!, eegId, meterPointId);
  } catch {
    // Non-critical — the edit form still works without the history card.
  }
  const timeline = buildRegistrationTimeline(history.periods || [], history.processes || []);

  async function updateMeterPointAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;

    const energierichtung = formData.get("energierichtung") as string;
    const verteilungsmodell = formData.get("verteilungsmodell") as string;
    const zugeteilte_menge_str = formData.get("zugeteilte_menge_pct") as string;
    const registriert_seit = formData.get("registriert_seit") as string;
    const abgemeldet_am_raw = formData.get("abgemeldet_am") as string;
    const generation_type = formData.get("generation_type") as string;
    const notes = formData.get("notes") as string;

    const abgemeldet_am_clear = formData.get("abgemeldet_am_clear") as string;
    // "clear" checkbox takes priority; empty date string = keep existing
    const abgemeldet_am = abgemeldet_am_clear === "1" ? "clear"
      : abgemeldet_am_raw === "" ? undefined
      : abgemeldet_am_raw;

    try {
      await updateMeterPoint(session.accessToken!, eegId, meterPointId, {
        zaehlpunkt: mp!.zaehlpunkt,
        energierichtung,
        verteilungsmodell: verteilungsmodell || undefined,
        zugeteilte_menge_pct: zugeteilte_menge_str
          ? parseFloat(zugeteilte_menge_str)
          : undefined,
        registriert_seit: registriert_seit || undefined,
        abgemeldet_am,
        generation_type: generation_type || undefined,
        notes: notes ?? "",
      });
    } catch (err: unknown) {
      const msg = (err as { message?: string }).message || "Aktualisierung fehlgeschlagen.";
      redirect(`/eegs/${eegId}/members/${memberId}/meter-points/${meterPointId}/edit?error=${encodeURIComponent(msg)}`);
    }
    redirect(`/eegs/${eegId}/members?success=1`);
  }

  const inputClass =
    "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";
  const labelClass = "block text-sm font-medium text-slate-700 mb-1.5";

  const registriertSeitValue = mp.registriert_seit
    ? mp.registriert_seit.slice(0, 10)
    : "";

  const abgemeldetAmValue = mp.abgemeldet_am
    ? mp.abgemeldet_am.slice(0, 10)
    : "";

  const mpShort = member.meter_points?.find((m) => m.id === meterPointId);
  const currentFactor = mpShort?.participation_factor ?? 100;
  const isStatic = mp.verteilungsmodell === "STATIC";

  const anmeldungStatus = mpShort?.anmeldung_status;
  const abmeldungStatus = mpShort?.abmeldung_status;

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          {eeg.name}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/members`} className="text-sm text-slate-500 hover:text-slate-700">
          Mitglieder
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-500">{member.name}</span>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Zählpunkt bearbeiten</span>
      </div>

      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900">Zählpunkt bearbeiten</h1>
        <p className="text-slate-500 mt-1 font-mono text-sm">{mp.zaehlpunkt}</p>
      </div>

      {spError && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{decodeURIComponent(spError)}</p>
        </div>
      )}

      <div className="max-w-2xl bg-white rounded-xl border border-slate-200">
      <form action={updateMeterPointAction}>
        <div className="p-6 space-y-4">
          <div>
            <label className={labelClass}>Zählpunkt-ID</label>
            <input
              type="text"
              value={mp.zaehlpunkt}
              disabled
              className={`${inputClass} bg-slate-50 text-slate-400 cursor-not-allowed font-mono`}
            />
            <p className="text-xs text-slate-400 mt-1">
              Die Zählpunkt-ID kann nicht geändert werden.
            </p>
          </div>

          <div>
            <label className={labelClass}>
              Energierichtung <span className="text-red-500">*</span>
            </label>
            <select name="energierichtung" required defaultValue={mp.energierichtung} className={inputClass}>
              <option value="CONSUMPTION">Bezug</option>
              <option value="GENERATION">Einspeisung</option>
            </select>
          </div>

          <div>
            <label className={labelClass}>Erzeugungsart</label>
            <select
              name="generation_type"
              defaultValue={mp.generation_type || ""}
              className={inputClass}
            >
              <option value="">Keine (Bezug)</option>
              <option value="PV">PV</option>
              <option value="Windkraft">Windkraft</option>
              <option value="Wasserkraft">Wasserkraft</option>
            </select>
            <p className="text-xs text-slate-400 mt-1">
              Nur relevant für Einspeise-Zählpunkte.
            </p>
          </div>

          <div>
            <label className={labelClass}>Verteilungsmodell</label>
            <select
              id="verteilungsmodell-select"
              name="verteilungsmodell"
              defaultValue={mp.verteilungsmodell}
              className={inputClass}
            >
              <option value="">Bitte wählen</option>
              <option value="DYNAMIC">Dynamisch</option>
              <option value="STATIC">Statisch</option>
            </select>
          </div>

          <div id="zugeteilte-menge-row" style={{ display: isStatic ? undefined : "none" }}>
            <label className={labelClass}>Zugeteilte Menge (%)</label>
            <div className="relative">
              <input
                type="number"
                name="zugeteilte_menge_pct"
                min="0"
                max="100"
                step="0.01"
                defaultValue={mp.zugeteilte_menge_pct ?? 0}
                className={`${inputClass} pr-8`}
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">
                %
              </span>
            </div>
            <p className="text-xs text-slate-400 mt-1">
              Nur relevant bei statischem Verteilungsmodell.
            </p>
          </div>


          <div>
            <label className={labelClass}>Status</label>
            <div className="flex items-center gap-2 py-1">
              <MeterPointStatusBadge
                registriertSeit={mp.registriert_seit}
                abgemeldetAm={mp.abgemeldet_am}
                anmeldungStatus={anmeldungStatus}
                abmeldungStatus={abmeldungStatus}
              />
              {mp.registriert_seit && (
                <span className="text-sm text-slate-400">
                  seit {new Date(mp.registriert_seit).toLocaleDateString("de-AT")}
                </span>
              )}
            </div>
          </div>

          <div>
            <label className={labelClass}>Registriert seit</label>
            <input
              type="date"
              name="registriert_seit"
              defaultValue={registriertSeitValue}
              className={inputClass}
            />
          </div>

          <div>
            <label className={labelClass}>Abgemeldet am</label>
            <input
              type="date"
              name="abgemeldet_am"
              defaultValue={abgemeldetAmValue}
              className={inputClass}
            />
            {abgemeldetAmValue && (
              <label className="flex items-center gap-2 mt-2 cursor-pointer">
                <input type="checkbox" name="abgemeldet_am_clear" value="1" className="rounded" />
                <span className="text-sm text-slate-600">Datum löschen (Zählpunkt wieder als aktiv markieren)</span>
              </label>
            )}
            <p className="text-xs text-slate-400 mt-1">
              Datum setzen, damit dieser Zählpunkt für ein neues Mitglied wiederverwendet werden kann. Bei EDA-EEGs wird dieses Datum automatisch nach Abschluss des Widerrufs (CM_REV_SP) gesetzt.
            </p>
          </div>

          <div>
            <label className={labelClass}>Notizen</label>
            <textarea
              name="notes"
              defaultValue={mp.notes || ""}
              rows={3}
              placeholder="Interne Anmerkungen (z.B. Zähler getauscht am …, Kontaktperson beim NB …)"
              className={`${inputClass} resize-none`}
            />
            <p className="text-xs text-slate-400 mt-1">Nur intern sichtbar.</p>
          </div>
        </div>

        <script dangerouslySetInnerHTML={{ __html: `
          (function() {
            var sel = document.getElementById('verteilungsmodell-select');
            var row = document.getElementById('zugeteilte-menge-row');
            if (!sel || !row) return;
            sel.addEventListener('change', function() {
              row.style.display = sel.value === 'STATIC' ? '' : 'none';
            });
          })();
        `}} />

        <div className="px-6 pb-6 flex gap-3">
          <button
            type="submit"
            className="px-6 py-2.5 bg-blue-700 text-white font-medium rounded-lg hover:bg-blue-800 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Speichern
          </button>
          <Link
            href={`/eegs/${eegId}/members`}
            className="px-6 py-2.5 border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-50 transition-colors"
          >
            Abbrechen
          </Link>
        </div>
      </form>

      <div className="border-t border-slate-200 mx-6" />

      <div className="p-6">
        <TeilnahmefaktorSection
          eegId={eegId}
          zaehlpunkt={mp.zaehlpunkt}
          energyDirection={mp.energierichtung}
          currentFactor={currentFactor !== 100 ? currentFactor : undefined}
          currentValidFrom={member.meter_points?.find((m) => m.id === meterPointId)?.factor_valid_from}
        />
      </div>

      {eeg.eda_marktpartner_id && eeg.eda_netzbetreiber_id && mp.consent_id && !mp.abgemeldet_am && !eeg.is_demo && (
        <ZaehlpunktAbmeldenSection eegId={eegId} zaehlpunkt={mp.zaehlpunkt} />
      )}
      </div>

      <div className="max-w-2xl bg-white rounded-xl border border-slate-200 p-6 mt-6">
        <h2 className="font-semibold text-slate-900 text-sm mb-4">Registrierungshistorie</h2>
        {timeline.length === 0 ? (
          <p className="text-sm text-slate-500">Keine Anmeldung/Abmeldung erfasst.</p>
        ) : (
          <ol className="relative border-l border-slate-200 ml-3 space-y-4">
            {timeline.map((e, i) => (
              <li key={i} className="ml-4">
                <div className="absolute -left-1.5 w-3 h-3 rounded-full bg-slate-300 border-2 border-white" />
                <div className="flex items-center gap-2 mb-0.5 flex-wrap">
                  <span className={`text-xs font-medium px-1.5 py-0.5 rounded ${e.cls}`}>
                    {e.label}
                  </span>
                  <span className="text-xs text-slate-500">
                    {new Date(e.at).toLocaleDateString("de-AT")}
                  </span>
                </div>
                {e.detail && (
                  <p className="text-xs text-slate-500 italic">{e.detail}</p>
                )}
              </li>
            ))}
          </ol>
        )}
      </div>
    </div>
  );
}
