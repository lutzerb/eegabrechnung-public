"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import {
  getEEG,
  listBillingRuns,
  listInvoicesByBillingRun,
  listMembers,
  updateInvoiceStatus,
  resendInvoice,
  sendAllInvoicesByRun,
  finalizeBillingRun,
  deleteBillingRun,
  cancelBillingRun,
  type BillingRun,
  type Invoice,
  type Member,
} from "@/lib/api";
import Link from "next/link";
import BillingRunForm from "@/components/billing-run-form";
import CancelRunButton from "@/components/cancel-run-button";
import BillingRunsFilters from "@/components/billing-runs-filters";
import { SepaDownloadButtons } from "@/components/sepa-download-buttons";
import SepaReturnInvoiceActions from "@/components/sepa-return-invoice-actions";
import Camt054Upload from "@/components/camt054-upload";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ success?: string; error?: string; show_cancelled?: string; sort?: string }>;
}

function StatusBadge({ status }: { status: string }) {
  const statusMap: Record<string, { label: string; className: string }> = {
    draft: { label: "Entwurf", className: "bg-slate-100 text-slate-600" },
    finalized: { label: "Finalisiert", className: "bg-green-50 text-green-700" },
    pending: { label: "Ausstehend", className: "bg-yellow-50 text-yellow-700" },
    sent: { label: "Versendet", className: "bg-blue-50 text-blue-700" },
    paid: { label: "Bezahlt", className: "bg-green-50 text-green-700" },
    cancelled: { label: "Storniert", className: "bg-red-50 text-red-700" },
  };
  const config = statusMap[status?.toLowerCase()] || {
    label: status,
    className: "bg-slate-100 text-slate-600",
  };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${config.className}`}>
      {config.label}
    </span>
  );
}

function formatDate(dateStr: string): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(amount);
}

function periodLabel(run: BillingRun): string {
  const s = new Date(run.period_start);
  const e = new Date(run.period_end);
  const sameMonth =
    s.getFullYear() === e.getFullYear() && s.getMonth() === e.getMonth();
  if (sameMonth) {
    const lastOfMonth = new Date(e.getFullYear(), e.getMonth() + 1, 0).getDate();
    if (s.getDate() === 1 && e.getDate() === lastOfMonth) {
      return s.toLocaleDateString("de-AT", { month: "long", year: "numeric" });
    }
  }
  return `${formatDate(run.period_start)} – ${formatDate(run.period_end)}`;
}

function periodDateRange(run: BillingRun): string {
  return `${formatDate(run.period_start)} – ${formatDate(run.period_end)}`;
}

function isFullMonth(run: BillingRun): boolean {
  const s = new Date(run.period_start);
  const e = new Date(run.period_end);
  const lastOfMonth = new Date(e.getFullYear(), e.getMonth() + 1, 0).getDate();
  return (
    s.getFullYear() === e.getFullYear() &&
    s.getMonth() === e.getMonth() &&
    s.getDate() === 1 &&
    e.getDate() === lastOfMonth
  );
}

interface RunSectionProps {
  run: BillingRun;
  invoices: Invoice[];
  memberNames: Record<string, string>;
  eegId: string;
  markPaidAction: (formData: FormData) => Promise<void>;
  cancelInvoiceAction: (formData: FormData) => Promise<void>;
  sendInvoiceAction: (formData: FormData) => Promise<void>;
  resendInvoiceAction: (formData: FormData) => Promise<void>;
  sendRunAction: (formData: FormData) => Promise<void>;
  finalizeRunAction: (formData: FormData) => Promise<void>;
  deleteRunAction: (formData: FormData) => Promise<void>;
  cancelRunAction: (formData: FormData) => Promise<void>;
}

function RunSection({
  run,
  invoices,
  memberNames,
  eegId,
  markPaidAction,
  cancelInvoiceAction,
  sendInvoiceAction,
  resendInvoiceAction,
  sendRunAction,
  finalizeRunAction,
  deleteRunAction,
  cancelRunAction,
}: RunSectionProps) {
  const total = invoices.reduce((s, i) => s + (i.total_amount || 0), 0);

  return (
    <details className="bg-white rounded-xl border border-slate-200 overflow-hidden" open>
      {/* Summary row */}
      <summary className="flex items-center justify-between px-5 py-4 cursor-pointer list-none select-none hover:bg-slate-50 transition-colors">
        <div className="flex items-center gap-4">
          <span className="font-semibold text-slate-900">{periodLabel(run)}</span>
            {isFullMonth(run) && (
              <span className="text-xs text-slate-400">{periodDateRange(run)}</span>
            )}
          <StatusBadge status={run.status} />
          <span className="text-xs text-slate-400">
            {run.invoice_count} {run.invoice_count === 1 ? "Rechnung" : "Rechnungen"}
          </span>
          <span className="font-medium text-slate-700">{formatCurrency(total)}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-400">
            {run.created_at ? formatDate(run.created_at) : ""}
          </span>
          {/* chevron */}
          <svg className="w-4 h-4 text-slate-400 transition-transform details-chevron" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </summary>

      {/* Toolbar — outside summary so links don't toggle the details */}
      <div className="flex flex-col gap-2 px-5 py-2 border-t border-slate-100 bg-slate-50">
      <div className="flex items-center gap-2 flex-wrap">
        <SepaDownloadButtons eegId={eegId} billingRunId={run.id} />
        <span className="text-xs text-slate-500 ml-2">Export:</span>
        <a
          href={`/api/eegs/${eegId}/billing/runs/${run.id}/zip`}
          download
          className="px-2.5 py-1 text-xs font-medium text-violet-700 bg-violet-50 border border-violet-200 rounded hover:bg-violet-100 transition-colors"
        >
          ZIP (PDFs)
        </a>
        <a
          href={`/api/eegs/${eegId}/billing/runs/${run.id}/export`}
          download
          className="px-2.5 py-1 text-xs font-medium text-teal-700 bg-teal-50 border border-teal-200 rounded hover:bg-teal-100 transition-colors"
        >
          Excel
        </a>
        <span className="flex-1" />
        {run.status === "draft" && (
          <form action={finalizeRunAction}>
            <input type="hidden" name="runId" value={run.id} />
            <button
              type="submit"
              className="px-3 py-1 text-xs font-medium text-white bg-green-600 rounded hover:bg-green-700 transition-colors"
            >
              Finalisieren
            </button>
          </form>
        )}
        {run.status === "draft" && (
          <form action={deleteRunAction}>
            <input type="hidden" name="runId" value={run.id} />
            <button
              type="submit"
              className="px-3 py-1 text-xs font-medium text-red-700 bg-red-50 border border-red-200 rounded hover:bg-red-100 transition-colors"
            >
              Löschen
            </button>
          </form>
        )}
        {run.status === "finalized" && (
          <form action={sendRunAction}>
            <input type="hidden" name="runId" value={run.id} />
            <button
              type="submit"
              className="px-3 py-1 text-xs font-medium text-white bg-blue-600 rounded hover:bg-blue-700 transition-colors"
            >
              Rechnungen versenden
            </button>
          </form>
        )}
        {run.status === "finalized" && (
          <CancelRunButton runId={run.id} cancelRunAction={cancelRunAction} />
        )}
      </div>
      <div className="flex items-center gap-2">
        <span className="text-xs text-slate-500">SEPA-Rücklastschriften:</span>
        <Camt054Upload eegId={eegId} />
      </div>
      </div>

      {/* Invoice table */}
      {invoices.length === 0 ? (
        <div className="px-5 py-6 text-center text-sm text-slate-400 border-t border-slate-100">
          Keine Rechnungen in diesem Abrechnungslauf.
        </div>
      ) : (
        <div className="overflow-x-auto">
        <table className="w-full text-sm border-t border-slate-100 min-w-[600px]">
          <thead>
            <tr className="bg-slate-50 border-b border-slate-100">
              <th className="text-left px-5 py-2.5 font-medium text-slate-500 text-xs">Nr.</th>
              <th className="text-left px-5 py-2.5 font-medium text-slate-500 text-xs">Mitglied</th>
              <th className="text-right px-5 py-2.5 font-medium text-slate-500 text-xs">Betrag</th>
              <th className="text-left px-5 py-2.5 font-medium text-slate-500 text-xs">Typ</th>
              <th className="text-left px-5 py-2.5 font-medium text-slate-500 text-xs">Status</th>
              <th className="text-right px-5 py-2.5 font-medium text-slate-500 text-xs">Aktionen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-50">
            {invoices.map((invoice) => (
              <tr key={invoice.id} className="hover:bg-slate-50 transition-colors">
                <td className="px-5 py-3 text-slate-400 font-mono text-xs">
                  {invoice.invoice_number ? `#${invoice.invoice_number}` : invoice.id.slice(0, 8)}
                </td>
                <td className="px-5 py-3 font-medium text-slate-900">
                  {memberNames[invoice.member_id] || invoice.member_id}
                </td>
                <td className="px-5 py-3 text-right font-medium text-slate-900">
                  {formatCurrency(invoice.total_amount)}
                </td>
                <td className="px-5 py-3">
                  {invoice.document_type === "credit_note" ? (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-50 text-purple-700">
                      Gutschrift
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">
                      Rechnung
                    </span>
                  )}
                </td>
                <td className="px-5 py-3">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <StatusBadge status={invoice.status} />
                    {invoice.sepa_return_at && (
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-50 text-red-700 border border-red-200">
                        Rücklastschr.
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-5 py-3">
                  <div className="flex items-center justify-end gap-1.5 flex-wrap">
                    <SepaReturnInvoiceActions eegId={eegId} invoice={invoice} />
                    {invoice.pdf_path && (
                      <a
                        href={`/api/eegs/${eegId}/invoices/${invoice.id}/pdf`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="px-2 py-1 text-xs font-medium text-slate-700 bg-slate-100 rounded hover:bg-slate-200 transition-colors"
                      >
                        PDF
                      </a>
                    )}
                    {invoice.sent_at ? (
                      <span className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-green-700 bg-green-50 rounded border border-green-200" title={`Gesendet am ${formatDate(invoice.sent_at)}`}>
                        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                        {formatDate(invoice.sent_at)}
                      </span>
                    ) : (
                      invoice.pdf_path && invoice.status?.toLowerCase() !== "cancelled" && (
                        <form action={sendInvoiceAction}>
                          <input type="hidden" name="invoiceId" value={invoice.id} />
                          <button type="submit" className="px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded hover:bg-blue-100 transition-colors">
                            Senden
                          </button>
                        </form>
                      )
                    )}
                    {invoice.sent_at && invoice.status?.toLowerCase() !== "cancelled" && (
                      <form action={resendInvoiceAction}>
                        <input type="hidden" name="invoiceId" value={invoice.id} />
                        <button type="submit" className="px-2 py-1 text-xs font-medium text-slate-600 bg-slate-100 rounded hover:bg-slate-200 transition-colors">
                          Erneut
                        </button>
                      </form>
                    )}
                    {invoice.status?.toLowerCase() !== "paid" && invoice.status?.toLowerCase() !== "cancelled" && (
                      <form action={markPaidAction}>
                        <input type="hidden" name="invoiceId" value={invoice.id} />
                        <button type="submit" className="px-2 py-1 text-xs font-medium text-green-700 bg-green-50 rounded hover:bg-green-100 transition-colors">
                          Bezahlt
                        </button>
                      </form>
                    )}
                    {invoice.status?.toLowerCase() !== "cancelled" && (
                      <form action={cancelInvoiceAction}>
                        <input type="hidden" name="invoiceId" value={invoice.id} />
                        <button type="submit" className="px-2 py-1 text-xs font-medium text-red-700 bg-red-50 rounded hover:bg-red-100 transition-colors">
                          Stornieren
                        </button>
                      </form>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="border-t border-slate-100 bg-slate-50">
              <td colSpan={2} className="px-5 py-2.5 text-xs font-medium text-slate-500">
                Gesamt ({invoices.length})
              </td>
              <td className="px-5 py-2.5 text-right font-bold text-slate-900 text-sm">
                {formatCurrency(total)}
              </td>
              <td colSpan={3} />
            </tr>
          </tfoot>
        </table>
        </div>
      )}
    </details>
  );
}

export default async function BillingPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { show_cancelled, sort: sortParam, success: successParam, error: errorParam } = await searchParams;
  const showCancelled = show_cancelled === "1";
  const sort = sortParam || "period_desc";

  let eeg = null;
  let runs: BillingRun[] = [];
  let members: Member[] = [];
  let memberNames: Record<string, string> = {};
  let error: string | null = null;
  let invoicesByRun: Record<string, Invoice[]> = {};

  try {
    const [eegResult, runsResult, membersResult] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      listBillingRuns(session.accessToken!, eegId),
      listMembers(session.accessToken!, eegId),
    ]);
    eeg = eegResult;
    runs = runsResult;
    members = membersResult;
    memberNames = Object.fromEntries(
      membersResult.map((m) => [m.id, [m.name1, m.name2].filter(Boolean).join(" ")])
    );
    // Fetch invoices for each run in parallel
    const runInvoices = await Promise.all(
      runs.map((run) =>
        listInvoicesByBillingRun(session.accessToken!, eegId, run.id).catch(() => [] as Invoice[])
      )
    );
    runs.forEach((run, i) => {
      invoicesByRun[run.id] = runInvoices[i];
    });
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    error = apiError.message || "Fehler beim Laden der Abrechnungsdaten.";
  }

  async function markPaidAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const invoiceId = formData.get("invoiceId") as string;
    let errorMsg: string | null = null;
    try {
      await updateInvoiceStatus(session.accessToken!, eegId, invoiceId, "paid");
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Status-Änderung fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
  }

  async function cancelInvoiceAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const invoiceId = formData.get("invoiceId") as string;
    let errorMsg: string | null = null;
    try {
      await updateInvoiceStatus(session.accessToken!, eegId, invoiceId, "cancelled");
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Stornieren fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
  }

  async function sendInvoiceAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const invoiceId = formData.get("invoiceId") as string;
    let errorMsg: string | null = null;
    try {
      await resendInvoice(session.accessToken!, eegId, invoiceId);
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Senden fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) {
      redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
    } else {
      redirect(`/eegs/${eegId}/billing?success=${encodeURIComponent("Rechnung versendet.")}`);
    }
  }

  async function resendInvoiceAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const invoiceId = formData.get("invoiceId") as string;
    let errorMsg: string | null = null;
    try {
      await resendInvoice(session.accessToken!, eegId, invoiceId);
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Erneutes Senden fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) {
      redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
    } else {
      redirect(`/eegs/${eegId}/billing?success=${encodeURIComponent("Rechnung erneut versendet.")}`);
    }
  }

  async function sendRunAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const runId = formData.get("runId") as string;
    let errorMsg: string | null = null;
    let successMsg = "";
    try {
      const result = await sendAllInvoicesByRun(session.accessToken!, eegId, runId);
      successMsg = `${result.sent} versendet, ${result.skipped} übersprungen (keine E-Mail/PDF), ${result.failed} fehlgeschlagen.`;
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Versenden fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) {
      redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
    } else {
      redirect(`/eegs/${eegId}/billing?success=${encodeURIComponent(successMsg)}`);
    }
  }

  async function finalizeRunAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const runId = formData.get("runId") as string;
    let errorMsg: string | null = null;
    try {
      await finalizeBillingRun(session.accessToken!, eegId, runId);
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Finalisieren fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) {
      redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
    } else {
      redirect(`/eegs/${eegId}/billing?success=${encodeURIComponent("Abrechnungslauf finalisiert.")}`);
    }
  }

  async function deleteRunAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const runId = formData.get("runId") as string;
    let errorMsg: string | null = null;
    try {
      await deleteBillingRun(session.accessToken!, eegId, runId);
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Löschen fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) {
      redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
    } else {
      redirect(`/eegs/${eegId}/billing?success=${encodeURIComponent("Entwurf gelöscht.")}`);
    }
  }

  async function cancelRunAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session) return;
    const runId = formData.get("runId") as string;
    let errorMsg: string | null = null;
    try {
      await cancelBillingRun(session.accessToken!, eegId, runId);
    } catch (err: unknown) {
      errorMsg = (err as { message?: string }).message || "Stornieren fehlgeschlagen.";
    }
    revalidatePath(`/eegs/${eegId}/billing`);
    if (errorMsg) {
      redirect(`/eegs/${eegId}/billing?error=${encodeURIComponent(errorMsg)}`);
    } else {
      redirect(`/eegs/${eegId}/billing?success=${encodeURIComponent("Abrechnungslauf storniert.")}`);
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
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          {eeg?.name || eegId}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Abrechnung</span>
      </div>

      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Abrechnung</h1>
        <p className="text-slate-500 mt-1">
          Abrechnungsläufe verwalten für {eeg?.name || "diese Energiegemeinschaft"}.
        </p>
      </div>

      {successParam && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
          <p className="font-medium">Erfolgreich</p>
          <p className="text-sm mt-1">{decodeURIComponent(successParam)}</p>
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

      <div className="space-y-6">
        <BillingRunForm eegId={eegId} members={members} />

        {/* Billing runs */}
        <div>
          {(() => {
            let visibleRuns = showCancelled ? runs : runs.filter((r) => r.status !== "cancelled");
            const statusOrder: Record<string, number> = { finalized: 0, draft: 1, sent: 2, pending: 3, cancelled: 4 };
            visibleRuns = [...visibleRuns].sort((a, b) => {
              switch (sort) {
                case "period_asc":
                  return new Date(a.period_start).getTime() - new Date(b.period_start).getTime();
                case "status":
                  return (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9);
                case "amount_desc":
                  return (b.total_amount || 0) - (a.total_amount || 0);
                case "amount_asc":
                  return (a.total_amount || 0) - (b.total_amount || 0);
                default: // period_desc
                  return new Date(b.period_start).getTime() - new Date(a.period_start).getTime();
              }
            });
            const cancelledCount = runs.filter((r) => r.status === "cancelled").length;

            return (
              <>
                <div className="flex items-center justify-between mb-4 flex-wrap gap-2">
                  <h2 className="text-lg font-semibold text-slate-800">
                    Abrechnungsläufe ({visibleRuns.length}
                    {!showCancelled && cancelledCount > 0 && (
                      <span className="text-sm font-normal text-slate-400 ml-1">
                        , {cancelledCount} storniert ausgeblendet
                      </span>
                    )})
                  </h2>
                  <BillingRunsFilters showCancelled={showCancelled} sort={sort} />
                </div>

                {visibleRuns.length === 0 ? (
                  <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
                    <p className="text-slate-600 font-medium">Noch keine Abrechnungsläufe.</p>
                    <p className="text-slate-400 text-sm mt-1">
                      Starten Sie oben eine Abrechnung, um Rechnungen zu generieren.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {visibleRuns.map((run) => (
                      <RunSection
                        key={run.id}
                        run={run}
                        invoices={invoicesByRun[run.id] || []}
                        memberNames={memberNames}
                        eegId={eegId}
                        markPaidAction={markPaidAction}
                        cancelInvoiceAction={cancelInvoiceAction}
                        sendInvoiceAction={sendInvoiceAction}
                        resendInvoiceAction={resendInvoiceAction}
                        sendRunAction={sendRunAction}
                        finalizeRunAction={finalizeRunAction}
                        deleteRunAction={deleteRunAction}
                        cancelRunAction={cancelRunAction}
                      />
                    ))}
                  </div>
                )}
              </>
            );
          })()}
        </div>
      </div>
    </div>
  );
}
