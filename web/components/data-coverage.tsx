"use client";

import { useState, useEffect } from "react";
import { useSession } from "next-auth/react";

interface CoverageDay {
  date: string;
  count: number;
}

interface CoverageData {
  year: number;
  days: CoverageDay[];
  gruendungsdatum?: string; // YYYY-MM-DD, optional
}

const MONTHS = ["Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];

function isLeapYear(y: number) {
  return y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0);
}

function daysInMonth(year: number, month: number) {
  return new Date(year, month + 1, 0).getDate();
}

export default function DataCoverage({ eegId, refreshKey = 0 }: { eegId: string; refreshKey?: number }) {
  const { data: session } = useSession();
  const currentYear = new Date().getFullYear();
  const [year, setYear] = useState(currentYear);
  const [data, setData] = useState<CoverageData | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!session?.accessToken) return;
    setLoading(true);
    fetch(`/api/eegs/${eegId}/readings/coverage?year=${year}`, {
      headers: { Authorization: `Bearer ${session.accessToken}` },
    })
      .then((r) => r.json())
      .then((d) => setData(d))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [eegId, year, session, refreshKey]);

  const todayStr = new Date().toISOString().slice(0, 10);
  const foundedStr = data?.gruendungsdatum ?? null;

  const countMap = new Map<string, number>();
  if (data?.days) {
    for (const d of data.days) {
      countMap.set(d.date.slice(0, 10), d.count);
    }
  }

  const totalDays = isLeapYear(year) ? 366 : 365;
  let daysWithData = 0;
  let daysExpected = 0; // past days from founding date onwards
  let daysBeforeFounding = 0;

  const months = Array.from({ length: 12 }, (_, m) => {
    const numDays = daysInMonth(year, m);
    const days = Array.from({ length: numDays }, (_, d) => {
      const day = d + 1;
      const dateStr = `${year}-${String(m + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
      const isFuture = dateStr > todayStr;
      const isBeforeFounding = foundedStr !== null && dateStr < foundedStr;
      const count = countMap.get(dateStr);
      const hasData = (count ?? 0) > 0;
      if (!isFuture && isBeforeFounding) {
        daysBeforeFounding++;
      } else if (!isFuture && !isBeforeFounding) {
        daysExpected++;
        if (hasData) daysWithData++;
      }
      return { dateStr, hasData, isFuture, isBeforeFounding };
    });
    return { m, numDays, days };
  });

  const coveragePct = daysExpected > 0
    ? Math.round((daysWithData / daysExpected) * 100)
    : 0;

  const foundingYear = foundedStr ? parseInt(foundedStr.slice(0, 4)) : currentYear - 5;
  const yearOptions = Array.from(
    { length: currentYear - foundingYear + 1 },
    (_, i) => currentYear - i
  );

  return (
    <div className="bg-white rounded-xl border border-slate-200 p-6">
      <div className="flex items-start justify-between mb-5">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-teal-50 flex items-center justify-center flex-shrink-0">
            <svg className="w-5 h-5 text-teal-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <div>
            <h3 className="font-semibold text-slate-900">Datenverfügbarkeit</h3>
            <p className="text-sm text-slate-500 mt-0.5">
              Tage mit vorhandenen Messdaten — grün = Daten vorhanden, rot = fehlend.
            </p>
          </div>
        </div>
        <select
          value={year}
          onChange={(e) => setYear(Number(e.target.value))}
          className="px-3 py-1.5 border border-slate-300 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-teal-500"
        >
          {yearOptions.map((y) => (
            <option key={y} value={y}>{y}</option>
          ))}
        </select>
      </div>

      {loading ? (
        <div className="h-20 flex items-center justify-center text-sm text-slate-400">
          <svg className="animate-spin h-5 w-5 mr-2 text-teal-500" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          Lade Daten...
        </div>
      ) : (
        <>
          {/* Timeline bar */}
          <div className="overflow-x-auto pb-1">
            <div style={{ minWidth: "560px" }}>
              {/* Month labels */}
              <div className="flex mb-1.5">
                {months.map(({ m, numDays }) => (
                  <div
                    key={m}
                    style={{ flex: numDays }}
                    className="text-center overflow-hidden"
                  >
                    <span className="text-xs font-medium text-slate-400">{MONTHS[m]}</span>
                  </div>
                ))}
              </div>

              {/* Day cells — single continuous bar */}
              <div className="flex gap-[1px] h-9">
                {months.map(({ m, days }) =>
                  days.map(({ dateStr, hasData, isFuture, isBeforeFounding }) => (
                    <div
                      key={dateStr}
                      title={
                        isBeforeFounding
                          ? `${dateStr}: vor Gründungsdatum`
                          : isFuture
                          ? `${dateStr}: Zukunft`
                          : hasData
                          ? `${dateStr}: Daten vorhanden`
                          : `${dateStr}: Keine Daten`
                      }
                      className={`flex-1 h-full cursor-default hover:opacity-70 ${
                        isBeforeFounding
                          ? "bg-slate-200"
                          : isFuture
                          ? "bg-slate-100"
                          : hasData
                          ? "bg-emerald-500"
                          : "bg-red-400"
                      }`}
                    />
                  ))
                )}
              </div>

              {/* Month separators — tiny tick marks at month boundaries */}
              <div className="flex mt-1">
                {months.map(({ m, numDays }) => (
                  <div key={m} style={{ flex: numDays }} className="border-l border-slate-200 h-1" />
                ))}
              </div>
            </div>
          </div>

          {/* Summary row */}
          <div className="flex flex-wrap items-center gap-6 mt-4 pt-3 border-t border-slate-100">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-sm bg-emerald-500 flex-shrink-0" />
              <span className="text-xs text-slate-500">
                Daten vorhanden <span className="font-medium text-slate-700">{daysWithData}</span> Tage
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-sm bg-red-400 flex-shrink-0" />
              <span className="text-xs text-slate-500">
                Fehlend <span className="font-medium text-slate-700">{daysExpected - daysWithData}</span> Tage
              </span>
            </div>
            {daysBeforeFounding > 0 && (
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-sm bg-slate-200 flex-shrink-0" />
                <span className="text-xs text-slate-500">
                  Vor Gründung <span className="font-medium text-slate-700">{daysBeforeFounding}</span> Tage
                </span>
              </div>
            )}
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-sm bg-slate-100 flex-shrink-0" />
              <span className="text-xs text-slate-500">
                Zukunft <span className="font-medium text-slate-700">{totalDays - daysExpected - daysBeforeFounding}</span> Tage
              </span>
            </div>
            <div className="flex-1" />
            {daysExpected > 0 && (
              <div className="flex items-center gap-2">
                <div className="w-24 h-1.5 bg-slate-100 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-emerald-500 rounded-full"
                    style={{ width: `${coveragePct}%` }}
                  />
                </div>
                <span className="text-xs font-semibold text-slate-700">{coveragePct}%</span>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
