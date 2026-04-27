import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, listEDAMessages, listEDAProcesses, listEDAErrors, getEDAWorkerStatus, type EDAMessage, type EDAProcess, type EDAError, type EDAWorkerStatus } from "@/lib/api";
import Link from "next/link";
import { EDAActionForms, PollNowButton } from "@/components/eda-action-forms";
import { EDAProcessesTable } from "@/components/eda-processes-table";
import { EDAMessagesTable } from "@/components/eda-messages-table";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ tab?: string; page?: string; pageSize?: string }>;
}

const TABS = [
  { key: "uebersicht", label: "Übersicht" },
  { key: "prozesse",   label: "Prozesse" },
  { key: "nachrichten", label: "Nachrichten" },
  { key: "aktionen",   label: "Aktionen" },
  { key: "fehler",     label: "Fehler" },
] as const;

type TabKey = (typeof TABS)[number]["key"];

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit", month: "2-digit", year: "numeric",
      hour: "2-digit", minute: "2-digit",
      timeZone: "Europe/Vienna",
    });
  } catch { return dateStr; }
}

function formatDateShort(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit", month: "2-digit", year: "numeric",
      timeZone: "Europe/Vienna",
    });
  } catch { return dateStr; }
}

export default async function EDAPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { tab, page, pageSize } = await searchParams;
  const activeTab = (tab as TabKey) || "uebersicht";
  const messagePageSize = (() => {
    const parsed = Number(pageSize);
    if (!Number.isFinite(parsed)) return 100;
    if (parsed <= 0) return 100;
    return Math.min(parsed, 500);
  })();
  const currentMessagePage = (() => {
    const parsed = Number(page);
    if (!Number.isFinite(parsed) || parsed < 1) return 1;
    return Math.floor(parsed);
  })();
  const messageOffset = (currentMessagePage - 1) * messagePageSize;

  let eeg = null;
  let messages: EDAMessage[] = [];
  let totalMessageCount = 0;
  let processes: EDAProcess[] = [];
  let edaErrors: EDAError[] = [];
  let workerStatus: EDAWorkerStatus | null = null;
  let error: string | null = null;

  try {
    const [eegResult, messageResult, processResult, errorResult, workerStatusResult] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      listEDAMessages(session.accessToken!, eegId, {
        limit: messagePageSize,
        offset: messageOffset,
      }).catch(() => ({ messages: [], total_count: 0, limit: messagePageSize, offset: messageOffset })),
      listEDAProcesses(session.accessToken!, eegId).catch(() => []),
      listEDAErrors(session.accessToken!, eegId).catch(() => []),
      getEDAWorkerStatus(session.accessToken!).catch(() => null),
    ]);
    eeg = eegResult;
    messages = messageResult.messages;
    totalMessageCount = messageResult.total_count;
    processes = processResult;
    edaErrors = errorResult;
    workerStatus = workerStatusResult;
  } catch (err: unknown) {
    error = (err as { message?: string }).message || "Fehler beim Laden.";
  }

  const edaConfigured = !!(eeg?.eda_marktpartner_id && eeg?.eda_netzbetreiber_id);
  const openCount = processes.filter(
    (p) => !["confirmed", "completed", "rejected", "error"].includes(p.status)
  ).length;

  const tabHref = (key: string) => `/eegs/${eegId}/eda?tab=${key}`;

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          {eeg?.name || eegId}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">EDA</span>
      </div>

      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Elektronischer Datenaustausch</h1>
        <p className="text-slate-500 mt-1">
          Marktkommunikation mit dem Netzbetreiber für {eeg?.name || "diese Energiegemeinschaft"}.
        </p>
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {error}
        </div>
      )}

      {/* Tab bar */}
      <div className="flex gap-1 mb-6 border-b border-slate-200">
        {TABS.map((tab) => {
          const badge =
            tab.key === "prozesse" && openCount > 0 ? openCount :
            tab.key === "fehler" && edaErrors.length > 0 ? edaErrors.length :
            null;
          return (
            <Link
              key={tab.key}
              href={tabHref(tab.key)}
              className={`inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-t-lg transition-colors border-b-2 -mb-px ${
                activeTab === tab.key
                  ? "border-blue-600 text-blue-700 bg-white"
                  : "border-transparent text-slate-500 hover:text-slate-700 hover:bg-slate-50"
              }`}
            >
              {tab.label}
              {badge != null && (
                <span className={`inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-xs font-semibold ${
                  activeTab === tab.key
                    ? "bg-blue-100 text-blue-700"
                    : "bg-slate-200 text-slate-600"
                }`}>
                  {badge}
                </span>
              )}
            </Link>
          );
        })}
      </div>

      {/* ── TAB: ÜBERSICHT ───────────────────────────────────── */}
      {activeTab === "uebersicht" && (
        <div className="max-w-3xl space-y-6">
          {/* Verbindungskonfiguration */}
          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <div className="flex items-start justify-between mb-4">
              <div>
                <h2 className="text-base font-semibold text-slate-900">Verbindungskonfiguration</h2>
                <p className="text-xs text-slate-500 mt-0.5">EDA-Kennnummern dieser Energiegemeinschaft.</p>
              </div>
              {!edaConfigured && (
                <Link
                  href={`/eegs/${eegId}/settings?tab=eda`}
                  className="text-xs text-blue-600 hover:underline whitespace-nowrap"
                >
                  In Einstellungen konfigurieren →
                </Link>
              )}
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="bg-slate-50 rounded-lg p-4">
                <p className="text-xs font-medium text-slate-500 mb-1">Status</p>
                {edaConfigured ? (
                  <span className="inline-flex items-center gap-1.5 text-sm font-medium text-green-700">
                    <span className="w-2 h-2 rounded-full bg-green-500 inline-block" />
                    Konfiguriert
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1.5 text-sm font-medium text-amber-700">
                    <span className="w-2 h-2 rounded-full bg-amber-400 inline-block" />
                    Nicht konfiguriert
                  </span>
                )}
              </div>
              <div className="bg-slate-50 rounded-lg p-4">
                <p className="text-xs font-medium text-slate-500 mb-1">Marktpartner-ID</p>
                <p className="text-sm font-mono text-slate-900 truncate">
                  {eeg?.eda_marktpartner_id || <span className="text-slate-400 font-sans">—</span>}
                </p>
              </div>
              <div className="bg-slate-50 rounded-lg p-4">
                <p className="text-xs font-medium text-slate-500 mb-1">Netzbetreiber-ID</p>
                <p className="text-sm font-mono text-slate-900 truncate">
                  {eeg?.eda_netzbetreiber_id || <span className="text-slate-400 font-sans">—</span>}
                </p>
              </div>
            </div>
            {eeg?.eda_transition_date && (
              <p className="text-xs text-slate-500 mt-3">
                EDA-Umstellungsdatum:{" "}
                <span className="font-medium text-slate-700">{formatDateShort(eeg.eda_transition_date)}</span>
                {" "}— ab diesem Datum ersetzt der EDA-Empfang den XLSX-Import.
              </p>
            )}
          </div>

          {/* Worker Status */}
          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <div className="flex items-start justify-between mb-4">
              <div>
                <h2 className="text-base font-semibold text-slate-900">EDA Worker</h2>
                <p className="text-xs text-slate-500 mt-0.5">Verbindungsstatus und letzter Poll-Zeitpunkt.</p>
              </div>
              {edaConfigured && (
                <PollNowButton eegId={eegId} />
              )}
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="bg-slate-50 rounded-lg p-4">
                <p className="text-xs font-medium text-slate-500 mb-1">Transport</p>
                <p className="text-sm font-mono text-slate-900">
                  {workerStatus?.transport_mode || <span className="text-slate-400 font-sans">—</span>}
                </p>
              </div>
              <div className="bg-slate-50 rounded-lg p-4">
                <p className="text-xs font-medium text-slate-500 mb-1">Letzter Poll</p>
                <p className="text-sm text-slate-900">
                  {workerStatus?.last_poll_at
                    ? formatDate(workerStatus.last_poll_at)
                    : <span className="text-slate-400">Noch nicht gelaufen</span>}
                </p>
              </div>
              <div className="bg-slate-50 rounded-lg p-4">
                <p className="text-xs font-medium text-slate-500 mb-1">Letzter Fehler</p>
                <p className="text-sm truncate" title={workerStatus?.last_error}>
                  {workerStatus?.last_error
                    ? <span className="text-red-600">{workerStatus.last_error}</span>
                    : <span className="text-green-600">Kein Fehler</span>}
                </p>
              </div>
            </div>
          </div>

          {/* Quick stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-white rounded-xl border border-slate-200 p-5">
              <p className="text-xs font-medium text-slate-500 mb-1">Laufende Prozesse</p>
              <p className="text-2xl font-bold text-slate-900">{openCount}</p>
              {openCount > 0 && (
                <Link href={tabHref("prozesse")} className="text-xs text-blue-600 hover:underline mt-1 block">
                  Anzeigen →
                </Link>
              )}
            </div>
            <div className="bg-white rounded-xl border border-slate-200 p-5">
              <p className="text-xs font-medium text-slate-500 mb-1">Nachrichten gesamt</p>
              <p className="text-2xl font-bold text-slate-900">{totalMessageCount}</p>
              {totalMessageCount > 0 && (
                <Link href={tabHref("nachrichten")} className="text-xs text-blue-600 hover:underline mt-1 block">
                  Anzeigen →
                </Link>
              )}
            </div>
            <div className="bg-white rounded-xl border border-slate-200 p-5">
              <p className="text-xs font-medium text-slate-500 mb-1">Offene Fehler</p>
              <p className={`text-2xl font-bold ${edaErrors.length > 0 ? "text-red-600" : "text-slate-900"}`}>
                {edaErrors.length}
              </p>
              {edaErrors.length > 0 && (
                <Link href={tabHref("fehler")} className="text-xs text-red-600 hover:underline mt-1 block">
                  Anzeigen →
                </Link>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ── TAB: PROZESSE ────────────────────────────────────── */}
      {activeTab === "prozesse" && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
            <div>
              <h2 className="text-base font-semibold text-slate-900">Prozesse</h2>
              <p className="text-xs text-slate-500 mt-0.5">
                Anmeldungen, Abmeldungen und Faktoränderungen.
              </p>
            </div>
            {openCount > 0 && (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
                {openCount} laufend
              </span>
            )}
          </div>
          <EDAProcessesTable processes={processes} />
        </div>
      )}

      {/* ── TAB: NACHRICHTEN ─────────────────────────────────── */}
      {activeTab === "nachrichten" && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200">
            <h2 className="text-base font-semibold text-slate-900">Nachrichtenprotokoll</h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Alle eingehenden und ausgehenden EDA-Nachrichten, neueste zuerst. Standard: 100 pro Seite.
            </p>
          </div>
          <EDAMessagesTable
            messages={messages}
            eegId={eegId}
            totalCount={totalMessageCount}
            page={currentMessagePage}
            pageSize={messagePageSize}
          />
        </div>
      )}

      {/* ── TAB: AKTIONEN ────────────────────────────────────── */}
      {activeTab === "aktionen" && (
        <EDAActionForms eegId={eegId} edaConfigured={edaConfigured} netzbetreiberId={eeg?.eda_netzbetreiber_id ?? ""} />
      )}

      {/* ── TAB: FEHLER ──────────────────────────────────────── */}
      {activeTab === "fehler" && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200">
            <h2 className="text-base font-semibold text-slate-900">Fehler-Log</h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Eingehende Nachrichten, die nicht verarbeitet werden konnten.
            </p>
          </div>
          {edaErrors.length === 0 ? (
            <div className="px-6 py-10 text-center">
              <svg className="mx-auto w-10 h-10 text-slate-300 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="text-sm text-slate-500">Keine Fehler vorhanden.</p>
            </div>
          ) : (
            <>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-200 bg-slate-50">
                    <th className="text-left px-6 py-3 font-medium text-slate-600">Zeitpunkt</th>
                    <th className="text-left px-6 py-3 font-medium text-slate-600">Typ</th>
                    <th className="text-left px-6 py-3 font-medium text-slate-600">Nachrichtenreferenz</th>
                    <th className="text-left px-6 py-3 font-medium text-slate-600">Fehler</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {edaErrors.map((e) => (
                    <tr key={e.id} className="hover:bg-slate-50 transition-colors">
                      <td className="px-6 py-3.5 text-slate-500 text-xs whitespace-nowrap">
                        {formatDate(e.created_at)}
                      </td>
                      <td className="px-6 py-3.5 font-mono text-xs text-slate-600">
                        {e.message_type || "—"}
                      </td>
                      <td className="px-6 py-3.5 font-mono text-xs text-slate-500 max-w-[200px] truncate" title={e.subject}>
                        {e.subject || "—"}
                      </td>
                      <td className="px-6 py-3.5 text-red-700 text-xs max-w-xs truncate" title={e.error_msg}>
                        {e.error_msg}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
                <p className="text-xs text-slate-500">{edaErrors.length} Fehler gesamt</p>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
