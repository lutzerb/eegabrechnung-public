"use client";

import { useEffect, useState, Suspense } from "react";
import { useSearchParams } from "next/navigation";

type Status = "loading" | "success" | "error";

function ConfirmForm() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const [status, setStatus] = useState<Status>("loading");
  const [message, setMessage] = useState("");
  const [email, setEmail] = useState("");

  useEffect(() => {
    if (!token) {
      setStatus("error");
      setMessage("Kein Bestätigungslink angegeben.");
      return;
    }
    fetch("/api/portal/email-change/confirm", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    })
      .then(async res => {
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          setStatus("error");
          setMessage((data as { error?: string }).error || "Link ungültig oder abgelaufen.");
          return;
        }
        setEmail((data as { email?: string }).email || "");
        setStatus("success");
      })
      .catch(() => {
        setStatus("error");
        setMessage("Netzwerkfehler. Bitte versuchen Sie es erneut.");
      });
  }, [token]);

  return (
    <div className="min-h-screen bg-slate-50 flex items-center justify-center px-4">
      <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8 max-w-md w-full text-center">
        {status === "loading" && (
          <p className="text-slate-500 text-sm">Bestätigung wird verarbeitet…</p>
        )}
        {status === "success" && (
          <>
            <h1 className="text-xl font-bold text-slate-900 mb-2">E-Mail-Adresse bestätigt</h1>
            <p className="text-slate-500 text-sm">
              Ihre neue E-Mail-Adresse {email && <span className="font-mono">{email}</span>} wurde erfolgreich übernommen.
              Sie können dieses Fenster nun schließen.
            </p>
          </>
        )}
        {status === "error" && (
          <>
            <h1 className="text-xl font-bold text-slate-900 mb-2">Link ungültig</h1>
            <p className="text-slate-500 text-sm">{message}</p>
          </>
        )}
      </div>
    </div>
  );
}

export default function EmailChangeConfirmPage() {
  return (
    <Suspense>
      <ConfirmForm />
    </Suspense>
  );
}
