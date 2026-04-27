"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "next-auth/react";
import Link from "next/link";

export default function NewEEGPage() {
  const router = useRouter();
  const { data: session } = useSession();

  const [formData, setFormData] = useState({
    name: "",
    gemeinschaft_id: "",
    netzbetreiber: "",
    energy_price: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session?.accessToken) return;

    setLoading(true);
    setError(null);

    try {
      const baseUrl =
        process.env.NEXT_PUBLIC_API_URL || "http://localhost:8101";
      const res = await fetch(`${baseUrl}/api/v1/eegs`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${session.accessToken}`,
        },
        body: JSON.stringify({
          name: formData.name,
          gemeinschaft_id: formData.gemeinschaft_id,
          netzbetreiber: formData.netzbetreiber,
          energy_price: parseFloat(formData.energy_price),
        }),
      });

      if (!res.ok) {
        let msg = `HTTP ${res.status}`;
        try {
          const body = await res.json();
          msg = body.message || body.error || msg;
        } catch {
          // ignore
        }
        throw new Error(msg);
      }

      const eeg = await res.json();
      router.push(`/eegs/${eeg.id}`);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Fehler beim Erstellen der Energiegemeinschaft.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link
          href="/eegs"
          className="text-sm text-slate-500 hover:text-slate-700 flex items-center gap-1 mb-4"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 19l-7-7 7-7"
            />
          </svg>
          Zurück zu EEGs
        </Link>
        <h1 className="text-2xl font-bold text-slate-900">
          Neue Energiegemeinschaft erstellen
        </h1>
        <p className="text-slate-500 mt-1">
          Geben Sie die Grunddaten der neuen Energiegemeinschaft ein.
        </p>
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
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Name der Energiegemeinschaft *
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="z.B. Sonnenkraft Graz"
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Gemeinschafts-ID *
              </label>
              <input
                type="text"
                required
                value={formData.gemeinschaft_id}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    gemeinschaft_id: e.target.value,
                  })
                }
                placeholder="z.B. EEG-2024-0001"
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Netzbetreiber *
              </label>
              <input
                type="text"
                required
                value={formData.netzbetreiber}
                onChange={(e) =>
                  setFormData({ ...formData, netzbetreiber: e.target.value })
                }
                placeholder="z.B. Netz Graz GmbH"
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Energiepreis (ct/kWh) *
              </label>
              <div className="relative">
                <input
                  type="number"
                  required
                  step="0.001"
                  min="0"
                  value={formData.energy_price}
                  onChange={(e) =>
                    setFormData({ ...formData, energy_price: e.target.value })
                  }
                  placeholder="z.B. 8.5"
                  className="w-full px-3 py-2 pr-16 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">
                  ct/kWh
                </span>
              </div>
            </div>

            <div className="flex gap-3 pt-2">
              <button
                type="submit"
                disabled={loading}
                className="px-6 py-2.5 bg-blue-700 text-white font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              >
                {loading ? (
                  <span className="flex items-center gap-2">
                    <svg
                      className="animate-spin h-4 w-4"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      />
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                      />
                    </svg>
                    Erstelle...
                  </span>
                ) : (
                  "Energiegemeinschaft erstellen"
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
