"use client";

import { useState } from "react";

interface EegChoice {
  eeg_id: string;
  eeg_name: string;
}

type Step = "email" | "choose" | "sent";

export default function PortalLoginPage() {
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");
  const [choices, setChoices] = useState<EegChoice[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleEmailSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/portal/request-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        setError((d as { error?: string }).error || "Fehler beim Senden.");
        return;
      }
      const data = await res.json();
      if (data.choices && data.choices.length > 1) {
        setChoices(data.choices);
        setStep("choose");
      } else {
        setStep("sent");
      }
    } catch {
      setError("Netzwerkfehler.");
    } finally {
      setLoading(false);
    }
  }

  async function handleChoiceSelect(eegId: string) {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/portal/request-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, eeg_id: eegId }),
      });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        setError((d as { error?: string }).error || "Fehler beim Senden.");
        return;
      }
      setStep("sent");
    } catch {
      setError("Netzwerkfehler.");
    } finally {
      setLoading(false);
    }
  }

  if (step === "sent") {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="bg-white rounded-xl border border-slate-200 p-8 max-w-md w-full text-center">
          <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </div>
          <h1 className="text-xl font-bold text-slate-900 mb-2">E-Mail gesendet</h1>
          <p className="text-slate-500 text-sm">
            Falls Ihre E-Mail-Adresse in unserem System hinterlegt ist, haben Sie soeben einen Login-Link erhalten.
          </p>
          <p className="text-slate-400 text-xs mt-3">Der Link ist 30 Minuten gültig.</p>
        </div>
      </div>
    );
  }

  if (step === "choose") {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="bg-white rounded-xl border border-slate-200 p-8 max-w-md w-full">
          <div className="mb-6 text-center">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <h1 className="text-xl font-bold text-slate-900">Energiegemeinschaft wählen</h1>
            <p className="text-slate-500 text-sm mt-1">
              Ihre E-Mail-Adresse ist in mehreren Energiegemeinschaften hinterlegt.<br />
              Für welche möchten Sie den Login-Link anfordern?
            </p>
          </div>
          {error && <p className="text-sm text-red-600 mb-4">{error}</p>}
          <div className="space-y-2">
            {choices.map((c) => (
              <button
                key={c.eeg_id}
                onClick={() => handleChoiceSelect(c.eeg_id)}
                disabled={loading}
                className="w-full text-left px-4 py-3 rounded-lg border border-slate-200 hover:border-blue-400 hover:bg-blue-50 transition-colors disabled:opacity-60 group"
              >
                <span className="text-sm font-medium text-slate-800 group-hover:text-blue-700">
                  {c.eeg_name}
                </span>
              </button>
            ))}
          </div>
          <button
            onClick={() => { setStep("email"); setError(null); }}
            className="mt-4 text-xs text-slate-400 hover:text-slate-600 w-full text-center"
          >
            ← Zurück
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="bg-white rounded-xl border border-slate-200 p-8 max-w-md w-full">
        <div className="mb-6 text-center">
          <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
          </div>
          <h1 className="text-xl font-bold text-slate-900">Mitglieder-Portal</h1>
          <p className="text-slate-500 text-sm mt-1">
            Geben Sie Ihre E-Mail-Adresse ein. Wir senden Ihnen einen Login-Link.
          </p>
        </div>
        <form onSubmit={handleEmailSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">E-Mail-Adresse</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
              placeholder="ihre@email.at"
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 transition-colors disabled:opacity-60"
          >
            {loading ? "Senden…" : "Login-Link anfordern"}
          </button>
        </form>
      </div>
    </div>
  );
}
