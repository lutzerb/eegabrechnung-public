"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "next-auth/react";
import Link from "next/link";
import NetzbetreiberSelect from "@/components/netzbetreiber-select";

const GEMEINSCHAFT_ID_RE = /^AT[A-Z0-9]{31}$/;
const MARKTPARTNER_RC_RE = /^RC\d{6}$/;
const MARKTPARTNER_GC_RE = /^GC\d{6}$/;
const MARKTPARTNER_CC_RE = /^CC\d{6}$/;

function validateGemeinschaftId(v: string): string | null {
  if (!v) return null;
  if (!v.startsWith("AT")) return 'Muss mit "AT" beginnen.';
  if (v.length !== 33) return `Muss genau 33 Zeichen lang sein (aktuell: ${v.length}).`;
  if (!GEMEINSCHAFT_ID_RE.test(v)) return "Nur Großbuchstaben und Ziffern erlaubt (nach AT).";
  return null;
}

function validateMarktpartnerId(v: string, typ: string): string | null {
  if (!v) return null;
  const prefix = typ === "GEA" ? "GC" : typ === "BEG" ? "CC" : "RC";
  const re = typ === "GEA" ? MARKTPARTNER_GC_RE : typ === "BEG" ? MARKTPARTNER_CC_RE : MARKTPARTNER_RC_RE;
  if (!v.startsWith(prefix)) return `Muss mit "${prefix}" beginnen (${typ}).`;
  if (v.length !== 8) return `Muss genau 8 Zeichen lang sein (aktuell: ${v.length}).`;
  if (!re.test(v)) return `Format: ${prefix} gefolgt von 6 Ziffern (z.B. ${prefix}105970).`;
  return null;
}

export default function NewEEGPage() {
  const router = useRouter();
  const { data: session } = useSession();

  const [formData, setFormData] = useState({
    name: "",
    gemeinschaft_typ: "EEG" as "EEG" | "GEA" | "BEG",
    gemeinschaft_id: "",
    eda_marktpartner_id: "",
    netzbetreiber: "",
    eda_netzbetreiber_id: "",
  });
  const [touched, setTouched] = useState({ gemeinschaft_id: false, eda_marktpartner_id: false });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const gemeinschaftIdError = validateGemeinschaftId(formData.gemeinschaft_id);
  const marktpartnerIdError = validateMarktpartnerId(formData.eda_marktpartner_id, formData.gemeinschaft_typ);
  const isValid = !gemeinschaftIdError && !marktpartnerIdError;

  const marktpartnerPrefix = formData.gemeinschaft_typ === "GEA" ? "GC" : formData.gemeinschaft_typ === "BEG" ? "CC" : "RC";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTouched({ gemeinschaft_id: true, eda_marktpartner_id: true });
    if (!session?.accessToken || !isValid) return;

    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`/api/eegs`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: formData.name,
          gemeinschaft_typ: formData.gemeinschaft_typ,
          gemeinschaft_id: formData.gemeinschaft_id,
          eda_marktpartner_id: formData.eda_marktpartner_id,
          netzbetreiber: formData.netzbetreiber,
          eda_netzbetreiber_id: formData.eda_netzbetreiber_id,
        }),
      });

      if (!res.ok) {
        let msg = `HTTP ${res.status}`;
        try {
          const body = await res.json();
          msg = body.message || body.error || msg;
        } catch { /* ignore */ }
        throw new Error(msg);
      }

      const eeg = await res.json();
      router.push(`/eegs/${eeg.id}`);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Fehler beim Erstellen.");
    } finally {
      setLoading(false);
    }
  };

  function fieldClass(hasError: boolean) {
    return `w-full px-3 py-2 border rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:border-transparent ${
      hasError
        ? "border-red-400 focus:ring-red-400"
        : "border-slate-300 focus:ring-blue-500"
    }`;
  }

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link
          href="/eegs"
          className="text-sm text-slate-500 hover:text-slate-700 flex items-center gap-1 mb-4"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Zurück zur Übersicht
        </Link>
        <h1 className="text-2xl font-bold text-slate-900">Neue Gemeinschaft anlegen</h1>
        <p className="text-slate-500 mt-1">Geben Sie die Grunddaten der neuen EEG, GEA oder BEG ein.</p>
      </div>

      <div className="max-w-2xl">
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
              <p className="font-medium">Fehler</p>
              <p className="text-sm mt-1">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">

            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Name der Gemeinschaft *
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="z.B. Sonnenkraft Graz"
                className={fieldClass(false)}
              />
            </div>

            {/* Typ */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Art der Gemeinschaft *
              </label>
              <div className="grid grid-cols-3 gap-3">
                {(["EEG", "GEA", "BEG"] as const).map((typ) => (
                  <label
                    key={typ}
                    className={`flex items-start gap-3 px-4 py-3 border rounded-lg cursor-pointer transition-colors ${
                      formData.gemeinschaft_typ === typ
                        ? "border-blue-500 bg-blue-50 text-blue-800"
                        : "border-slate-300 hover:bg-slate-50 text-slate-700"
                    }`}
                  >
                    <input
                      type="radio"
                      name="gemeinschaft_typ"
                      value={typ}
                      checked={formData.gemeinschaft_typ === typ}
                      onChange={() => setFormData({
                        ...formData,
                        gemeinschaft_typ: typ,
                        eda_marktpartner_id: "",
                      })}
                      className="accent-blue-600 mt-0.5 shrink-0"
                    />
                    <div>
                      <span className="block font-medium">{typ}</span>
                      <span className="block text-xs text-slate-500 mt-0.5">
                        {typ === "EEG" ? "Erneuerbare-Energie-Gemeinschaft" : typ === "GEA" ? "Gemeinschaftliche Erzeugungsanlage" : "Bürgerenergiegemeinschaft"}
                      </span>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            {/* Gemeinschafts-ID */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Gemeinschafts-ID *
              </label>
              <input
                type="text"
                required
                value={formData.gemeinschaft_id}
                onChange={(e) => setFormData({ ...formData, gemeinschaft_id: e.target.value.toUpperCase() })}
                onBlur={() => setTouched((t) => ({ ...t, gemeinschaft_id: true }))}
                placeholder="z.B. AT00999900000MUSTERTAL00000000001"
                className={fieldClass(!!(touched.gemeinschaft_id && gemeinschaftIdError))}
              />
              {touched.gemeinschaft_id && gemeinschaftIdError ? (
                <p className="mt-1.5 text-xs text-red-600">{gemeinschaftIdError}</p>
              ) : (
                <p className="mt-1.5 text-xs text-slate-400">AT + 31 alphanumerische Zeichen, gesamt 33</p>
              )}
            </div>

            {/* Marktpartner-ID */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Marktpartner-ID (EC-Nummer) *
              </label>
              <input
                type="text"
                required
                value={formData.eda_marktpartner_id}
                onChange={(e) => setFormData({ ...formData, eda_marktpartner_id: e.target.value.toUpperCase() })}
                onBlur={() => setTouched((t) => ({ ...t, eda_marktpartner_id: true }))}
                placeholder={`z.B. ${marktpartnerPrefix}105970`}
                className={fieldClass(!!(touched.eda_marktpartner_id && marktpartnerIdError))}
              />
              {touched.eda_marktpartner_id && marktpartnerIdError ? (
                <p className="mt-1.5 text-xs text-red-600">{marktpartnerIdError}</p>
              ) : (
                <p className="mt-1.5 text-xs text-slate-400">{marktpartnerPrefix} + 6 Ziffern, gesamt 8 Zeichen</p>
              )}
            </div>

            {/* Netzbetreiber — nicht relevant für BEG (wird pro Zählpunkt aus dem ZP-Präfix abgeleitet) */}
            {formData.gemeinschaft_typ !== "BEG" && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">
                  Netzbetreiber *
                </label>
                <NetzbetreiberSelect
                  required
                  value={formData.netzbetreiber}
                  ecNummer={formData.eda_netzbetreiber_id}
                  onChange={(name, ecNummer) =>
                    setFormData({ ...formData, netzbetreiber: name, eda_netzbetreiber_id: ecNummer })
                  }
                />
                {formData.eda_netzbetreiber_id && (
                  <p className="mt-1.5 text-xs text-slate-400">
                    Netzbetreiber-ID: <span className="font-mono">{formData.eda_netzbetreiber_id}</span>
                  </p>
                )}
              </div>
            )}

            <div className="flex gap-3 pt-2">
              <button
                type="submit"
                disabled={loading || (touched.gemeinschaft_id && !!gemeinschaftIdError) || (touched.eda_marktpartner_id && !!marktpartnerIdError)}
                className="px-6 py-2.5 bg-blue-700 text-white font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              >
                {loading ? (
                  <span className="flex items-center gap-2">
                    <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    Erstelle...
                  </span>
                ) : (
                  `${formData.gemeinschaft_typ} anlegen`
                )}
              </button>
              <Link
                href="/eegs"
                className="px-6 py-2.5 border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-50 transition-colors"
              >
                Abbrechen
              </Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
