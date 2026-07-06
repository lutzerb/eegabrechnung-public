"use client";

import { useState } from "react";

interface Campaign {
  id: string;
  subject: string;
  created_at: string;
  recipient_count: number;
  attachment_count: number;
}

interface CampaignDetail extends Campaign {
  html_body: string;
}

interface Props {
  campaigns: Campaign[];
  eegId: string;
}

function fmtDate(s: string) {
  return new Date(s).toLocaleString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function CampaignHistory({ campaigns, eegId }: Props) {
  const [selected, setSelected] = useState<CampaignDetail | null>(null);
  const [loading, setLoading] = useState(false);

  async function openCampaign(id: string) {
    setLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/communications/${id}`);
      if (res.ok) setSelected(await res.json());
    } finally {
      setLoading(false);
    }
  }

  if (campaigns.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 px-6 py-12 text-center">
        <p className="text-slate-400 text-sm">Noch keine E-Mails versendet.</p>
      </div>
    );
  }

  return (
    <>
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-slate-50 border-b border-slate-200">
              <th className="px-4 py-3 text-left font-medium text-slate-600">Datum</th>
              <th className="px-4 py-3 text-left font-medium text-slate-600">Betreff</th>
              <th className="px-4 py-3 text-right font-medium text-slate-600">Empfänger</th>
              <th className="px-4 py-3 text-right font-medium text-slate-600">Anhänge</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {campaigns.map((c) => (
              <tr
                key={c.id}
                onClick={() => openCampaign(c.id)}
                className="hover:bg-slate-50 cursor-pointer"
              >
                <td className="px-4 py-3 text-slate-500 text-xs whitespace-nowrap font-mono">
                  {fmtDate(c.created_at)}
                </td>
                <td className="px-4 py-3 text-slate-900 font-medium">{c.subject}</td>
                <td className="px-4 py-3 text-right text-slate-600">{c.recipient_count}</td>
                <td className="px-4 py-3 text-right text-slate-600">{c.attachment_count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Loading overlay */}
      {loading && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30">
          <div className="bg-white rounded-xl p-6 shadow-xl">
            <svg className="animate-spin w-6 h-6 text-blue-600 mx-auto" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
          </div>
        </div>
      )}

      {/* Detail modal */}
      {selected && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-xl shadow-xl max-w-2xl w-full max-h-[85vh] flex flex-col">
            <div className="flex items-start justify-between px-6 py-4 border-b border-slate-200">
              <div>
                <p className="text-xs text-slate-400 mb-1">{fmtDate(selected.created_at)} · {selected.recipient_count} Empfänger</p>
                <p className="text-xs text-slate-500 mb-0.5">Betreff</p>
                <p className="font-semibold text-slate-900">{selected.subject}</p>
              </div>
              <button
                type="button"
                onClick={() => setSelected(null)}
                className="text-slate-400 hover:text-slate-700 transition-colors ml-4 flex-shrink-0"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-6">
              <div
                className="prose prose-sm max-w-none text-slate-800"
                dangerouslySetInnerHTML={{ __html: selected.html_body }}
              />
            </div>
          </div>
        </div>
      )}
    </>
  );
}
