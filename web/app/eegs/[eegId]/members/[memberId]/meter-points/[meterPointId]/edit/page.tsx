"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getMember, getMeterPoint, updateMeterPoint } from "@/lib/api";
import Link from "next/link";
import { TeilnahmefaktorSection } from "./TeilnahmefaktorSection";

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

  async function updateMeterPointAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;

    const energierichtung = formData.get("energierichtung") as string;
    const verteilungsmodell = formData.get("verteilungsmodell") as string;
    const zugeteilte_menge_str = formData.get("zugeteilte_menge_pct") as string;
    const registriert_seit = formData.get("registriert_seit") as string;
    const status = formData.get("status") as string;
    const generation_type = formData.get("generation_type") as string;
    const notes = formData.get("notes") as string;

    try {
      await updateMeterPoint(session.accessToken!, eegId, meterPointId, {
        zaehlpunkt: mp!.zaehlpunkt,
        energierichtung,
        verteilungsmodell: verteilungsmodell || undefined,
        zugeteilte_menge_pct: zugeteilte_menge_str
          ? parseFloat(zugeteilte_menge_str)
          : undefined,
        status: status || undefined,
        registriert_seit: registriert_seit || undefined,
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

  const currentFactor = member.meter_points?.find((m) => m.id === meterPointId)?.participation_factor ?? 100;
  const isStatic = mp.verteilungsmodell === "STATIC";

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
            <select name="status" defaultValue={mp.status} className={inputClass}>
              <option value="NEW">Neu</option>
              <option value="ACTIVATED">Aktiviert</option>
            </select>
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
      </div>
    </div>
  );
}
