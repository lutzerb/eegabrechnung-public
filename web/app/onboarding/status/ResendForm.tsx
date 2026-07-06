"use client";

import { useState } from "react";

export default function ResendForm() {
  const [email, setEmail] = useState("");
  const [eegId, setEegId] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!email.trim()) {
      setError("Bitte geben Sie Ihre E-Mail-Adresse ein.");
      return;
    }
    setLoading(true);
    setError(null);

    try {
      await fetch("/api/public/onboarding/resend-token", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: email.trim(), eeg_id: eegId.trim() }),
      });
      // Always show success to avoid email enumeration
      setSent(true);
    } catch {
      setError("Fehler beim Senden. Bitte versuchen Sie es später erneut.");
    } finally {
      setLoading(false);
    }
  }

  if (sent) {
    return (
      <div className="text-center p-4 bg-green-50 rounded-lg border border-green-200">
        <p className="text-sm text-green-800 font-medium">
          Falls ein Antrag mit dieser E-Mail-Adresse existiert, haben wir
          Ihnen einen neuen Status-Link gesendet.
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1">
          E-Mail-Adresse
        </label>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="Ihre E-Mail-Adresse"
          required
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1">
          EEG-ID (optional)
        </label>
        <input
          type="text"
          value={eegId}
          onChange={(e) => setEegId(e.target.value)}
          placeholder="ID der Energiegemeinschaft"
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
      {error && (
        <p className="text-sm text-red-600">{error}</p>
      )}
      <button
        type="submit"
        disabled={loading}
        className="w-full px-4 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors disabled:opacity-50"
      >
        {loading ? "Wird gesendet…" : "Status-Link zusenden"}
      </button>
    </form>
  );
}
