"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, updateEEG, UpdateEEGRequest } from "@/lib/api";
import { ValidatedInput } from "@/components/validated-input";
import { BackupRestoreSection } from "@/components/backup-restore-section";
import { DeleteEEGSection } from "@/components/delete-eeg-section";
import LogoUpload from "./LogoUpload";
import Link from "next/link";
import { revalidatePath } from "next/cache";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ success?: string; error?: string; tab?: string }>;
}

const TABS = [
  { key: "allgemein", label: "Allgemein" },
  { key: "rechnungen", label: "Rechnungen" },
  { key: "sepa", label: "SEPA" },
  { key: "eda", label: "EDA" },
  { key: "onboarding", label: "Onboarding" },
  { key: "abrechnung", label: "Auto-Abrechnung" },
  { key: "luecken", label: "Lücken-Alarm" },
  { key: "portal", label: "Mitgliederportal" },
  { key: "system", label: "System" },
] as const;

type TabKey = (typeof TABS)[number]["key"];

export default async function EEGSettingsPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { tab, success: successParam, error: errorParam } = await searchParams;
  const activeTab: TabKey = (tab as TabKey) || "allgemein";

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

  async function saveSettings(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;

    const tab = formData.get("_tab") as string || "allgemein";

    const data: UpdateEEGRequest = {
      name: formData.get("name") as string || undefined,
      netzbetreiber: formData.get("netzbetreiber") as string || undefined,
      energy_price: parseFloat(formData.get("energy_price") as string) || 0,
      producer_price: parseFloat(formData.get("producer_price") as string) || 0,
      use_vat: formData.get("use_vat") === "yes",
      vat_pct: parseFloat(formData.get("vat_pct") as string) || 20,
      meter_fee_eur: parseFloat(formData.get("meter_fee_eur") as string) || 0,
      free_kwh: parseFloat(formData.get("free_kwh") as string) || 0,
      discount_pct: parseFloat(formData.get("discount_pct") as string) || 0,
      participation_fee_eur: parseFloat(formData.get("participation_fee_eur") as string) || 0,
      billing_period: formData.get("billing_period") as string || "monthly",
      invoice_number_prefix: (formData.get("invoice_number_prefix") as string) || "INV",
      invoice_number_digits: parseInt(formData.get("invoice_number_digits") as string) || 5,
      invoice_number_start: parseInt(formData.get("invoice_number_start") as string) || 1,
      invoice_pre_text: (formData.get("invoice_pre_text") as string) || "",
      invoice_post_text: (formData.get("invoice_post_text") as string) || "",
      invoice_footer_text: (formData.get("invoice_footer_text") as string) || "",
      generate_credit_notes: formData.get("generate_credit_notes") === "on",
      credit_note_number_prefix: (formData.get("credit_note_number_prefix") as string) || "GS",
      credit_note_number_digits: parseInt(formData.get("credit_note_number_digits") as string) || 5,
      iban: (formData.get("iban") as string) || "",
      bic: (formData.get("bic") as string) || "",
      sepa_creditor_id: (formData.get("sepa_creditor_id") as string) || "",
      sepa_pre_notification_days: parseInt(formData.get("sepa_pre_notification_days") as string) || 14,
      gemeinschaft_id: (formData.get("gemeinschaft_id") as string) || "",
      eda_marktpartner_id: (formData.get("eda_marktpartner_id") as string) || "",
      eda_netzbetreiber_id: (formData.get("eda_netzbetreiber_id") as string) || "",
      eda_transition_date: (formData.get("eda_transition_date") as string) || undefined,
      accounting_revenue_account: parseInt(formData.get("accounting_revenue_account") as string) || 4000,
      accounting_expense_account: parseInt(formData.get("accounting_expense_account") as string) || 5000,
      accounting_debitor_prefix: parseInt(formData.get("accounting_debitor_prefix") as string) || 10000,
      datev_consultant_nr: (formData.get("datev_consultant_nr") as string) || "",
      datev_client_nr: (formData.get("datev_client_nr") as string) || "",
      strasse: (formData.get("strasse") as string) || "",
      plz: (formData.get("plz") as string) || "",
      ort: (formData.get("ort") as string) || "",
      uid_nummer: (formData.get("uid_nummer") as string) || "",
      gruendungsdatum: (formData.get("gruendungsdatum") as string) || undefined,
      onboarding_contract_text: (formData.get("onboarding_contract_text") as string) || "",
      // Per-EEG credentials — only sent when non-empty (empty = keep existing)
      eda_imap_host: (formData.get("eda_imap_host") as string) || undefined,
      eda_imap_user: (formData.get("eda_imap_user") as string) || undefined,
      eda_imap_password: (formData.get("eda_imap_password") as string) || undefined,
      eda_smtp_host: (formData.get("eda_smtp_host") as string) || undefined,
      eda_smtp_user: (formData.get("eda_smtp_user") as string) || undefined,
      eda_smtp_password: (formData.get("eda_smtp_password") as string) || undefined,
      eda_smtp_from: (formData.get("eda_smtp_from") as string) || undefined,
      smtp_host: (formData.get("smtp_host") as string) || undefined,
      smtp_user: (formData.get("smtp_user") as string) || undefined,
      smtp_password: (formData.get("smtp_password") as string) || undefined,
      smtp_from: (formData.get("smtp_from") as string) || undefined,
      // Auto-billing
      auto_billing_enabled: formData.get("auto_billing_enabled") === "on",
      auto_billing_day_of_month: parseInt(formData.get("auto_billing_day_of_month") as string) || 0,
      auto_billing_period: (formData.get("auto_billing_period") as string) || "monthly",
      // Gap alert
      gap_alert_enabled: formData.get("gap_alert_enabled") === "on",
      gap_alert_threshold_days: parseInt(formData.get("gap_alert_threshold_days") as string) || 5,
      // Member portal
      portal_show_full_energy: formData.get("portal_show_full_energy") === "on",
    };

    let saveError: string | null = null;
    try {
      await updateEEG(session.accessToken!, eegId, data);
      revalidatePath(`/eegs/${eegId}/settings`);
      revalidatePath(`/eegs/${eegId}`);
    } catch (err: unknown) {
      saveError = (err as { message?: string }).message || "Speichern fehlgeschlagen.";
    }
    if (saveError) {
      redirect(`/eegs/${eegId}/settings?tab=${tab}&error=${encodeURIComponent(saveError)}`);
    }
    redirect(`/eegs/${eegId}/settings?tab=${tab}&success=1`);
  }

  const inputClass =
    "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";
  const labelClass = "block text-sm font-medium text-slate-700 mb-1.5";

  const tabHref = (key: string) =>
    `/eegs/${eegId}/settings?tab=${key}`;

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
        <span className="text-sm text-slate-900 font-medium">Einstellungen</span>
      </div>

      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Einstellungen</h1>
        <p className="text-slate-500 mt-1">
          Abrechnungsparameter für {eeg.name} bearbeiten.
        </p>
      </div>

      {/* Success / Error feedback */}
      {successParam && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
          <p className="font-medium">Gespeichert</p>
          <p className="text-sm mt-1">Die Einstellungen wurden erfolgreich gespeichert.</p>
        </div>
      )}
      {errorParam && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{decodeURIComponent(errorParam)}</p>
        </div>
      )}

      {/* Tab bar */}
      <div className="flex gap-1 mb-6 border-b border-slate-200">
        {TABS.map((tab) => (
          <Link
            key={tab.key}
            href={tabHref(tab.key)}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors border-b-2 -mb-px ${
              activeTab === tab.key
                ? "border-blue-600 text-blue-700 bg-white"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:bg-slate-50"
            }`}
          >
            {tab.label}
          </Link>
        ))}
      </div>

      <form action={saveSettings} className="max-w-2xl space-y-6">
        {/* Hidden field to track active tab */}
        <input type="hidden" name="_tab" value={activeTab} />

        {/* Hidden fields for values not in current tab */}
        {activeTab !== "allgemein" && (
          <>
            <input type="hidden" name="name" value={eeg.name} />
            <input type="hidden" name="netzbetreiber" value={eeg.netzbetreiber || ""} />
            <input type="hidden" name="gruendungsdatum" value={eeg.gruendungsdatum ? eeg.gruendungsdatum.substring(0, 10) : ""} />
            <input type="hidden" name="strasse" value={eeg.strasse || ""} />
            <input type="hidden" name="plz" value={eeg.plz || ""} />
            <input type="hidden" name="ort" value={eeg.ort || ""} />
            <input type="hidden" name="uid_nummer" value={eeg.uid_nummer || ""} />
            <input type="hidden" name="billing_period" value={eeg.billing_period || "monthly"} />
            <input type="hidden" name="accounting_revenue_account" value={String((eeg as any).accounting_revenue_account || 4000)} />
            <input type="hidden" name="accounting_expense_account" value={String((eeg as any).accounting_expense_account || 5000)} />
            <input type="hidden" name="accounting_debitor_prefix" value={String((eeg as any).accounting_debitor_prefix || 10000)} />
            <input type="hidden" name="datev_consultant_nr" value={(eeg as any).datev_consultant_nr || ""} />
            <input type="hidden" name="datev_client_nr" value={(eeg as any).datev_client_nr || ""} />
          </>
        )}
        {activeTab !== "rechnungen" && (
          <>
            <input type="hidden" name="use_vat" value={eeg.use_vat ? "yes" : "no"} />
            <input type="hidden" name="vat_pct" value={String(eeg.vat_pct)} />
            <input type="hidden" name="meter_fee_eur" value={String(eeg.meter_fee_eur)} />
            <input type="hidden" name="free_kwh" value={String(eeg.free_kwh)} />
            <input type="hidden" name="discount_pct" value={String(eeg.discount_pct)} />
            <input type="hidden" name="participation_fee_eur" value={String(eeg.participation_fee_eur)} />
            <input type="hidden" name="invoice_number_prefix" value={eeg.invoice_number_prefix || "INV"} />
            <input type="hidden" name="invoice_number_digits" value={String(eeg.invoice_number_digits || 5)} />
            <input type="hidden" name="invoice_number_start" value={String(eeg.invoice_number_start || 1)} />
            <input type="hidden" name="invoice_pre_text" value={eeg.invoice_pre_text || ""} />
            <input type="hidden" name="invoice_post_text" value={eeg.invoice_post_text || ""} />
            <input type="hidden" name="invoice_footer_text" value={eeg.invoice_footer_text || ""} />
            <input type="hidden" name="generate_credit_notes" value={eeg.generate_credit_notes ? "on" : "off"} />
            <input type="hidden" name="credit_note_number_prefix" value={eeg.credit_note_number_prefix || "GS"} />
            <input type="hidden" name="credit_note_number_digits" value={String(eeg.credit_note_number_digits || 5)} />
          </>
        )}
        {activeTab !== "sepa" && (
          <>
            <input type="hidden" name="iban" value={eeg.iban || ""} />
            <input type="hidden" name="bic" value={eeg.bic || ""} />
            <input type="hidden" name="sepa_creditor_id" value={eeg.sepa_creditor_id || ""} />
            <input type="hidden" name="sepa_pre_notification_days" value={String(eeg.sepa_pre_notification_days ?? 14)} />
          </>
        )}
        {activeTab !== "eda" && (
          <>
            <input type="hidden" name="gemeinschaft_id" value={eeg.gemeinschaft_id || ""} />
            <input type="hidden" name="eda_marktpartner_id" value={eeg.eda_marktpartner_id || ""} />
            <input type="hidden" name="eda_netzbetreiber_id" value={eeg.eda_netzbetreiber_id || ""} />
            <input type="hidden" name="eda_transition_date" value={eeg.eda_transition_date ? eeg.eda_transition_date.substring(0, 10) : ""} />
          </>
        )}
        {activeTab !== "onboarding" && (
          <input type="hidden" name="onboarding_contract_text" value={eeg.onboarding_contract_text || ""} />
        )}
        {activeTab !== "abrechnung" && (
          <>
            <input type="hidden" name="auto_billing_enabled" value={eeg.auto_billing_enabled ? "on" : ""} />
            <input type="hidden" name="auto_billing_day_of_month" value={String(eeg.auto_billing_day_of_month ?? 0)} />
            <input type="hidden" name="auto_billing_period" value={eeg.auto_billing_period || "monthly"} />
          </>
        )}
        {activeTab !== "luecken" && (
          <>
            <input type="hidden" name="gap_alert_enabled" value={(eeg as any).gap_alert_enabled !== false ? "on" : ""} />
            <input type="hidden" name="gap_alert_threshold_days" value={String((eeg as any).gap_alert_threshold_days ?? 5)} />
          </>
        )}
        {activeTab !== "portal" && (
          <input type="hidden" name="portal_show_full_energy" value={(eeg as any).portal_show_full_energy !== false ? "on" : ""} />
        )}
        {/* energy_price / producer_price always hidden (managed in tariffs) */}
        <input type="hidden" name="energy_price" value={String(eeg.energy_price)} />
        <input type="hidden" name="producer_price" value={String(eeg.producer_price)} />

        {/* ── TAB: ALLGEMEIN ─────────────────────────────────────── */}
        {activeTab === "allgemein" && (
          <>
            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-4">Allgemein</h2>
              <div className="space-y-4">
                <div>
                  <label className={labelClass}>Name der Energiegemeinschaft</label>
                  <input type="text" name="name" defaultValue={eeg.name} className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Netzbetreiber</label>
                  <input type="text" name="netzbetreiber" defaultValue={eeg.netzbetreiber} className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Gründungsdatum</label>
                  <input
                    type="date"
                    name="gruendungsdatum"
                    defaultValue={eeg.gruendungsdatum ? eeg.gruendungsdatum.substring(0, 10) : ""}
                    max={new Date().toISOString().substring(0, 10)}
                    className={inputClass}
                  />
                  <p className="text-xs text-slate-400 mt-1">
                    Tage vor diesem Datum werden in der Datenverfügbarkeit nicht als fehlend gewertet.
                  </p>
                </div>
                <div>
                  <label className={labelClass}>Abrechnungsperiode</label>
                  <select name="billing_period" defaultValue={eeg.billing_period} className={inputClass}>
                    <option value="monthly">Monatlich</option>
                    <option value="quarterly">Vierteljährlich</option>
                    <option value="semiannual">Halbjährlich</option>
                    <option value="annual">Jährlich</option>
                  </select>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-1">Rechnungssteller (§11 UStG)</h2>
              <p className="text-xs text-slate-500 mb-4">
                Adresse und UID-Nummer der EEG — erscheint auf allen Rechnungen und Gutschriften.
              </p>
              <div className="space-y-4">
                <div>
                  <label className={labelClass}>Straße und Hausnummer</label>
                  <input type="text" name="strasse" defaultValue={eeg.strasse || ""} placeholder="Musterstraße 1" className={inputClass} />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={labelClass}>PLZ</label>
                    <input type="text" name="plz" defaultValue={eeg.plz || ""} placeholder="1010" className={inputClass} />
                  </div>
                  <div>
                    <label className={labelClass}>Ort</label>
                    <input type="text" name="ort" defaultValue={eeg.ort || ""} placeholder="Wien" className={inputClass} />
                  </div>
                </div>
                <div>
                  <label className={labelClass}>UID-Nummer (optional)</label>
                  <input type="text" name="uid_nummer" defaultValue={eeg.uid_nummer || ""} placeholder="ATU12345678" className={inputClass} />
                  <p className="text-xs text-slate-400 mt-1">
                    Nur ausfüllen wenn die EEG umsatzsteuerrechtlich registriert ist.
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-1">Buchhaltung / DATEV</h2>
              <p className="text-xs text-slate-500 mb-4">
                Kontonummern für den{" "}
                <a href={`/eegs/${eegId}/accounting`} className="text-blue-600 hover:underline">Buchhaltungsexport</a>.
              </p>
              <div className="space-y-4">
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className={labelClass}>Erlöskonto</label>
                    <input type="number" name="accounting_revenue_account" defaultValue={(eeg as any).accounting_revenue_account || 4000} className={inputClass} />
                    <p className="text-xs text-slate-400 mt-1">Verbrauchsrechnungen</p>
                  </div>
                  <div>
                    <label className={labelClass}>Aufwandskonto</label>
                    <input type="number" name="accounting_expense_account" defaultValue={(eeg as any).accounting_expense_account || 5000} className={inputClass} />
                    <p className="text-xs text-slate-400 mt-1">Gutschriften Einspeisung</p>
                  </div>
                  <div>
                    <label className={labelClass}>Debitorenbasis</label>
                    <input type="number" name="accounting_debitor_prefix" defaultValue={(eeg as any).accounting_debitor_prefix || 10000} className={inputClass} />
                    <p className="text-xs text-slate-400 mt-1">+ Mitgliedsnummer</p>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={labelClass}>DATEV Beraternummer</label>
                    <input type="text" name="datev_consultant_nr" defaultValue={(eeg as any).datev_consultant_nr || ""} placeholder="z.B. 12345" className={inputClass} />
                  </div>
                  <div>
                    <label className={labelClass}>DATEV Mandantennummer</label>
                    <input type="text" name="datev_client_nr" defaultValue={(eeg as any).datev_client_nr || ""} placeholder="z.B. 1" className={inputClass} />
                  </div>
                </div>
              </div>
            </div>
          </>
        )}

        {/* ── TAB: RECHNUNGEN ────────────────────────────────────── */}
        {activeTab === "rechnungen" && (
          <>
            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-4">Abrechnungsgebühren</h2>
              <p className="text-xs text-slate-500 mb-4">
                Arbeitspreise (Bezug/Einspeisung) werden im{" "}
                <a href={`/eegs/${eegId}/tariffs`} className="text-blue-600 hover:underline">Tarifplan</a>{" "}
                festgelegt.
              </p>
              <div className="space-y-4">
                <div>
                  <label className={labelClass}>Freikontingent (kWh/Periode)</label>
                  <div className="relative">
                    <input type="number" name="free_kwh" step="0.1" min="0" defaultValue={eeg.free_kwh} className={`${inputClass} pr-12`} />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">kWh</span>
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Rabatt (%)</label>
                  <div className="relative">
                    <input type="number" name="discount_pct" step="0.1" min="0" max="100" defaultValue={eeg.discount_pct} className={`${inputClass} pr-8`} />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">%</span>
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Zählpunktgebühr (EUR/Periode)</label>
                  <div className="relative">
                    <input type="number" name="meter_fee_eur" step="0.01" min="0" defaultValue={eeg.meter_fee_eur} className={`${inputClass} pr-12`} />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">EUR</span>
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Mitgliedsbeitrag (EUR/Periode)</label>
                  <div className="relative">
                    <input type="number" name="participation_fee_eur" step="0.01" min="0" defaultValue={eeg.participation_fee_eur} className={`${inputClass} pr-12`} />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">EUR</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-1">Umsatzsteuer der EEG</h2>
              <p className="text-xs text-slate-500 mb-4">Betrifft die Verbrauchsrechnungen an Mitglieder. Gutschriften an Erzeuger folgen immer den individuellen Regeln des Mitglieds (§ 6, § 19, § 22 UStG).</p>
              <div className="space-y-3">
                <label className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 cursor-pointer has-[:checked]:border-blue-500 has-[:checked]:bg-blue-50">
                  <input
                    type="radio"
                    name="use_vat"
                    value="no"
                    defaultChecked={!eeg.use_vat}
                    className="mt-0.5 h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  <div>
                    <p className="text-sm font-medium text-slate-800">Kleinunternehmer gem. § 6 Abs. 1 Z 27 UStG</p>
                    <p className="text-xs text-slate-500 mt-0.5">Keine USt auf Verbrauchsrechnungen — Hinweis &ldquo;steuerbefreit gem. § 6 UStG&rdquo; wird ausgewiesen.</p>
                  </div>
                </label>
                <label className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 cursor-pointer has-[:checked]:border-blue-500 has-[:checked]:bg-blue-50">
                  <input
                    type="radio"
                    name="use_vat"
                    value="yes"
                    defaultChecked={eeg.use_vat}
                    className="mt-0.5 h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  <div>
                    <p className="text-sm font-medium text-slate-800">Regelbesteuerung — USt-pflichtig</p>
                    <p className="text-xs text-slate-500 mt-0.5">MwSt. wird auf Verbrauchsrechnungen ausgewiesen und abgeführt.</p>
                  </div>
                </label>
                <div className="pl-7">
                  <label className={labelClass}>MwSt.-Satz (%)</label>
                  <div className="relative max-w-xs">
                    <input type="number" name="vat_pct" step="0.1" min="0" max="100" defaultValue={eeg.vat_pct || 20} className={`${inputClass} pr-8`} />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">%</span>
                  </div>
                  <p className="text-xs text-slate-400 mt-1">Nur relevant bei Regelbesteuerung. Standardwert: 20 %</p>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-4">Rechnungsnummern</h2>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={labelClass}>Rechnungsnummer-Präfix</label>
                    <input type="text" name="invoice_number_prefix" defaultValue={eeg.invoice_number_prefix || "INV"} placeholder="INV" className={inputClass} />
                  </div>
                  <div>
                    <label className={labelClass}>Stellen</label>
                    <input type="number" name="invoice_number_digits" min="1" max="10" defaultValue={eeg.invoice_number_digits || 5} className={inputClass} />
                    <p className="text-xs text-slate-400 mt-1">Anzahl der Ziffern in der Rechnungsnummer</p>
                  </div>
                  <div>
                    <label className={labelClass}>Startnummer</label>
                    <input type="number" name="invoice_number_start" min="1" defaultValue={eeg.invoice_number_start || 1} className={inputClass} />
                    <p className="text-xs text-slate-400 mt-1">Erste Rechnungsnummer dieser EEG</p>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-4">Rechnungstexte</h2>
              <div className="space-y-4">
                <div>
                  <label className={labelClass}>Text vor Positionen</label>
                  <textarea name="invoice_pre_text" rows={3} defaultValue={eeg.invoice_pre_text || ""} placeholder="Text der vor den Rechnungspositionen erscheint..." className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Text nach Positionen</label>
                  <textarea name="invoice_post_text" rows={3} defaultValue={eeg.invoice_post_text || ""} placeholder="Text der nach den Rechnungspositionen erscheint..." className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Fußzeile</label>
                  <textarea name="invoice_footer_text" rows={3} defaultValue={eeg.invoice_footer_text || ""} placeholder="Text der in der Fußzeile der Rechnung erscheint..." className={inputClass} />
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-1">Gutschriften (Produzenten)</h2>
              <p className="text-xs text-slate-500 mb-4">
                Wenn aktiviert, erhalten reine Produzenten ein separates Gutschrift-PDF.
              </p>
              <div className="space-y-4">
                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    id="generate_credit_notes"
                    name="generate_credit_notes"
                    defaultChecked={eeg.generate_credit_notes}
                    className="h-4 w-4 rounded border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  <label htmlFor="generate_credit_notes" className="text-sm font-medium text-slate-700">
                    Gutschriften für Produzenten generieren
                  </label>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={labelClass}>Gutschrift-Präfix</label>
                    <input type="text" name="credit_note_number_prefix" defaultValue={eeg.credit_note_number_prefix || "GS"} placeholder="GS" className={inputClass} />
                  </div>
                  <div>
                    <label className={labelClass}>Stellen</label>
                    <input type="number" name="credit_note_number_digits" min="1" max="10" defaultValue={eeg.credit_note_number_digits || 5} className={inputClass} />
                  </div>
                </div>
              </div>
            </div>
          </>
        )}

        {/* ── TAB: SEPA ──────────────────────────────────────────── */}
        {activeTab === "sepa" && (
          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <h2 className="text-base font-semibold text-slate-900 mb-1">SEPA Bankverbindung</h2>
            <p className="text-xs text-slate-500 mb-4">
              Für die Erzeugung von pain.001 (Gutschriften) und pain.008 (Lastschriften) Zahlungsdateien.
            </p>
            <div className="space-y-4">
              <div>
                <label className={labelClass}>IBAN der Gemeinschaft</label>
                <ValidatedInput
                  name="iban"
                  defaultValue={eeg.iban || ""}
                  placeholder="AT12 3456 7890 1234 5678"
                  validatorName="iban"
                  inputClassName={inputClass}
                />
              </div>
              <div>
                <label className={labelClass}>BIC (optional)</label>
                <ValidatedInput
                  name="bic"
                  defaultValue={eeg.bic || ""}
                  placeholder="RLNWATWWXXX"
                  validatorName="bic"
                  inputClassName={inputClass}
                />
              </div>
              <div>
                <label className={labelClass}>SEPA Gläubiger-ID</label>
                <ValidatedInput
                  name="sepa_creditor_id"
                  defaultValue={eeg.sepa_creditor_id || ""}
                  placeholder="AT00ZZZ00000000001"
                  validatorName="sepa_creditor_id"
                  inputClassName={inputClass}
                />
                <p className="text-xs text-slate-400 mt-1">
                  Erforderlich für pain.008 (Lastschriften). Ausgestellt von der Hausbank.
                </p>
              </div>
              <div>
                <label className={labelClass}>Voranmeldungsfrist (Tage)</label>
                <input
                  type="number"
                  name="sepa_pre_notification_days"
                  defaultValue={eeg.sepa_pre_notification_days ?? 14}
                  min={1}
                  max={90}
                  className={inputClass}
                />
                <p className="text-xs text-slate-400 mt-1">
                  Tage zwischen Rechnungsdatum und frühestem SEPA-Einzug (SEPA-Regelwerk: mind. 14 Tage).
                  Erscheint als „Der Betrag wird frühestens am XX fällig" auf der Rechnung.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* ── TAB: EDA ───────────────────────────────────────────── */}
        {activeTab === "eda" && (
          <>
          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <h2 className="text-base font-semibold text-slate-900 mb-1">EDA Kommunikation</h2>
            <p className="text-xs text-slate-500 mb-4">
              Österreichische Marktkommunikation (MaKo) — Elektronischer Datenaustausch mit dem Netzbetreiber.
            </p>
            <div className="space-y-4">
              <div>
                <label className={labelClass}>Gemeinschafts-ID (ECID)</label>
                <input type="text" name="gemeinschaft_id" defaultValue={eeg.gemeinschaft_id || ""} placeholder="AT00200000000RC1059700000000XXXXX" className={inputClass} />
                <p className="text-xs text-slate-400 mt-1">
                  Lange EC-Nummer (ECID) der Energiegemeinschaft — wird im XML-Body aller EDA-Nachrichten verwendet.
                </p>
              </div>
              <div>
                <label className={labelClass}>Marktpartner-ID (kurze EC-Nummer)</label>
                <input type="text" name="eda_marktpartner_id" defaultValue={eeg.eda_marktpartner_id || ""} placeholder="RC..." className={inputClass} />
                <p className="text-xs text-slate-400 mt-1">
                  Kurze EC-Nummer für SMTP-Routing und als Absender in EDA-Nachrichten.
                </p>
              </div>
              <div>
                <label className={labelClass}>Netzbetreiber-ID (EC-Nummer des Netzbetreibers)</label>
                <input type="text" name="eda_netzbetreiber_id" defaultValue={eeg.eda_netzbetreiber_id || ""} placeholder="AT..." className={inputClass} />
                <p className="text-xs text-slate-400 mt-1">
                  An diese Adresse werden EDA-Nachrichten gesendet.
                </p>
              </div>
              <div>
                <label className={labelClass}>EDA Umstellungsdatum</label>
                <input
                  type="date"
                  name="eda_transition_date"
                  defaultValue={eeg.eda_transition_date ? eeg.eda_transition_date.substring(0, 10) : ""}
                  className={inputClass}
                />
                <p className="text-xs text-slate-400 mt-1">
                  Ab diesem Datum ersetzt der EDA-E-Mail-Empfang den manuellen XLSX-Import (optional).
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <h2 className="text-base font-semibold text-slate-900 mb-1">EDA Postfach (IMAP)</h2>
            <p className="text-xs text-slate-500 mb-4">
              edanet.at Postfach zum Empfang eingehender EDA-Nachrichten (Bestätigungen, Energiedaten).
              Passwort leer lassen um das bestehende beizubehalten.
            </p>
            <div className="space-y-4">
              <div>
                <label className={labelClass}>IMAP Host:Port</label>
                <input type="text" name="eda_imap_host" defaultValue={(eeg as any).eda_imap_host || ""} placeholder="mail.edanet.at:993" className={inputClass} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Benutzername</label>
                  <input type="text" name="eda_imap_user" defaultValue={(eeg as any).eda_imap_user || ""} placeholder="rc105970" className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Passwort</label>
                  <input type="password" name="eda_imap_password" placeholder="••••••••  (unverändert)" className={inputClass} autoComplete="new-password" />
                </div>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <h2 className="text-base font-semibold text-slate-900 mb-1">EDA Ausgang (SMTP)</h2>
            <p className="text-xs text-slate-500 mb-4">
              edanet.at Postfach zum Versand ausgehender EDA-Nachrichten (Anmeldungen, Anforderungen).
              Passwort leer lassen um das bestehende beizubehalten.
            </p>
            <div className="space-y-4">
              <div>
                <label className={labelClass}>SMTP Host:Port</label>
                <input type="text" name="eda_smtp_host" defaultValue={(eeg as any).eda_smtp_host || ""} placeholder="mail.edanet.at:465" className={inputClass} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Benutzername</label>
                  <input type="text" name="eda_smtp_user" defaultValue={(eeg as any).eda_smtp_user || ""} placeholder="rc105970" className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Passwort</label>
                  <input type="password" name="eda_smtp_password" placeholder="••••••••  (unverändert)" className={inputClass} autoComplete="new-password" />
                </div>
              </div>
              <div>
                <label className={labelClass}>Absender-Adresse (From)</label>
                <input type="text" name="eda_smtp_from" defaultValue={(eeg as any).eda_smtp_from || ""} placeholder="rc105970@edanet.at" className={inputClass} />
              </div>
            </div>
          </div>
          </>
        )}

        {/* ── TAB: ONBOARDING ────────────────────────────────────── */}
        {activeTab === "onboarding" && (
          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <h2 className="text-base font-semibold text-slate-900 mb-1">Vertragstext (Beitrittserklärung)</h2>
            <p className="text-xs text-slate-500 mb-4">
              Dieser Text wird neuen Mitgliedern im Onboarding-Formular zur Unterzeichnung vorgelegt.
              Verfügbare Platzhalter: <code className="bg-slate-100 px-1 rounded">{"{iban}"}</code> (IBAN des Antragstellers),{" "}
              <code className="bg-slate-100 px-1 rounded">{"{datum}"}</code> (heutiges Datum).
              Wenn leer, wird ein Standard-Vertragstext verwendet.
            </p>
            <textarea
              name="onboarding_contract_text"
              rows={18}
              defaultValue={eeg.onboarding_contract_text || ""}
              placeholder={`BEITRITTSERKLÄRUNG ZUR ENERGIEGEMEINSCHAFT\n\nHiermit erkläre ich/wir den Beitritt zur Energiegemeinschaft ...\n\nKontoverbindung: IBAN: {iban}\n\nDatum: {datum}`}
              className={`${inputClass} font-mono text-xs`}
            />
          </div>
        )}

        {/* ── TAB: AUTO-ABRECHNUNG ──────────────────────────────── */}
        {activeTab === "abrechnung" && (
          <>
            <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-5">
              <div>
                <h2 className="text-base font-semibold text-slate-900 mb-1">Automatische Abrechnung</h2>
                <p className="text-sm text-slate-500">
                  Erstellt automatisch einen Abrechnungsentwurf an einem festgelegten Tag des Monats.
                  Der Entwurf bleibt im Status <strong>Entwurf</strong> — Sie erhalten eine E-Mail zur manuellen Prüfung und Freigabe.
                </p>
              </div>

              <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg border border-slate-200">
                <input
                  type="checkbox"
                  name="auto_billing_enabled"
                  id="auto_billing_enabled"
                  defaultChecked={!!eeg.auto_billing_enabled}
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                <label htmlFor="auto_billing_enabled" className="text-sm font-medium text-slate-700 cursor-pointer">
                  Automatische Abrechnung aktivieren
                </label>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Tag des Monats (1–28)</label>
                  <select
                    name="auto_billing_day_of_month"
                    defaultValue={eeg.auto_billing_day_of_month ?? 5}
                    className={inputClass}
                  >
                    {Array.from({ length: 28 }, (_, i) => i + 1).map((d) => (
                      <option key={d} value={d}>{d}.</option>
                    ))}
                  </select>
                  <p className="text-xs text-slate-400 mt-1">
                    An diesem Tag wird geprüft ob ein neuer Abrechnungslauf erstellt werden soll.
                  </p>
                </div>
                <div>
                  <label className={labelClass}>Abrechnungsperiode</label>
                  <select
                    name="auto_billing_period"
                    defaultValue={eeg.auto_billing_period || "monthly"}
                    className={inputClass}
                  >
                    <option value="monthly">Monatlich (Vormonat)</option>
                    <option value="quarterly">Quartalsweise (Vorquartal)</option>
                  </select>
                </div>
              </div>

              <div className="p-4 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800 space-y-1">
                <p className="font-medium">Hinweise:</p>
                <ul className="list-disc list-inside space-y-0.5 text-amber-700">
                  <li>Der Lauf wird <strong>nur erstellt wenn alle Readings lückenlos vorhanden sind</strong>. Bei fehlenden Daten erhalten Sie stattdessen eine Warn-E-Mail.</li>
                  <li>Voraussetzung: SMTP-Zugangsdaten müssen im Tab „System" konfiguriert sein (für E-Mail-Benachrichtigungen).</li>
                  <li>Laufzeit-Prüfung erfolgt täglich um 06:00 Uhr (Vienna-Zeit).</li>
                </ul>
              </div>

              {eeg.auto_billing_last_run_at && (
                <p className="text-xs text-slate-400">
                  Letzter automatischer Lauf: {new Date(eeg.auto_billing_last_run_at).toLocaleString("de-AT")}
                </p>
              )}
            </div>
          </>
        )}

        {/* ── TAB: LÜCKEN-ALARM ──────────────────────────────────── */}
        {activeTab === "luecken" && (
          <>
            <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-5">
              <div>
                <h2 className="text-base font-semibold text-slate-900 mb-1">Datenlücken-Alarm</h2>
                <p className="text-sm text-slate-500">
                  Erkennt registrierte Zählpunkte, die über einen festgelegten Zeitraum keine Energiedaten liefern,
                  und benachrichtigt per E-Mail.
                </p>
              </div>

              <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg border border-slate-200">
                <input
                  type="checkbox"
                  name="gap_alert_enabled"
                  id="gap_alert_enabled"
                  defaultChecked={(eeg as any).gap_alert_enabled !== false}
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                <label htmlFor="gap_alert_enabled" className="text-sm font-medium text-slate-700 cursor-pointer">
                  Lücken-Alarm aktivieren
                </label>
              </div>

              <div className="max-w-xs">
                <label className={labelClass}>Schwellenwert (Tage)</label>
                <div className="relative">
                  <input
                    type="number"
                    name="gap_alert_threshold_days"
                    min="1"
                    max="30"
                    defaultValue={(eeg as any).gap_alert_threshold_days ?? 5}
                    className={`${inputClass} pr-14`}
                  />
                  <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">Tage</span>
                </div>
                <p className="text-xs text-slate-400 mt-1">
                  Alarm wird ausgelöst wenn ein aktiver Zählpunkt länger als N Tage kein Reading liefert (Standard: 5).
                </p>
              </div>

              <div className="p-4 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800 space-y-1">
                <p className="font-medium">Hinweise:</p>
                <ul className="list-disc list-inside space-y-0.5 text-amber-700">
                  <li>Nur aktive (EDA-registrierte) Zählpunkte werden geprüft — abgemeldete werden ignoriert.</li>
                  <li>Ein Alarm wird pro Zählpunkt nur einmal gesendet. Sobald wieder Readings vorhanden sind, wird der Alarm automatisch zurückgesetzt.</li>
                  <li>Voraussetzung: SMTP-Zugangsdaten müssen im Tab „System" konfiguriert sein.</li>
                  <li>Prüfung erfolgt stündlich.</li>
                </ul>
              </div>
            </div>
          </>
        )}

        {/* ── TAB: MITGLIEDERPORTAL ──────────────────────────────── */}
        {activeTab === "portal" && (
          <>
            <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-5">
              <div>
                <h2 className="text-base font-semibold text-slate-900 mb-1">Mitgliederportal</h2>
                <p className="text-sm text-slate-500">
                  Steuert, welche Energiedaten Mitglieder im Self-Service-Portal sehen können.
                </p>
              </div>

              <div className="flex items-start gap-3 p-4 bg-slate-50 rounded-lg border border-slate-200">
                <input
                  type="checkbox"
                  name="portal_show_full_energy"
                  id="portal_show_full_energy"
                  defaultChecked={(eeg as any).portal_show_full_energy !== false}
                  className="mt-0.5 h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                <div>
                  <label htmlFor="portal_show_full_energy" className="text-sm font-medium text-slate-700 cursor-pointer">
                    Gesamtverbrauch und Reststrom anzeigen
                  </label>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Wenn aktiviert, sehen Mitglieder im Portal neben dem EEG-Anteil auch ihren Gesamtbezug,
                    Restbezug, Gesamteinspeisung und Resteinspeisung. Wenn deaktiviert, werden ausschließlich
                    die über die Energiegemeinschaft verrechneten Anteile angezeigt.
                  </p>
                </div>
              </div>
            </div>
          </>
        )}

        {/* ── TAB: SYSTEM ────────────────────────────────────────── */}
        {activeTab === "system" && (
          <>
            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-1">Firmenlogo</h2>
              <p className="text-xs text-slate-500 mb-4">
                Logo wird oben rechts auf Rechnungen und Gutschriften eingebettet (JPEG oder PNG, max. 4 MB).
              </p>
              <LogoUpload eegId={eegId} currentLogoPath={eeg.logo_path} />
            </div>

            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <h2 className="text-base font-semibold text-slate-900 mb-1">E-Mail Versand (Rechnungen)</h2>
              <p className="text-xs text-slate-500 mb-4">
                SMTP-Zugangsdaten für den Versand von Rechnungen, Gutschriften und Kommunikations-Mails an Mitglieder.
                Passwort leer lassen um das bestehende beizubehalten.
              </p>
              <div className="space-y-4">
                <div>
                  <label className={labelClass}>SMTP Host:Port</label>
                  <input type="text" name="smtp_host" defaultValue={(eeg as any).smtp_host || ""} placeholder="smtp.resend.com:587" className={inputClass} />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={labelClass}>Benutzername</label>
                    <input type="text" name="smtp_user" defaultValue={(eeg as any).smtp_user || ""} placeholder="resend" className={inputClass} />
                  </div>
                  <div>
                    <label className={labelClass}>Passwort / API-Key</label>
                    <input type="password" name="smtp_password" placeholder="••••••••  (unverändert)" className={inputClass} autoComplete="new-password" />
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Absender-Adresse (From)</label>
                  <input type="text" name="smtp_from" defaultValue={(eeg as any).smtp_from || ""} placeholder="kontakt@energiegemeinschaft.at" className={inputClass} />
                  <p className="text-xs text-slate-400 mt-1">
                    Muss mit der Domain des SMTP-Anbieters verifiziert sein.
                  </p>
                </div>
              </div>
            </div>
          </>
        )}

        {/* Actions — always show save button (system tab now has SMTP fields) */}
        {true && (
          <div className="flex gap-3">
            <button
              type="submit"
              className="px-6 py-2.5 bg-blue-700 text-white font-medium rounded-lg hover:bg-blue-800 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
            >
              Einstellungen speichern
            </button>
            <Link
              href={`/eegs/${eegId}`}
              className="px-6 py-2.5 border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-50 transition-colors"
            >
              Abbrechen
            </Link>
          </div>
        )}
      </form>

      {/* Backup & Wiederherstellung — only in system tab */}
      {activeTab === "system" && (
        <>
          <BackupRestoreSection eegId={eegId} eegName={eeg.name} />
          <DeleteEEGSection eegId={eegId} eegName={eeg.name} />
        </>
      )}
    </div>
  );
}
