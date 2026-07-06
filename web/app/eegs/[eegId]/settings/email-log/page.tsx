"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import Link from "next/link";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ offset?: string }>;
}

interface EmailLogEntry {
  id: string;
  eeg_id: string;
  email_type: string;
  to_address: string;
  subject: string;
  status: "sent" | "failed";
  error_msg?: string;
  member_id?: string;
  invoice_id?: string;
  created_at: string;
}

interface EmailLogResponse {
  total: number;
  entries: EmailLogEntry[];
}

const PAGE_SIZE = 100;

const EMAIL_TYPE_LABELS: Record<string, string> = {
  invoice: "Rechnung",
  campaign: "Kampagne",
  auto_billing_success: "Auto-Abrechnung (Erfolg)",
  auto_billing_warning: "Auto-Abrechnung (Warnung)",
  auto_billing_error: "Auto-Abrechnung (Fehler)",
  gap_alert: "Datenlücken-Alarm",
  eda_error: "EDA Fehler",
  eda_confirmation: "EDA Bestätigung",
  onboarding_token: "Onboarding Link",
  onboarding_admin: "Onboarding Admin",
  onboarding_verify: "E-Mail Verifizierung",
  onboarding_conversion: "Onboarding Genehmigung",
  onboarding_reminder: "Onboarding Erinnerung",
  portal_link: "Portal Login-Link",
};

function formatDate(iso: string) {
  return new Date(iso).toLocaleString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default async function EmailLogPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { offset: offsetParam } = await searchParams;
  const offset = Math.max(0, parseInt(offsetParam ?? "0", 10) || 0);

  const API = process.env.API_INTERNAL_URL || "http://localhost:8080";
  const res = await fetch(
    `${API}/api/v1/eegs/${eegId}/email-log?limit=${PAGE_SIZE}&offset=${offset}`,
    {
      headers: { Authorization: `Bearer ${session.accessToken}` },
      cache: "no-store",
    }
  );

  let data: EmailLogResponse = { total: 0, entries: [] };
  if (res.ok) {
    data = await res.json();
  }

  const prevOffset = Math.max(0, offset - PAGE_SIZE);
  const nextOffset = offset + PAGE_SIZE;
  const hasPrev = offset > 0;
  const hasNext = nextOffset < data.total;

  const failedCount = data.entries.filter((e) => e.status === "failed").length;

  return (
    <div className="p-8 max-w-6xl">
      {/* Breadcrumb */}
      <div className="mb-6 flex items-center gap-2 text-sm text-slate-500">
        <Link href="/eegs" className="hover:text-slate-700">Energiegemeinschaften</Link>
        <span>/</span>
        <Link href={`/eegs/${eegId}/settings`} className="hover:text-slate-700">Einstellungen</Link>
        <span>/</span>
        <span className="text-slate-900 font-medium">E-Mail-Protokoll</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">E-Mail-Protokoll</h1>
          <p className="text-sm text-slate-500 mt-1">
            {data.total} Einträge gesamt
            {failedCount > 0 && (
              <span className="ml-2 text-red-600 font-medium">· {failedCount} Fehler auf dieser Seite</span>
            )}
          </p>
        </div>
      </div>

      {data.entries.length === 0 ? (
        <div className="text-center py-16 text-slate-400">Noch keine E-Mails protokolliert.</div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-slate-50 border-b border-slate-200">
                <th className="text-left px-4 py-3 font-semibold text-slate-600 w-36">Zeitpunkt</th>
                <th className="text-left px-4 py-3 font-semibold text-slate-600 w-36">Typ</th>
                <th className="text-left px-4 py-3 font-semibold text-slate-600">Empfänger</th>
                <th className="text-left px-4 py-3 font-semibold text-slate-600">Betreff</th>
                <th className="text-left px-4 py-3 font-semibold text-slate-600 w-24">Status</th>
              </tr>
            </thead>
            <tbody>
              {data.entries.map((entry) => (
                <>
                  <tr
                    key={entry.id}
                    className={`border-b border-slate-100 ${entry.status === "failed" ? "bg-red-50" : "hover:bg-slate-50"}`}
                  >
                    <td className="px-4 py-3 text-slate-500 whitespace-nowrap font-mono text-xs">
                      {formatDate(entry.created_at)}
                    </td>
                    <td className="px-4 py-3 text-slate-700">
                      {EMAIL_TYPE_LABELS[entry.email_type] ?? entry.email_type}
                    </td>
                    <td className="px-4 py-3 text-slate-700 font-mono text-xs">{entry.to_address}</td>
                    <td className="px-4 py-3 text-slate-600 truncate max-w-xs" title={entry.subject}>
                      {entry.subject}
                    </td>
                    <td className="px-4 py-3">
                      {entry.status === "sent" ? (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700">
                          Gesendet
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700">
                          Fehler
                        </span>
                      )}
                    </td>
                  </tr>
                  {entry.status === "failed" && entry.error_msg && (
                    <tr key={entry.id + "-err"} className="bg-red-50 border-b border-slate-100">
                      <td colSpan={5} className="px-4 pb-3 pt-0">
                        <div className="text-xs text-red-700 bg-red-100 rounded px-3 py-2 font-mono break-all">
                          {entry.error_msg}
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {(hasPrev || hasNext) && (
        <div className="flex items-center justify-between mt-4">
          <div className="text-sm text-slate-500">
            {offset + 1}–{Math.min(offset + PAGE_SIZE, data.total)} von {data.total}
          </div>
          <div className="flex gap-2">
            {hasPrev && (
              <Link
                href={`/eegs/${eegId}/settings/email-log?offset=${prevOffset}`}
                className="px-3 py-1.5 text-sm border border-slate-300 rounded-lg hover:bg-slate-50"
              >
                Zurück
              </Link>
            )}
            {hasNext && (
              <Link
                href={`/eegs/${eegId}/settings/email-log?offset=${nextOffset}`}
                className="px-3 py-1.5 text-sm border border-slate-300 rounded-lg hover:bg-slate-50"
              >
                Weiter
              </Link>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
