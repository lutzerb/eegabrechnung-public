"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "next-auth/react";

interface SearchResult {
  members: Array<{ id: string; name: string; email: string; mitglieds_nr: string }>;
  meter_points: Array<{ id: string; zaehlpunkt: string; direction: string; member_id: string; member_name: string }>;
  invoices: Array<{ id: string; invoice_number: number; total_amount: number; member_id: string; status: string }>;
}

interface Props {
  eegId: string;
  onClose: () => void;
}

export function SearchOverlay({ eegId, onClose }: Props) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();
  const { data: session } = useSession();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const search = useCallback(
    async (q: string) => {
      if (q.length < 2) {
        setResults(null);
        return;
      }
      setLoading(true);
      try {
        const res = await fetch(
          `/api/eegs/${eegId}/search?q=${encodeURIComponent(q)}`,
          { headers: session?.accessToken ? { Authorization: `Bearer ${session.accessToken}` } : {} }
        );
        if (res.ok) {
          const data = await res.json();
          setResults(data);
        }
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    },
    [eegId, session?.accessToken]
  );

  function handleInput(e: React.ChangeEvent<HTMLInputElement>) {
    const q = e.target.value;
    setQuery(q);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => search(q), 300);
  }

  function navigate(href: string) {
    router.push(href);
    onClose();
  }

  const hasResults =
    results &&
    (results.members.length > 0 ||
      results.meter_points.length > 0 ||
      results.invoices.length > 0);

  return (
    <div
      className="fixed inset-0 z-50 bg-black/40 flex items-start justify-center pt-20 px-4"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-xl overflow-hidden">
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-200">
          <svg className="w-5 h-5 text-slate-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={handleInput}
            placeholder="Mitglieder, Rechnungen, Zählpunkte suchen…"
            className="flex-1 text-slate-900 placeholder-slate-400 outline-none text-sm"
          />
          {loading && (
            <svg className="w-4 h-4 text-slate-400 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          )}
          <button
            onClick={onClose}
            className="text-xs text-slate-400 hover:text-slate-600 px-1.5 py-0.5 border border-slate-200 rounded"
          >
            Esc
          </button>
        </div>

        {/* Results */}
        {query.length >= 2 && (
          <div className="max-h-96 overflow-y-auto">
            {!hasResults && !loading && (
              <p className="px-4 py-8 text-center text-sm text-slate-400">Keine Ergebnisse gefunden.</p>
            )}

            {results && results.members.length > 0 && (
              <div>
                <p className="px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wide bg-slate-50 border-b border-slate-100">
                  Mitglieder
                </p>
                {results.members.map((m) => (
                  <button
                    key={m.id}
                    onClick={() => navigate(`/eegs/${eegId}/members/${m.id}`)}
                    className="w-full flex items-center gap-3 px-4 py-3 hover:bg-blue-50 transition-colors text-left border-b border-slate-50"
                  >
                    <div className="w-7 h-7 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
                      <svg className="w-3.5 h-3.5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-slate-900 truncate">{m.name}</p>
                      <p className="text-xs text-slate-400 truncate">{m.email || m.mitglieds_nr || ""}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}

            {results && results.meter_points.length > 0 && (
              <div>
                <p className="px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wide bg-slate-50 border-b border-slate-100">
                  Zählpunkte
                </p>
                {results.meter_points.map((mp) => (
                  <button
                    key={mp.id}
                    onClick={() => navigate(`/eegs/${eegId}/members/${mp.member_id}`)}
                    className="w-full flex items-center gap-3 px-4 py-3 hover:bg-blue-50 transition-colors text-left border-b border-slate-50"
                  >
                    <div className="w-7 h-7 rounded-full bg-green-100 flex items-center justify-center flex-shrink-0">
                      <svg className="w-3.5 h-3.5 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z" />
                      </svg>
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-mono text-slate-900 truncate">{mp.zaehlpunkt}</p>
                      <p className="text-xs text-slate-400 truncate">{mp.member_name}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}

            {results && results.invoices.length > 0 && (
              <div>
                <p className="px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wide bg-slate-50 border-b border-slate-100">
                  Rechnungen
                </p>
                {results.invoices.map((inv) => (
                  <button
                    key={inv.id}
                    onClick={() => navigate(`/eegs/${eegId}/billing`)}
                    className="w-full flex items-center gap-3 px-4 py-3 hover:bg-blue-50 transition-colors text-left border-b border-slate-50"
                  >
                    <div className="w-7 h-7 rounded-full bg-amber-100 flex items-center justify-center flex-shrink-0">
                      <svg className="w-3.5 h-3.5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-slate-900">#{inv.invoice_number}</p>
                      <p className="text-xs text-slate-400">
                        {new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(inv.total_amount)} · {inv.status}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {query.length < 2 && (
          <p className="px-4 py-6 text-center text-xs text-slate-400">
            Mindestens 2 Zeichen eingeben…
          </p>
        )}
      </div>
    </div>
  );
}
