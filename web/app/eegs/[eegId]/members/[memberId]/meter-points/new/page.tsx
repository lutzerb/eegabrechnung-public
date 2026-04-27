"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getMember, createMeterPoint } from "@/lib/api";
import { ValidatedInput } from "@/components/validated-input";
import Link from "next/link";

interface Props {
  params: Promise<{ eegId: string; memberId: string }>;
  searchParams: Promise<{ error?: string }>;
}

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export default async function NewMeterPointPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId, memberId } = await params;
  const { error: spError } = await searchParams;

  let eeg = null;
  let member = null;
  let loadError: string | null = null;

  try {
    [eeg, member] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      getMember(session.accessToken!, eegId, memberId),
    ]);
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    loadError = apiError.message || "Fehler beim Laden.";
  }

  if (loadError || !eeg || !member) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{loadError}</p>
        </div>
      </div>
    );
  }

  const edaConfigured = !!(eeg.eda_marktpartner_id && eeg.eda_netzbetreiber_id);
  const edaNetzbetreiberId = eeg.eda_netzbetreiber_id ?? "";

  // EDA Anmeldung: frühestens morgen, höchstens 30 Tage in der Zukunft
  const edaTomorrowStr = (() => {
    const d = new Date(Date.now() + 24 * 60 * 60 * 1000);
    return d.toISOString().slice(0, 10);
  })();
  const edaMaxDateStr = (() => {
    const d = new Date(Date.now() + 31 * 24 * 60 * 60 * 1000);
    return d.toISOString().slice(0, 10);
  })();

  async function createMeterPointAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;

    const zaehlpunkt = formData.get("zaehlpunkt") as string;
    const energierichtung = formData.get("energierichtung") as string;

    if (!zaehlpunkt || !energierichtung) {
      redirect(
        `/eegs/${eegId}/members/${memberId}/meter-points/new?error=${encodeURIComponent("Zählpunkt-ID und Energierichtung sind Pflichtfelder.")}`
      );
    }
    if (edaNetzbetreiberId && zaehlpunkt.length >= 8 && zaehlpunkt.substring(0, 8) !== edaNetzbetreiberId) {
      redirect(
        `/eegs/${eegId}/members/${memberId}/meter-points/new?error=${encodeURIComponent(`Zählpunkt-Präfix „${zaehlpunkt.substring(0, 8)}" passt nicht zum konfigurierten Netzbetreiber „${edaNetzbetreiberId}"`)}`
      );
    }

    const verteilungsmodell = formData.get("verteilungsmodell") as string;
    const zugeteilte_menge_str = formData.get("zugeteilte_menge_pct") as string;
    const registriert_seit = formData.get("registriert_seit") as string;
    const status = formData.get("status") as string;
    const generationType = formData.get("generation_type") as string;

    let saveError: string | null = null;
    try {
      await createMeterPoint(session.accessToken!, eegId, memberId, {
        zaehlpunkt,
        energierichtung,
        verteilungsmodell: verteilungsmodell || undefined,
        zugeteilte_menge_pct: zugeteilte_menge_str ? parseFloat(zugeteilte_menge_str) : undefined,
        status: status || undefined,
        registriert_seit: registriert_seit || undefined,
        generation_type: generationType || undefined,
      });
    } catch (err: unknown) {
      saveError = (err as { message?: string }).message || "Erstellen fehlgeschlagen.";
    }
    if (saveError) {
      redirect(`/eegs/${eegId}/members/${memberId}/meter-points/new?error=${encodeURIComponent(saveError)}`);
    }

    // Optional: trigger EDA Anmeldung immediately
    const edaAnmeldung = formData.get("eda_anmeldung") === "on";
    if (edaAnmeldung) {
      const validFrom = formData.get("eda_valid_from") as string;
      const shareType = (formData.get("eda_share_type") as string) || "GC";
      const factorStr = formData.get("eda_participation_factor") as string;
      const participationFactor = factorStr ? parseFloat(factorStr) : 100;

      try {
        await fetch(`${API}/api/v1/eegs/${eegId}/eda/anmeldung`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${session.accessToken}`,
          },
          body: JSON.stringify({
            zaehlpunkt,
            valid_from: validFrom || undefined,
            share_type: shareType,
            participation_factor: participationFactor,
          }),
        });
        // Non-fatal: EDA can always be triggered manually from the EDA page
      } catch {
        // ignore — meter point was created, EDA can be sent manually
      }
    }

    redirect(`/eegs/${eegId}/members/${memberId}?success=1`);
  }

  const inputClass =
    "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";
  const labelClass = "block text-sm font-medium text-slate-700 mb-1.5";

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">Energiegemeinschaften</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">{eeg.name}</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/members`} className="text-sm text-slate-500 hover:text-slate-700">Mitglieder</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/members/${memberId}`} className="text-sm text-slate-500 hover:text-slate-700">{member.name}</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Zählpunkt hinzufügen</span>
      </div>

      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900">Zählpunkt hinzufügen</h1>
        <p className="text-slate-500 mt-1">Neuen Zählpunkt für {member.name} erstellen.</p>
      </div>

      {spError && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{decodeURIComponent(spError)}</p>
        </div>
      )}

      <form action={createMeterPointAction} className="max-w-2xl space-y-4">
        {/* Zählpunkt Stammdaten */}
        <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-4">
          <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Stammdaten</h2>

          <div>
            <label className={labelClass}>
              Zählpunkt-ID <span className="text-red-500">*</span>
            </label>
            <ValidatedInput
              name="zaehlpunkt"
              placeholder="AT0010000000000000001000000000001"
              validatorName="zaehlpunkt"
              inputClassName={inputClass}
            />
            <p className="text-xs text-slate-400 mt-1">33-stellige Zählpunktnummer gemäß österreichischem Standard.</p>
          </div>

          <div>
            <label className={labelClass}>Energierichtung <span className="text-red-500">*</span></label>
            <select name="energierichtung" required className={inputClass}>
              <option value="">Bitte wählen</option>
              <option value="CONSUMPTION">Bezug</option>
              <option value="GENERATION">Einspeisung</option>
            </select>
          </div>

          <div>
            <label className={labelClass}>Einspeisungsart</label>
            <select name="generation_type" className={inputClass}>
              <option value="">— (Bezug / keine Angabe)</option>
              <option value="PV">Photovoltaik (PV)</option>
              <option value="Windkraft">Windkraft</option>
              <option value="Wasserkraft">Wasserkraft</option>
              <option value="Biomasse">Biomasse</option>
              <option value="Sonstige">Sonstige</option>
            </select>
          </div>

          <div>
            <label className={labelClass}>Verteilungsmodell</label>
            <select name="verteilungsmodell" className={inputClass}>
              <option value="">Bitte wählen</option>
              <option value="DYNAMIC">Dynamisch</option>
              <option value="STATIC">Statisch</option>
            </select>
          </div>

          <div>
            <label className={labelClass}>Zugeteilte Menge (%)</label>
            <div className="relative">
              <input
                type="number"
                name="zugeteilte_menge_pct"
                min="0"
                max="100"
                step="0.01"
                placeholder="0"
                className={`${inputClass} pr-8`}
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">%</span>
            </div>
            <p className="text-xs text-slate-400 mt-1">Nur relevant bei statischem Verteilungsmodell.</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelClass}>Status</label>
              <select name="status" className={inputClass}>
                <option value="NEW">Neu</option>
                <option value="ACTIVATED">Aktiviert</option>
              </select>
            </div>
            <div>
              <label className={labelClass}>Registriert seit</label>
              <input type="date" name="registriert_seit" className={inputClass} />
            </div>
          </div>
        </div>

        {/* EDA Anmeldung */}
        <div className={`rounded-xl border p-6 ${edaConfigured ? "bg-white border-slate-200" : "bg-slate-50 border-slate-200"}`}>
          <div className="flex items-start gap-3">
            <input
              type="checkbox"
              id="eda_anmeldung"
              name="eda_anmeldung"
              disabled={!edaConfigured}
              className="mt-0.5 h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 disabled:opacity-40"
            />
            <div className="flex-1">
              <label htmlFor="eda_anmeldung" className={`text-sm font-semibold ${edaConfigured ? "text-slate-900 cursor-pointer" : "text-slate-400"}`}>
                EDA-Anmeldung direkt auslösen (EC_EINZEL_ANM)
              </label>
              {edaConfigured ? (
                <p className="text-xs text-slate-500 mt-0.5">
                  Sendet sofort eine Anmeldung an den Netzbetreiber. Kann auch nachträglich über die EDA-Seite ausgelöst werden.
                </p>
              ) : (
                <p className="text-xs text-slate-400 mt-0.5">
                  Nicht verfügbar — Marktpartner-ID und Netzbetreiber-ID müssen zuerst in den{" "}
                  <Link href={`/eegs/${eegId}/settings?tab=eda`} className="underline hover:text-slate-600">EEG-Einstellungen</Link>{" "}
                  konfiguriert werden.
                </p>
              )}
            </div>
          </div>

          {edaConfigured && (
            <div className="mt-4 ml-7 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Gültig ab</label>
                  <input
                    type="date"
                    name="eda_valid_from"
                    defaultValue={edaTomorrowStr}
                    min={edaTomorrowStr}
                    max={edaMaxDateStr}
                    className={inputClass}
                  />
                  <p className="text-xs text-slate-400 mt-1">Frühestens morgen, höchstens 30 Tage in der Zukunft.</p>
                </div>
                <div>
                  <label className={labelClass}>Teilnahmefaktor (%)</label>
                  <div className="relative">
                    <input
                      type="number"
                      name="eda_participation_factor"
                      min="0"
                      max="100"
                      step="0.01"
                      defaultValue="100"
                      className={`${inputClass} pr-8`}
                    />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">%</span>
                  </div>
                </div>
              </div>
              <div>
                <label className={labelClass}>Anteilstyp</label>
                <select name="eda_share_type" className={inputClass}>
                  <option value="GC">GC — Gemeinschaft (Vollzuteilung, Standard)</option>
                  <option value="RC_R">RC_R — Residualeinspeiser</option>
                  <option value="RC_L">RC_L — Lastabhängig</option>
                  <option value="CC">CC — Konstant</option>
                </select>
              </div>
            </div>
          )}
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            className="px-6 py-2.5 bg-blue-700 text-white font-medium rounded-lg hover:bg-blue-800 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Zählpunkt erstellen
          </button>
          <Link
            href={`/eegs/${eegId}/members/${memberId}`}
            className="px-6 py-2.5 border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-50 transition-colors"
          >
            Abbrechen
          </Link>
        </div>
      </form>
    </div>
  );
}
