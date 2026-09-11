"use client";

import { useState, useEffect, useRef, useMemo } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { fmtKwh } from "@/components/energy-charts";

interface Props {
  eegId: string;
}

interface SearchMeterPoint {
  id: string;
  zaehlpunkt: string;
  direction: string;
  member_id: string;
  member_name: string;
}

interface SearchMember {
  id: string;
  name: string;
  email: string;
  mitglieds_nr: string;
}

interface SearchResponse {
  members: SearchMember[];
  meter_points: SearchMeterPoint[];
}

interface MemberMeterPoint {
  id: string;
  meter_id: string;
  direction: string;
}

interface MemberResponse {
  id: string;
  name: string;
  meter_points: MemberMeterPoint[];
}

interface OBISEntry {
  meter_code: string;
  value: number;
  quality: string;
  message_id: string;
  message_created_at: string;
}

interface ReadingRow {
  id: string;
  meter_point_id: string;
  ts: string;
  wh_total: number;
  wh_community: number;
  wh_self: number;
  source: string;
  quality: string;
  obis: OBISEntry[];
}

interface ReadingsResponse {
  readings: ReadingRow[];
  total_count: number;
  limit: number;
  offset: number;
}

const PAGE_SIZE_OPTIONS = [50, 100, 250, 500];

const BASE_COLUMNS: { key: string; label: string }[] = [
  { key: "wh_total", label: "wh_total" },
  { key: "wh_community", label: "wh_community" },
  { key: "wh_self", label: "wh_self" },
  { key: "source", label: "Quelle" },
  { key: "quality", label: "Qualität" },
];

// Sprechende Bezeichnungen der OBIS-Codes, wie sie laut EDA-Energiedaten-Export
// definiert sind. Derselbe Suffix (G.01/G.01T/P.01/…) bedeutet je nach OBIS-Präfix
// (1.9.0 = Bezug, 2.9.0 = Erzeugung) etwas anderes — deshalb ist der volle Code der
// Schlüssel. G.03R teilt sich die Bedeutung von G.03 (Eigendeckung), siehe worker.go.
const OBIS_LABELS: Record<string, string> = {
  "1-1:1.9.0 G.01": "Verbrauch laut Messung",
  "1-1:1.9.0 G.01T": "Verbrauch laut Messung mit Teilnahmefaktor",
  "1-1:1.9.0 P.01": "Restnetzbezug",
  "1-1:2.9.0 G.02": "Erzeugungsanteil je Energiegemeinschaft",
  "1-1:2.9.0 G.03": "Eigendeckung je Energiegemeinschaft",
  "1-1:2.9.0 G.03R": "Eigendeckung je Energiegemeinschaft",
  "1-1:2.9.0 G.01": "Erzeugung laut Messung",
  "1-1:2.9.0 G.01T": "Erzeugung laut Messung mit Teilnahmefaktor",
  "1-1:2.9.0 P.01": "Errechneter Überschuss der Erzeugungsanlage",
  "1-1:2.9.0 P.01T": "Errechneter Restüberschuss bei der Energiegemeinschaft",
};

function obisLabel(code: string): string {
  return OBIS_LABELS[code] ?? code;
}

// Default date range for a freshly opened meter point: yesterday (Vienna calendar day),
// so the page doesn't unconditionally pull a Zählpunkt's entire multi-year history on
// first load. "Yesterday" rather than "today" because today's data is still
// accumulating and usually incomplete/less interesting for a debug view.
function yesterdayRangeVienna(): { from: string; to: string } {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Europe/Vienna",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "";
  const today = new Date(`${get("year")}-${get("month")}-${get("day")}T00:00:00`);
  today.setDate(today.getDate() - 1);
  const y = today.getFullYear();
  const m = String(today.getMonth() + 1).padStart(2, "0");
  const d = String(today.getDate()).padStart(2, "0");
  return { from: `${y}-${m}-${d}T00:00`, to: `${y}-${m}-${d}T23:59` };
}

function formatTs(iso: string): string {
  try {
    return new Date(iso).toLocaleString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      timeZone: "Europe/Vienna",
    });
  } catch {
    return iso;
  }
}

const QUALITY_STYLES: Record<string, string> = {
  L0: "bg-slate-100 text-slate-600",
  L1: "bg-green-50 text-green-700",
  L2: "bg-amber-50 text-amber-700",
  L3: "bg-red-50 text-red-700",
};

function QualityBadge({ quality }: { quality: string }) {
  const cls = QUALITY_STYLES[quality] ?? "bg-slate-100 text-slate-600";
  return (
    <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${cls}`}>
      {quality || "—"}
    </span>
  );
}

export function ReadingsDebugTable({ eegId }: Props) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  // Selection: which meter point is currently being inspected.
  const [selectedMpId, setSelectedMpId] = useState<string | null>(searchParams.get("mp"));
  const [selectedMpLabel, setSelectedMpLabel] = useState<string>(searchParams.get("mpLabel") || "");

  // Two-step picker (only relevant before a meter point is selected).
  const [searchMode, setSearchMode] = useState<"zaehlpunkt" | "member">("zaehlpunkt");
  const [query, setQuery] = useState("");
  const [searchResult, setSearchResult] = useState<SearchResponse | null>(null);
  const [searching, setSearching] = useState(false);
  const [pickedMember, setPickedMember] = useState<SearchMember | null>(null);
  const [memberMeterPoints, setMemberMeterPoints] = useState<MemberMeterPoint[] | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Filters + pagination (persisted in the URL once a meter point is selected).
  const [from, setFrom] = useState(() => searchParams.get("from") || yesterdayRangeVienna().from);
  const [to, setTo] = useState(() => searchParams.get("to") || yesterdayRangeVienna().to);
  const [page, setPage] = useState(Math.max(1, parseInt(searchParams.get("page") || "1", 10) || 1));
  const [pageSize, setPageSize] = useState(() => {
    const parsed = parseInt(searchParams.get("pageSize") || "100", 10);
    return PAGE_SIZE_OPTIONS.includes(parsed) ? parsed : 100;
  });

  // Column visibility — pure display preference, not persisted in the URL.
  const [hiddenColumns, setHiddenColumns] = useState<Set<string>>(new Set());
  const [hiddenObisCodes, setHiddenObisCodes] = useState<Set<string>>(new Set());
  const [obisLatestOnly, setObisLatestOnly] = useState(false);

  const [data, setData] = useState<ReadingsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Keep the URL in sync so the current selection/filters survive a reload and are
  // bookmarkable/shareable — a lightweight replace, no full navigation.
  useEffect(() => {
    const params = new URLSearchParams();
    if (selectedMpId) {
      params.set("mp", selectedMpId);
      if (selectedMpLabel) params.set("mpLabel", selectedMpLabel);
      if (from) params.set("from", from);
      if (to) params.set("to", to);
      params.set("page", String(page));
      params.set("pageSize", String(pageSize));
    }
    const qs = params.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedMpId, selectedMpLabel, from, to, page, pageSize]);

  // Fetch the readings page whenever the selection or filters change.
  useEffect(() => {
    if (!selectedMpId) {
      setData(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setLoadError(null);
    const qs = new URLSearchParams();
    if (from) qs.set("from", from);
    if (to) qs.set("to", to);
    qs.set("limit", String(pageSize));
    qs.set("offset", String((page - 1) * pageSize));
    fetch(`/api/eegs/${eegId}/meter-points/${selectedMpId}/readings?${qs.toString()}`)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((json: ReadingsResponse) => {
        if (!cancelled) setData(json);
      })
      .catch(() => {
        if (!cancelled) {
          setData(null);
          setLoadError("Messwerte konnten nicht geladen werden.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [eegId, selectedMpId, from, to, page, pageSize]);

  // Debounced search-as-you-type against the existing /search endpoint (min 2 chars).
  useEffect(() => {
    if (query.trim().length < 2) {
      setSearchResult(null);
      return;
    }
    setSearching(true);
    debounceRef.current = setTimeout(() => {
      fetch(`/api/eegs/${eegId}/search?q=${encodeURIComponent(query.trim())}`)
        .then((res) => res.json())
        .then((json: SearchResponse) => setSearchResult(json))
        .catch(() => setSearchResult(null))
        .finally(() => setSearching(false));
    }, 500);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, eegId]);

  function pickMeterPoint(id: string, label: string) {
    setSelectedMpId(id);
    setSelectedMpLabel(label);
    setPage(1);
  }

  function pickMember(member: SearchMember) {
    setPickedMember(member);
    setMemberMeterPoints(null);
    fetch(`/api/eegs/${eegId}/members/${member.id}`)
      .then((res) => res.json())
      .then((json: MemberResponse) => setMemberMeterPoints(json.meter_points || []))
      .catch(() => setMemberMeterPoints([]));
  }

  function resetSelection() {
    setSelectedMpId(null);
    setSelectedMpLabel("");
    setQuery("");
    setSearchResult(null);
    setPickedMember(null);
    setMemberMeterPoints(null);
    const r = yesterdayRangeVienna();
    setFrom(r.from);
    setTo(r.to);
    setPage(1);
  }

  const obisCodesOnPage = useMemo(() => {
    if (!data) return [];
    const codes = new Set<string>();
    for (const row of data.readings) {
      for (const entry of row.obis) codes.add(entry.meter_code);
    }
    return Array.from(codes).sort();
  }, [data]);

  function toggleColumn(key: string) {
    setHiddenColumns((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  function toggleObisColumn(code: string) {
    setHiddenObisCodes((prev) => {
      const next = new Set(prev);
      if (next.has(code)) next.delete(code);
      else next.add(code);
      return next;
    });
  }

  const totalCount = data?.total_count ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const currentPage = Math.min(page, totalPages);
  const pageStart = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const pageEnd = totalCount === 0 ? 0 : Math.min(currentPage * pageSize, totalCount);

  // ── Step 1/2: pick a meter point ──────────────────────────────────────
  if (!selectedMpId) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <div className="flex gap-2 mb-4">
          <button
            onClick={() => { setSearchMode("zaehlpunkt"); setQuery(""); setSearchResult(null); setPickedMember(null); setMemberMeterPoints(null); }}
            className={`px-3 py-1.5 text-sm rounded-lg border ${
              searchMode === "zaehlpunkt" ? "bg-slate-800 text-white border-slate-800" : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50"
            }`}
          >
            Nach Zählpunkt
          </button>
          <button
            onClick={() => { setSearchMode("member"); setQuery(""); setSearchResult(null); setPickedMember(null); setMemberMeterPoints(null); }}
            className={`px-3 py-1.5 text-sm rounded-lg border ${
              searchMode === "member" ? "bg-slate-800 text-white border-slate-800" : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50"
            }`}
          >
            Nach Mitglied
          </button>
        </div>

        {searchMode === "zaehlpunkt" || !pickedMember ? (
          <input
            type="search"
            autoFocus
            placeholder={searchMode === "zaehlpunkt" ? "Zählpunktnummer eingeben…" : "Mitgliedsname eingeben…"}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        ) : null}

        {searching && <p className="text-xs text-slate-400 mt-2">Suche…</p>}

        {searchMode === "zaehlpunkt" && searchResult && (
          <ul className="mt-3 divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
            {searchResult.meter_points.length === 0 ? (
              <li className="px-3 py-2 text-sm text-slate-400">Keine Treffer.</li>
            ) : (
              searchResult.meter_points.map((mp) => (
                <li key={mp.id}>
                  <button
                    onClick={() => pickMeterPoint(mp.id, mp.zaehlpunkt)}
                    className="w-full text-left px-3 py-2 hover:bg-slate-50 text-sm flex items-center justify-between"
                  >
                    <span className="font-mono text-slate-800">{mp.zaehlpunkt}</span>
                    <span className="text-slate-500">{mp.member_name}</span>
                  </button>
                </li>
              ))
            )}
          </ul>
        )}

        {searchMode === "member" && !pickedMember && searchResult && (
          <ul className="mt-3 divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
            {searchResult.members.length === 0 ? (
              <li className="px-3 py-2 text-sm text-slate-400">Keine Treffer.</li>
            ) : (
              searchResult.members.map((m) => (
                <li key={m.id}>
                  <button
                    onClick={() => pickMember(m)}
                    className="w-full text-left px-3 py-2 hover:bg-slate-50 text-sm"
                  >
                    {m.name}
                  </button>
                </li>
              ))
            )}
          </ul>
        )}

        {searchMode === "member" && pickedMember && (
          <div className="mt-3">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-slate-700">
                Zählpunkte von <span className="font-medium">{pickedMember.name}</span>
              </p>
              <button
                onClick={() => { setPickedMember(null); setMemberMeterPoints(null); }}
                className="text-xs text-slate-500 hover:text-slate-700 underline"
              >
                Anderes Mitglied
              </button>
            </div>
            {memberMeterPoints === null ? (
              <p className="text-xs text-slate-400">Lädt…</p>
            ) : memberMeterPoints.length === 0 ? (
              <p className="text-xs text-slate-400">Dieses Mitglied hat keine Zählpunkte.</p>
            ) : (
              <ul className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
                {memberMeterPoints.map((mp) => (
                  <li key={mp.id}>
                    <button
                      onClick={() => pickMeterPoint(mp.id, mp.meter_id)}
                      className="w-full text-left px-3 py-2 hover:bg-slate-50 text-sm flex items-center justify-between"
                    >
                      <span className="font-mono text-slate-800">{mp.meter_id}</span>
                      <span className="text-slate-500">{mp.direction === "CONSUMPTION" ? "Verbrauch" : "Erzeugung"}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    );
  }

  // ── Step 2/2: readings table for the selected meter point ────────────
  return (
    <div>
      <div className="flex items-center gap-2 mb-4">
        <span className="inline-flex items-center gap-2 px-3 py-1.5 bg-slate-100 rounded-lg text-sm">
          <span className="font-mono font-medium text-slate-800">{selectedMpLabel}</span>
        </span>
        <button onClick={resetSelection} className="text-xs text-slate-500 hover:text-slate-700 underline">
          Ändern
        </button>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        {/* Filters */}
        <div className="px-4 py-3 border-b border-slate-100 flex flex-wrap gap-3 items-end bg-slate-50/50">
          <div>
            <label className="block text-xs text-slate-500 mb-1">Von</label>
            <input
              type="datetime-local"
              value={from}
              onChange={(e) => { setFrom(e.target.value); setPage(1); }}
              className="text-sm border border-slate-200 rounded-lg px-2 py-1.5"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Bis</label>
            <input
              type="datetime-local"
              value={to}
              onChange={(e) => { setTo(e.target.value); setPage(1); }}
              className="text-sm border border-slate-200 rounded-lg px-2 py-1.5"
            />
          </div>
          <button
            onClick={() => { const r = yesterdayRangeVienna(); setFrom(r.from); setTo(r.to); setPage(1); }}
            className="text-xs text-slate-500 hover:text-slate-700 underline mb-2"
          >
            Auf gestern zurücksetzen
          </button>
          {(from || to) && (
            <button
              onClick={() => { setFrom(""); setTo(""); setPage(1); }}
              className="text-xs text-slate-500 hover:text-slate-700 underline mb-2"
            >
              Zeitraum-Filter entfernen
            </button>
          )}
          <div className="ml-auto flex items-center gap-2 mb-0.5">
            <label className="text-xs text-slate-500">Pro Seite</label>
            <select
              value={pageSize}
              onChange={(e) => { setPageSize(Number(e.target.value) || 100); setPage(1); }}
              className="text-sm border border-slate-200 rounded-lg px-2 py-1.5"
            >
              {PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>{size}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Column toggles */}
        <div className="px-4 py-2 border-b border-slate-100 flex flex-wrap gap-x-4 gap-y-1 text-xs bg-white">
          {BASE_COLUMNS.map((col) => (
            <label key={col.key} className="inline-flex items-center gap-1.5 text-slate-600">
              <input
                type="checkbox"
                checked={!hiddenColumns.has(col.key)}
                onChange={() => toggleColumn(col.key)}
              />
              {col.label}
            </label>
          ))}
          {obisCodesOnPage.length > 0 && (
            <span className="text-slate-300">|</span>
          )}
          {obisCodesOnPage.map((code) => (
            <label key={code} className="inline-flex items-center gap-1.5 text-slate-600" title={code}>
              <input
                type="checkbox"
                checked={!hiddenObisCodes.has(code)}
                onChange={() => toggleObisColumn(code)}
              />
              {obisLabel(code)}
            </label>
          ))}
          {obisCodesOnPage.length > 0 && (
            <>
              <span className="text-slate-300">|</span>
              <label className="inline-flex items-center gap-1.5 text-slate-600">
                <input
                  type="checkbox"
                  checked={obisLatestOnly}
                  onChange={(e) => setObisLatestOnly(e.target.checked)}
                />
                Nur letzten Wert je Zeitstempel (OBIS-Verlauf ausblenden)
              </label>
            </>
          )}
        </div>

        {/* Table */}
        <div className="overflow-x-auto">
          <table className="w-full text-sm min-w-[700px]">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50">
                <th className="text-left px-4 py-3 font-medium text-slate-600 w-40">Zeitpunkt</th>
                {BASE_COLUMNS.filter((c) => !hiddenColumns.has(c.key)).map((col) => (
                  <th key={col.key} className="text-left px-4 py-3 font-medium text-slate-600">{col.label}</th>
                ))}
                {obisCodesOnPage.filter((c) => !hiddenObisCodes.has(c)).map((code) => (
                  <th key={code} className="text-left px-4 py-3 font-medium text-slate-600 w-40" title={code}>
                    <div className="flex flex-col gap-0.5">
                      <span>{obisLabel(code)}</span>
                      <span className="font-mono text-[10px] font-normal text-slate-400">{code}</span>
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {loading ? (
                <tr>
                  <td colSpan={99} className="px-4 py-8 text-center text-slate-400 text-sm">Lädt…</td>
                </tr>
              ) : loadError ? (
                <tr>
                  <td colSpan={99} className="px-4 py-8 text-center text-red-600 text-sm">{loadError}</td>
                </tr>
              ) : !data || data.readings.length === 0 ? (
                <tr>
                  <td colSpan={99} className="px-4 py-8 text-center text-slate-400 text-sm">Keine Messwerte im gewählten Zeitraum.</td>
                </tr>
              ) : (
                data.readings.map((row) => (
                  <tr key={row.id} className="hover:bg-slate-50">
                    <td className="px-4 py-2 text-slate-500 whitespace-nowrap font-mono text-xs">{formatTs(row.ts)}</td>
                    {!hiddenColumns.has("wh_total") && <td className="px-4 py-2 text-slate-700">{fmtKwh(row.wh_total)}</td>}
                    {!hiddenColumns.has("wh_community") && <td className="px-4 py-2 text-slate-700">{fmtKwh(row.wh_community)}</td>}
                    {!hiddenColumns.has("wh_self") && <td className="px-4 py-2 text-slate-700">{fmtKwh(row.wh_self)}</td>}
                    {!hiddenColumns.has("source") && (
                      <td className="px-4 py-2 text-xs text-slate-500 uppercase">{row.source}</td>
                    )}
                    {!hiddenColumns.has("quality") && (
                      <td className="px-4 py-2"><QualityBadge quality={row.quality} /></td>
                    )}
                    {obisCodesOnPage.filter((c) => !hiddenObisCodes.has(c)).map((code) => {
                      let entries = row.obis.filter((o) => o.meter_code === code)
                        .sort((a, b) => b.message_created_at.localeCompare(a.message_created_at));
                      if (obisLatestOnly) entries = entries.slice(0, 1);
                      return (
                        <td key={code} className="px-4 py-2">
                          {entries.length === 0 ? (
                            <span className="text-slate-300">—</span>
                          ) : (
                            <div className="flex flex-col gap-1">
                              {entries.map((e, i) => (
                                <div key={i} className="flex items-center gap-1.5 text-xs">
                                  <span className="text-slate-700">{fmtKwh(e.value)}</span>
                                  <QualityBadge quality={e.quality} />
                                  {i > 0 && (
                                    <span className="text-slate-400" title={e.message_created_at}>
                                      ({formatTs(e.message_created_at)})
                                    </span>
                                  )}
                                </div>
                              ))}
                            </div>
                          )}
                        </td>
                      );
                    })}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-xs text-slate-500">
              {pageStart} bis {pageEnd} von {totalCount} Messwerten
            </p>
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={currentPage <= 1}
                className={`inline-flex items-center rounded-lg px-3 py-1.5 text-sm ${
                  currentPage <= 1
                    ? "pointer-events-none bg-slate-100 text-slate-400"
                    : "bg-white border border-slate-200 text-slate-700 hover:bg-slate-50"
                }`}
              >
                Zurück
              </button>
              <span className="text-sm text-slate-600 px-2">
                Seite {currentPage} von {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={currentPage >= totalPages}
                className={`inline-flex items-center rounded-lg px-3 py-1.5 text-sm ${
                  currentPage >= totalPages
                    ? "pointer-events-none bg-slate-100 text-slate-400"
                    : "bg-white border border-slate-200 text-slate-700 hover:bg-slate-50"
                }`}
              >
                Weiter
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
