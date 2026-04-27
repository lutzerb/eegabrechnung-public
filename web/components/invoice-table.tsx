"use client";

import { useState } from "react";
import type { Invoice } from "@/lib/api";

interface InvoiceTableProps {
  invoices: Invoice[];
  memberNames?: Record<string, string>;
}

function StatusBadge({ status }: { status: string }) {
  const statusMap: Record<string, { label: string; className: string }> = {
    draft: { label: "Entwurf", className: "bg-slate-100 text-slate-600" },
    pending: { label: "Ausstehend", className: "bg-yellow-50 text-yellow-700" },
    sent: { label: "Versendet", className: "bg-blue-50 text-blue-700" },
    paid: { label: "Bezahlt", className: "bg-green-50 text-green-700" },
    cancelled: {
      label: "Storniert",
      className: "bg-red-50 text-red-700",
    },
  };

  const config = statusMap[status.toLowerCase()] || {
    label: status,
    className: "bg-slate-100 text-slate-600",
  };

  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${config.className}`}
    >
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
  return new Intl.NumberFormat("de-AT", {
    style: "currency",
    currency: "EUR",
  }).format(amount);
}

interface PdfModalProps {
  url: string;
  title: string;
  onClose: () => void;
}

function PdfModal({ url, title, onClose }: PdfModalProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex flex-col bg-black/70"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      {/* Toolbar */}
      <div className="flex items-center justify-between px-4 py-2 bg-slate-800 text-white shrink-0">
        <span className="text-sm font-medium truncate">{title}</span>
        <div className="flex items-center gap-3 ml-4">
          <a
            href={url}
            download
            className="text-xs px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
          >
            Herunterladen
          </a>
          <button
            onClick={onClose}
            className="text-slate-300 hover:text-white transition-colors p-1"
            aria-label="Schließen"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>
      {/* PDF iframe */}
      <iframe
        src={url}
        className="flex-1 w-full border-0"
        title={title}
      />
    </div>
  );
}

export default function InvoiceTable({ invoices, memberNames = {} }: InvoiceTableProps) {
  const [pdfModal, setPdfModal] = useState<{ url: string; title: string } | null>(null);

  if (invoices.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
        <svg
          className="mx-auto w-12 h-12 text-slate-300 mb-3"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z"
          />
        </svg>
        <p className="text-slate-600 font-medium">Keine Rechnungen vorhanden.</p>
        <p className="text-slate-400 text-sm mt-1">
          Starten Sie eine Abrechnung, um Rechnungen zu generieren.
        </p>
      </div>
    );
  }

  const total = invoices.reduce((sum, inv) => sum + (inv.total_amount || 0), 0);

  return (
    <>
      {pdfModal && (
        <PdfModal
          url={pdfModal.url}
          title={pdfModal.title}
          onClose={() => setPdfModal(null)}
        />
      )}

      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              <th className="text-left px-6 py-3.5 font-medium text-slate-600">
                Mitglied
              </th>
              <th className="text-left px-6 py-3.5 font-medium text-slate-600">
                Zeitraum
              </th>
              <th className="text-right px-6 py-3.5 font-medium text-slate-600">
                Betrag
              </th>
              <th className="text-left px-6 py-3.5 font-medium text-slate-600">
                Status
              </th>
              <th className="text-left px-6 py-3.5 font-medium text-slate-600">
                Erstellt am
              </th>
              <th className="text-right px-6 py-3.5 font-medium text-slate-600">
                Aktionen
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {invoices.map((invoice) => {
              const memberName = memberNames[invoice.member_id] || invoice.member_id;
              const pdfUrl = `/api/eegs/${invoice.eeg_id}/invoices/${invoice.id}/pdf`;
              const pdfTitle = `Rechnung ${invoice.id.slice(0, 8)} – ${memberName}`;
              return (
                <tr
                  key={invoice.id}
                  className="hover:bg-slate-50 transition-colors"
                >
                  <td className="px-6 py-4">
                    <p className="font-medium text-slate-900">{memberName}</p>
                    <p className="text-xs text-slate-400 font-mono">
                      {invoice.id.slice(0, 8)}...
                    </p>
                  </td>
                  <td className="px-6 py-4 text-slate-600">
                    {formatDate(invoice.period_start)} –{" "}
                    {formatDate(invoice.period_end)}
                  </td>
                  <td className="px-6 py-4 text-right font-medium text-slate-900">
                    {formatCurrency(invoice.total_amount)}
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={invoice.status} />
                  </td>
                  <td className="px-6 py-4 text-slate-500">
                    {invoice.created_at ? formatDate(invoice.created_at) : "—"}
                  </td>
                  <td className="px-6 py-4 text-right">
                    {invoice.pdf_path ? (
                      <button
                        onClick={() => setPdfModal({ url: pdfUrl, title: pdfTitle })}
                        className="px-3 py-1.5 text-xs font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                      >
                        PDF anzeigen
                      </button>
                    ) : (
                      <span className="text-xs text-slate-400">—</span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
          <tfoot>
            <tr className="border-t border-slate-200 bg-slate-50">
              <td
                colSpan={2}
                className="px-6 py-3.5 text-sm font-medium text-slate-600"
              >
                Gesamt ({invoices.length} Rechnungen)
              </td>
              <td className="px-6 py-3.5 text-right font-bold text-slate-900">
                {formatCurrency(total)}
              </td>
              <td colSpan={3}></td>
            </tr>
          </tfoot>
        </table>
      </div>
    </>
  );
}
