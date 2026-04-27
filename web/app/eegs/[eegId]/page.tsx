import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getStats, listMembers, listEDAProcesses, listBillingRuns, countSepaReturns, listGapAlerts, type EEGStats } from "@/lib/api";
import Link from "next/link";

interface Props {
  params: Promise<{ eegId: string }>;
}

function AlertCard({
  color,
  title,
  description,
  linkHref,
  linkLabel,
}: {
  color: "orange" | "yellow" | "red" | "blue";
  title: string;
  description: string;
  linkHref: string;
  linkLabel: string;
}) {
  const colors = {
    orange: "bg-orange-50 border-orange-200 text-orange-800",
    yellow: "bg-yellow-50 border-yellow-200 text-yellow-800",
    red: "bg-red-50 border-red-200 text-red-800",
    blue: "bg-blue-50 border-blue-200 text-blue-800",
  };
  const dotColors = {
    orange: "bg-orange-400",
    yellow: "bg-yellow-400",
    red: "bg-red-500",
    blue: "bg-blue-400",
  };
  const linkColors = {
    orange: "text-orange-700 hover:text-orange-900 underline",
    yellow: "text-yellow-700 hover:text-yellow-900 underline",
    red: "text-red-700 hover:text-red-900 underline",
    blue: "text-blue-700 hover:text-blue-900 underline",
  };
  return (
    <div className={`rounded-lg border p-4 flex items-start gap-3 ${colors[color]}`}>
      <span className={`w-2 h-2 rounded-full mt-1.5 flex-shrink-0 ${dotColors[color]}`} />
      <div className="flex-1 min-w-0">
        <p className="font-medium text-sm">{title}</p>
        <p className="text-sm mt-0.5 opacity-80">{description}</p>
        <Link href={linkHref} className={`text-xs mt-1 inline-block font-medium ${linkColors[color]}`}>
          {linkLabel} →
        </Link>
      </div>
    </div>
  );
}

export default async function EEGOverviewPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;

  let eeg = null;
  let stats: EEGStats | null = null;
  let error: string | null = null;

  // Fetch additional data for alerts
  const now = new Date();
  const currentYear = now.getFullYear();
  const currentMonth = now.getMonth(); // 0-indexed

  const [eegResult, statsResult, membersResult, edaProcessesResult, billingRunsResult, sepaReturnsResult, gapAlertsResult] =
    await Promise.allSettled([
      getEEG(session.accessToken!, eegId),
      getStats(session.accessToken!, eegId),
      listMembers(session.accessToken!, eegId),
      listEDAProcesses(session.accessToken!, eegId),
      listBillingRuns(session.accessToken!, eegId),
      countSepaReturns(session.accessToken!, eegId),
      listGapAlerts(session.accessToken!, eegId),
    ]);

  if (eegResult.status === "fulfilled") {
    eeg = eegResult.value;
  } else {
    error = (eegResult.reason as { message?: string })?.message || "Fehler beim Laden.";
  }
  if (statsResult.status === "fulfilled") stats = statsResult.value;

  const members = membersResult.status === "fulfilled" ? membersResult.value : [];
  const edaProcesses = edaProcessesResult.status === "fulfilled" ? edaProcessesResult.value : [];
  const billingRuns = billingRunsResult.status === "fulfilled" ? billingRunsResult.value : [];
  const sepaReturnCount = sepaReturnsResult.status === "fulfilled" ? sepaReturnsResult.value : 0;
  const gapAlerts = gapAlertsResult.status === "fulfilled" ? gapAlertsResult.value : [];

  if (error && !eeg) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{error}</p>
        </div>
      </div>
    );
  }

  // ── Compute alerts ──────────────────────────────────────────────────────────

  // 1. Members without IBAN
  const membersWithoutIban = members.filter((m) => !m.iban || m.iban.trim() === "");

  // 2. Open EDA processes with deadline < 7 days
  const openStatuses = ["pending", "sent", "first_confirmed", "confirmed"];
  const urgentEdaProcesses = edaProcesses.filter((p) => {
    if (!openStatuses.includes(p.status)) return false;
    if (!p.deadline_at) return false;
    const dl = new Date(p.deadline_at);
    const diffDays = (dl.getTime() - now.getTime()) / (1000 * 60 * 60 * 24);
    return diffDays < 7;
  });

  // 3. Last billing run age
  const activeBillingRuns = billingRuns.filter((r) => r.status !== "cancelled");
  const lastRun = activeBillingRuns.length > 0
    ? activeBillingRuns.reduce((a, b) =>
        new Date(a.created_at || 0) > new Date(b.created_at || 0) ? a : b
      )
    : null;
  const lastRunDaysAgo = lastRun?.created_at
    ? (now.getTime() - new Date(lastRun.created_at).getTime()) / (1000 * 60 * 60 * 24)
    : null;

  const alerts: React.ReactNode[] = [];

  if (membersWithoutIban.length > 0) {
    alerts.push(
      <AlertCard
        key="iban"
        color="orange"
        title={`${membersWithoutIban.length} Mitglied${membersWithoutIban.length !== 1 ? "er" : ""} ohne IBAN`}
        description="Diese Mitglieder können nicht per SEPA-Lastschrift abgerechnet werden."
        linkHref={`/eegs/${eegId}/members`}
        linkLabel="Zu den Mitgliedern"
      />
    );
  }

  if (urgentEdaProcesses.length > 0) {
    const overdue = urgentEdaProcesses.filter((p) => new Date(p.deadline_at!) < now);
    const color = overdue.length > 0 ? "red" : "yellow";
    alerts.push(
      <AlertCard
        key="eda"
        color={color}
        title={`${urgentEdaProcesses.length} offene EDA-Prozesse mit ablaufender Frist`}
        description={overdue.length > 0
          ? `${overdue.length} Prozess${overdue.length !== 1 ? "e" : ""} bereits überfällig.`
          : "Frist in weniger als 7 Tagen."}
        linkHref={`/eegs/${eegId}/eda`}
        linkLabel="EDA-Prozesse öffnen"
      />
    );
  }

  if (!lastRun) {
    alerts.push(
      <AlertCard
        key="billing-none"
        color="blue"
        title="Noch kein Abrechnungslauf vorhanden"
        description="Starten Sie die erste Abrechnung für diese Energiegemeinschaft."
        linkHref={`/eegs/${eegId}/billing`}
        linkLabel="Zur Abrechnung"
      />
    );
  } else if (lastRunDaysAgo !== null && lastRunDaysAgo > 35) {
    alerts.push(
      <AlertCard
        key="billing-old"
        color="blue"
        title="Letzter Abrechnungslauf vor mehr als 35 Tagen"
        description={`Letzter Lauf: ${new Date(lastRun.created_at!).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" })}`}
        linkHref={`/eegs/${eegId}/billing`}
        linkLabel="Neue Abrechnung starten"
      />
    );
  }

  if (sepaReturnCount > 0) {
    alerts.push(
      <AlertCard
        key="sepa-returns"
        color="orange"
        title={`${sepaReturnCount} offene Rücklastschrift${sepaReturnCount !== 1 ? "en" : ""}`}
        description="Es gibt Rechnungen mit SEPA-Rücklastschriften, die noch bearbeitet werden müssen."
        linkHref={`/eegs/${eegId}/billing`}
        linkLabel="Zur Abrechnung"
      />
    );
  }

  if (gapAlerts.length > 0) {
    alerts.push(
      <AlertCard
        key="gap-alerts"
        color="orange"
        title={`${gapAlerts.length} Zählpunkt${gapAlerts.length !== 1 ? "e" : ""} ohne aktuelle Readings`}
        description={`${gapAlerts.map((g) => g.member_name).filter((v, i, a) => a.indexOf(v) === i).slice(0, 3).join(", ")}${gapAlerts.length > 3 ? " u.a." : ""} — Datenlücke erkannt.`}
        linkHref={`/eegs/${eegId}/members`}
        linkLabel="Zu den Mitgliedern"
      />
    );
  }

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link
          href="/eegs"
          className="text-sm text-slate-500 hover:text-slate-700"
        >
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">
          {eeg?.name || eegId}
        </span>
      </div>

      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900">{eeg?.name}</h1>
        <div className="flex gap-4 mt-2 flex-wrap">
          <span className="text-sm text-slate-500">
            <span className="font-medium">Gemeinschafts-ID:</span>{" "}
            {eeg?.gemeinschaft_id}
          </span>
          <span className="text-slate-300">|</span>
          <span className="text-sm text-slate-500">
            <span className="font-medium">Netzbetreiber:</span>{" "}
            {eeg?.netzbetreiber}
          </span>
          <span className="text-slate-300">|</span>
          <span className="text-sm text-slate-500">
            <span className="font-medium">Abrechnungsperiode:</span>{" "}
            {eeg?.billing_period
              ? ({ monthly: "Monatlich", quarterly: "Vierteljährlich", semiannual: "Halbjährlich", annual: "Jährlich" } as Record<string,string>)[eeg.billing_period] ?? eeg.billing_period
              : "—"}
          </span>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Mitglieder</p>
          <p className="text-2xl font-bold text-slate-900 mt-1">{stats?.member_count ?? "—"}</p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Zählpunkte</p>
          <p className="text-2xl font-bold text-slate-900 mt-1">{stats?.meter_point_count ?? "—"}</p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Abrechnungen</p>
          <p className="text-2xl font-bold text-slate-900 mt-1">{stats?.billing_run_count ?? "—"}</p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Rechnungen</p>
          <p className="text-2xl font-bold text-slate-900 mt-1">{stats?.invoice_count ?? "—"}</p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Ausgetauschte Energie</p>
          <p className="text-2xl font-bold text-slate-900 mt-1">
            {stats ? (stats.total_kwh >= 1000
              ? `${(stats.total_kwh / 1000).toFixed(1)} MWh`
              : `${stats.total_kwh.toFixed(0)} kWh`)
            : "—"}
          </p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Umsatz (€)</p>
          <p className="text-2xl font-bold text-slate-900 mt-1">
            {stats ? `€ ${stats.total_revenue.toFixed(2)}` : "—"}
          </p>
        </div>
      </div>

      {/* Alerts */}
      {alerts.length > 0 && (
        <div className="mb-8 bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200">
            <h2 className="text-base font-semibold text-slate-900">Hinweise &amp; Handlungsbedarf</h2>
          </div>
          <div className="p-4 space-y-3">
            {alerts}
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Link
          href={`/eegs/${eegId}/members`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center mb-3 group-hover:bg-blue-100 transition-colors">
            <svg
              className="w-5 h-5 text-blue-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Mitglieder</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Mitglieder und Zählpunkte verwalten
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/import`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-green-50 flex items-center justify-center mb-3 group-hover:bg-green-100 transition-colors">
            <svg
              className="w-5 h-5 text-green-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Daten importieren</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Stammdaten und Energiedaten hochladen
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/billing`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-amber-50 flex items-center justify-center mb-3 group-hover:bg-amber-100 transition-colors">
            <svg
              className="w-5 h-5 text-amber-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Abrechnung</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Abrechnung starten und Rechnungen verwalten
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/tariffs`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-yellow-50 flex items-center justify-center mb-3 group-hover:bg-yellow-100 transition-colors">
            <svg
              className="w-5 h-5 text-yellow-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Tarifpläne</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Zeitvariable Preise für Bezug und Einspeisung
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/settings`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-slate-50 flex items-center justify-center mb-3 group-hover:bg-slate-100 transition-colors">
            <svg
              className="w-5 h-5 text-slate-600"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Einstellungen</p>
          <p className="text-sm text-slate-500 mt-0.5">Tarife und Abrechnungsparameter</p>
        </Link>

        <Link
          href={`/eegs/${eegId}/eda`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-purple-50 flex items-center justify-center mb-3 group-hover:bg-purple-100 transition-colors">
            <svg
              className="w-5 h-5 text-purple-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">EDA</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Datenaustausch mit dem Netzbetreiber
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/reports`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-teal-50 flex items-center justify-center mb-3 group-hover:bg-teal-100 transition-colors">
            <svg
              className="w-5 h-5 text-teal-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Auswertungen</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Energiebericht und Statistiken
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/participations`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-orange-50 flex items-center justify-center mb-3 group-hover:bg-orange-100 transition-colors">
            <svg
              className="w-5 h-5 text-orange-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Mehrfachteilnahme</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Parallele EEG-Teilnahme verwalten
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/onboarding`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-indigo-50 flex items-center justify-center mb-3 group-hover:bg-indigo-100 transition-colors">
            <svg
              className="w-5 h-5 text-indigo-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Onboarding</p>
          <p className="text-sm text-slate-500 mt-0.5">
            Beitrittsanträge verwalten
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/accounting`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-emerald-50 flex items-center justify-center mb-3 group-hover:bg-emerald-100 transition-colors">
            <svg
              className="w-5 h-5 text-emerald-700"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z"
              />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Buchhaltungsexport</p>
          <p className="text-sm text-slate-500 mt-0.5">
            XLSX / DATEV exportieren
          </p>
        </Link>

        <Link
          href={`/eegs/${eegId}/communications`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-sky-50 flex items-center justify-center mb-3 group-hover:bg-sky-100 transition-colors">
            <svg className="w-5 h-5 text-sky-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Aussendungen</p>
          <p className="text-sm text-slate-500 mt-0.5">E-Mails an Mitglieder senden</p>
        </Link>

        <Link
          href={`/eegs/${eegId}/documents`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-rose-50 flex items-center justify-center mb-3 group-hover:bg-rose-100 transition-colors">
            <svg className="w-5 h-5 text-rose-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </div>
          <p className="font-medium text-slate-900">Dokumente</p>
          <p className="text-sm text-slate-500 mt-0.5">Dateien für Mitglieder bereitstellen</p>
        </Link>

        <Link
          href={`/eegs/${eegId}/ea`}
          className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
        >
          <div className="w-10 h-10 rounded-lg bg-violet-50 flex items-center justify-center mb-3 group-hover:bg-violet-100 transition-colors">
            <svg className="w-5 h-5 text-violet-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
            </svg>
          </div>
          <p className="font-medium text-slate-900">E/A-Buchhaltung</p>
          <p className="text-sm text-slate-500 mt-0.5">Einnahmen-Ausgaben, USt, UVA, Jahresabschluss</p>
        </Link>
      </div>
    </div>
  );
}
