"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface SaldenEintrag {
  konto_id: string;
  nummer: string;
  name: string;
  typ: string;
  einnahmen: number;
  ausgaben: number;
  saldo: number;
}

interface UVAPeriode {
  id: string;
  jahr: number;
  quartal?: number;
  monat?: number;
  periodentyp: string;
  eingereicht_am?: string;
  kennzahlen?: { zahllast: number };
}

interface BankTransaktion {
  id: string;
  match_status: string;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

export default function EADashboardPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const year = new Date().getFullYear();

  const [saldenliste, setSaldenliste] = useState<SaldenEintrag[]>([]);
  const [uvas, setUvas] = useState<UVAPeriode[]>([]);
  const [offeneTransaktionen, setOffeneTransaktionen] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [sRes, uRes, tRes] = await Promise.allSettled([
          fetch(`/api/eegs/${eegId}/ea/saldenliste?jahr=${year}`),
          fetch(`/api/eegs/${eegId}/ea/uva`),
          fetch(`/api/eegs/${eegId}/ea/bank/transaktionen?status=offen`),
        ]);
        if (sRes.status === "fulfilled" && sRes.value.ok) { const d = await sRes.value.json(); if (Array.isArray(d)) setSaldenliste(d); }
        if (uRes.status === "fulfilled" && uRes.value.ok) setUvas(await uRes.value.json());
        if (tRes.status === "fulfilled" && tRes.value.ok) {
          const t = await tRes.value.json();
          setOffeneTransaktionen(Array.isArray(t) ? t.length : 0);
        }
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [eegId, year]);

  const summeEinnahmen = saldenliste.filter((k) => k.typ === "EINNAHME").reduce((s, k) => s + k.einnahmen, 0);
  const summeAusgaben = saldenliste.filter((k) => k.typ === "AUSGABE").reduce((s, k) => s + k.ausgaben, 0);
  const gewinnVerlust = summeEinnahmen - summeAusgaben;

  const offeneUvas = uvas.filter((u) => !u.eingereicht_am);
  const zahllastGesamt = uvas
    .filter((u) => !u.eingereicht_am)
    .reduce((sum, u) => sum + (u.kennzahlen?.zahllast ?? 0), 0);

  const navItems = [
    { href: "ea/buchungen", label: "Journal", desc: "Alle Buchungen", color: "blue" },
    { href: "ea/buchungen/neu", label: "Buchung erfassen", desc: "Neue Einnahme/Ausgabe", color: "green" },
    { href: "ea/konten", label: "Kontenplan", desc: "Konten verwalten", color: "slate" },
    { href: "ea/saldenliste", label: "Saldenliste", desc: "Alle Konten auf einen Blick", color: "teal" },
    { href: "ea/jahresabschluss", label: "Jahresabschluss", desc: "E/A Jahresauswertung", color: "emerald" },
    { href: "ea/uva", label: "UVA", desc: "Umsatzsteuervoranmeldung", color: "purple" },
    { href: "ea/erklaerungen", label: "Jahreserklärungen", desc: "U1 / K1 Grundlagen", color: "indigo" },
    { href: "ea/import", label: "Rechnungsimport", desc: "EEG-Rechnungen importieren", color: "amber" },
    { href: "ea/bank", label: "Bankimport", desc: "Kontoauszug importieren + zuordnen", color: "orange" },
    { href: "ea/changelog", label: "Audit-Trail", desc: "Alle Buchungsänderungen (BAO §131)", color: "zinc" },
    { href: "ea/settings", label: "E/A-Einstellungen", desc: "Steuernummer, UVA-Periode", color: "rose" },
  ];

  const colorMap: Record<string, string> = {
    blue: "bg-blue-50 group-hover:bg-blue-100 text-blue-700",
    green: "bg-green-50 group-hover:bg-green-100 text-green-700",
    slate: "bg-slate-50 group-hover:bg-slate-100 text-slate-600",
    teal: "bg-teal-50 group-hover:bg-teal-100 text-teal-700",
    emerald: "bg-emerald-50 group-hover:bg-emerald-100 text-emerald-700",
    purple: "bg-purple-50 group-hover:bg-purple-100 text-purple-700",
    indigo: "bg-indigo-50 group-hover:bg-indigo-100 text-indigo-700",
    amber: "bg-amber-50 group-hover:bg-amber-100 text-amber-700",
    orange: "bg-orange-50 group-hover:bg-orange-100 text-orange-700",
    rose: "bg-rose-50 group-hover:bg-rose-100 text-rose-700",
    zinc: "bg-zinc-50 group-hover:bg-zinc-100 text-zinc-600",
  };

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          Übersicht
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">E/A-Buchhaltung</span>
      </div>

      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900">Einnahmen-Ausgaben-Buchhaltung</h1>
        <p className="text-slate-500 mt-1">Steuerliche Buchhaltung nach IST-Prinzip, Geschäftsjahr {year}</p>
      </div>

      {/* KPI Cards */}
      {loading ? (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-slate-200 p-5 animate-pulse">
              <div className="h-3 bg-slate-200 rounded w-24 mb-2" />
              <div className="h-7 bg-slate-200 rounded w-16" />
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          <div className="bg-white rounded-xl border border-slate-200 p-5">
            <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Einnahmen {year}</p>
            <p className="text-2xl font-bold text-slate-900 mt-1">
              {saldenliste.length > 0 ? fmt(summeEinnahmen) : "—"}
            </p>
          </div>
          <div className="bg-white rounded-xl border border-slate-200 p-5">
            <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Ausgaben {year}</p>
            <p className="text-2xl font-bold text-slate-900 mt-1">
              {saldenliste.length > 0 ? fmt(summeAusgaben) : "—"}
            </p>
          </div>
          <div className="bg-white rounded-xl border border-slate-200 p-5">
            <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Gewinn/Verlust {year}</p>
            <p className={`text-2xl font-bold mt-1 ${gewinnVerlust >= 0 ? "text-green-700" : "text-red-600"}`}>
              {saldenliste.length > 0 ? fmt(gewinnVerlust) : "—"}
            </p>
          </div>
          <div className="bg-white rounded-xl border border-slate-200 p-5">
            <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">USt-Zahllast offen</p>
            <p className="text-2xl font-bold text-slate-900 mt-1">
              {offeneUvas.length > 0 ? fmt(zahllastGesamt) : "—"}
            </p>
          </div>
        </div>
      )}

      {/* Alerts */}
      {(offeneUvas.length > 0 || offeneTransaktionen > 0) && (
        <div className="mb-8 space-y-2">
          {offeneUvas.length > 0 && (
            <div className="rounded-lg border border-orange-200 bg-orange-50 p-4 flex items-start gap-3">
              <span className="w-2 h-2 rounded-full mt-1.5 bg-orange-400 flex-shrink-0" />
              <div>
                <p className="font-medium text-sm text-orange-800">
                  {offeneUvas.length} UVA-Periode{offeneUvas.length !== 1 ? "n" : ""} noch nicht eingereicht
                </p>
                <Link href={`/eegs/${eegId}/ea/uva`} className="text-xs text-orange-700 underline mt-1 inline-block">
                  Zur UVA →
                </Link>
              </div>
            </div>
          )}
          {offeneTransaktionen > 0 && (
            <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 flex items-start gap-3">
              <span className="w-2 h-2 rounded-full mt-1.5 bg-blue-400 flex-shrink-0" />
              <div>
                <p className="font-medium text-sm text-blue-800">
                  {offeneTransaktionen} Banktransaktion{offeneTransaktionen !== 1 ? "en" : ""} noch nicht zugeordnet
                </p>
                <Link href={`/eegs/${eegId}/ea/bank`} className="text-xs text-blue-700 underline mt-1 inline-block">
                  Zum Bankimport →
                </Link>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Navigation Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {navItems.map((item) => (
          <Link
            key={item.href}
            href={`/eegs/${eegId}/${item.href}`}
            className="bg-white rounded-xl border border-slate-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
          >
            <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-3 transition-colors ${colorMap[item.color]}`}>
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                  d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
              </svg>
            </div>
            <p className="font-medium text-slate-900">{item.label}</p>
            <p className="text-sm text-slate-500 mt-0.5">{item.desc}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
