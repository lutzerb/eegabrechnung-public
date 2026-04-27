"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, createMember } from "@/lib/api";
import { ValidatedInput } from "@/components/validated-input";
import Link from "next/link";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ error?: string }>;
}

export default async function NewMemberPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { error: spError } = await searchParams;

  let eeg = null;
  let loadError: string | null = null;

  try {
    eeg = await getEEG(session.accessToken!, eegId);
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    loadError = apiError.message || "Fehler beim Laden der Energiegemeinschaft.";
  }

  if (loadError || !eeg) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{loadError}</p>
        </div>
      </div>
    );
  }

  async function createMemberAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;

    const name1 = formData.get("name1") as string;

    if (!name1) {
      redirect(
        `/eegs/${eegId}/members/new?error=${encodeURIComponent("Name ist Pflichtfeld.")}`
      );
    }

    const vatMode = formData.get("vat_mode") as string;
    let use_vat: boolean | null = null;
    let vat_pct: number | null = null;
    if (vatMode === "yes") {
      use_vat = true;
      vat_pct = parseFloat(formData.get("vat_pct") as string) || null;
    } else if (vatMode === "no") {
      use_vat = false;
    }

    let saveError: string | null = null;
    try {
      await createMember(session.accessToken!, eegId, {
        mitglieds_nr: (formData.get("mitglieds_nr") as string) || undefined,
        name1,
        name2: (formData.get("name2") as string) || undefined,
        email: (formData.get("email") as string) || undefined,
        iban: (formData.get("iban") as string) || undefined,
        strasse: (formData.get("strasse") as string) || undefined,
        plz: (formData.get("plz") as string) || undefined,
        ort: (formData.get("ort") as string) || undefined,
        business_role: (formData.get("business_role") as string) || undefined,
        uid_nummer: (formData.get("uid_nummer") as string) || undefined,
        use_vat,
        vat_pct,
        beitritts_datum: (formData.get("beitritts_datum") as string) || undefined,
        austritts_datum: (formData.get("austritts_datum") as string) || undefined,
      });
    } catch (err: unknown) {
      saveError = (err as { message?: string }).message || "Erstellen fehlgeschlagen.";
    }
    if (saveError) {
      redirect(`/eegs/${eegId}/members/new?error=${encodeURIComponent(saveError)}`);
    }
    redirect(`/eegs/${eegId}/members?success=1`);
  }

  const inputClass =
    "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";
  const labelClass = "block text-sm font-medium text-slate-700 mb-1.5";

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
          {eeg.name}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link
          href={`/eegs/${eegId}/members`}
          className="text-sm text-slate-500 hover:text-slate-700"
        >
          Mitglieder
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Neu</span>
      </div>

      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900">Neues Mitglied</h1>
        <p className="text-slate-500 mt-1">
          Neues Mitglied für {eeg.name} erstellen.
        </p>
      </div>

      {spError && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{decodeURIComponent(spError)}</p>
        </div>
      )}

      <form action={createMemberAction} className="max-w-2xl">
        <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelClass}>
                Vorname / Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                name="name1"
                required
                placeholder="Max"
                className={inputClass}
              />
            </div>
            <div>
              <label className={labelClass}>Nachname</label>
              <input
                type="text"
                name="name2"
                placeholder="Mustermann"
                className={inputClass}
              />
            </div>
          </div>

          <div>
            <label className={labelClass}>Mitgliedsnummer</label>
            <input
              type="text"
              name="mitglieds_nr"
              placeholder="automatisch vergeben"
              className={inputClass}
            />
            <p className="text-xs text-slate-500 mt-1">
              Leer lassen für automatische Vergabe
            </p>
          </div>

          <div>
            <label className={labelClass}>E-Mail</label>
            <input
              type="email"
              name="email"
              placeholder="max@example.com"
              className={inputClass}
            />
          </div>

          <div>
            <label className={labelClass}>IBAN</label>
            <ValidatedInput
              name="iban"
              placeholder="AT12 3456 7890 1234 5678"
              validatorName="iban"
              inputClassName={inputClass}
            />
          </div>

          <div>
            <label className={labelClass}>Straße</label>
            <input
              type="text"
              name="strasse"
              placeholder="Hauptstraße 1"
              className={inputClass}
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className={labelClass}>PLZ</label>
              <input
                type="text"
                name="plz"
                placeholder="1010"
                className={inputClass}
              />
            </div>
            <div className="col-span-2">
              <label className={labelClass}>Ort</label>
              <input
                type="text"
                name="ort"
                placeholder="Wien"
                className={inputClass}
              />
            </div>
          </div>

          <div>
            <label className={labelClass}>Unternehmensart</label>
            <select name="business_role" className={inputClass}>
              <option value="privat">Privatperson</option>
              <option value="kleinunternehmer">Kleinunternehmer</option>
              <option value="verein">Verein</option>
              <option value="landwirt_pauschaliert">Landwirt (pauschaliert, § 22 UStG)</option>
              <option value="landwirt">Landwirt (buchführungspflichtig)</option>
              <option value="unternehmen">Unternehmen</option>
              <option value="gemeinde_bga">Gemeinde (BgA)</option>
              <option value="gemeinde_hoheitlich">Gemeinde (hoheitlich)</option>
            </select>
          </div>

          <div>
            <label className={labelClass}>UID-Nummer</label>
            <ValidatedInput
              name="uid_nummer"
              placeholder="ATU12345678"
              validatorName="uid_nummer"
              inputClassName={inputClass}
            />
            <p className="text-xs text-slate-500 mt-1">
              Wenn angegeben → Reverse Charge auf Gutschriften (§ 19 UStG)
            </p>
          </div>

          <div>
            <label className={labelClass}>USt. auf Gutschriften</label>
            <select name="vat_mode" className={inputClass}>
              <option value="inherit">Automatisch (laut Unternehmensart)</option>
              <option value="yes">USt-pflichtig (manuell)</option>
              <option value="no">Nicht USt-pflichtig (manuell)</option>
            </select>
            <p className="text-xs text-slate-500 mt-1">
              Gilt nur für Einspeisvergütungen (Gutschriften) — nicht für Verbrauchsrechnungen
            </p>
          </div>

          <div>
            <label className={labelClass}>USt.-Satz (%) — nur bei manuell USt-pflichtig</label>
            <div className="relative">
              <input
                type="number"
                name="vat_pct"
                step="0.1"
                min="0"
                max="100"
                placeholder="20"
                className={`${inputClass} pr-8`}
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">%</span>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelClass}>Beitrittsdatum</label>
              <input
                type="date"
                name="beitritts_datum"
                className={inputClass}
              />
            </div>
            <div>
              <label className={labelClass}>Austrittsdatum</label>
              <input
                type="date"
                name="austritts_datum"
                className={inputClass}
              />
            </div>
          </div>
        </div>

        <div className="mt-6 flex gap-3">
          <button
            type="submit"
            className="px-6 py-2.5 bg-blue-700 text-white font-medium rounded-lg hover:bg-blue-800 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Mitglied erstellen
          </button>
          <Link
            href={`/eegs/${eegId}/members`}
            className="px-6 py-2.5 border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-50 transition-colors"
          >
            Abbrechen
          </Link>
        </div>
      </form>
    </div>
  );
}
