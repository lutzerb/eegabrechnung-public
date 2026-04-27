"use client";

import { useState } from "react";
import { validateIBAN, validateBIC, validateZaehlpunkt, validateUIDNummer, formatIBAN } from "@/lib/validation";

interface MeterPointEntry {
  zaehlpunkt: string;
  direction: string;
  generationType: string;
  participationFactor: number;
}

interface FormData {
  vorname: string;
  nachname: string;
  name1: string;
  name2: string;
  email: string;
  phone: string;
  strasse: string;
  plz: string;
  ort: string;
  iban: string;
  bic: string;
  memberType: string;
  businessRole: string;
  useVat: boolean;
  uidNummer: string;
  beitrittsDatum: string;
  meterPoints: MeterPointEntry[];
}

interface PublicDocument {
  id: string;
  title: string;
  filename: string;
  mime_type: string;
}

interface Props {
  eegId: string;
  eegName: string;
  contractText?: string;
  documents?: PublicDocument[];
  verifiedEmail?: string;
  verifiedName1?: string;
  verifiedName2?: string;
}

const STEPS = [
  "Persönliche Daten",
  "Adresse & Bankdaten",
  "Zählpunkte",
  "Vertrag & Unterzeichnung",
];

function StepIndicator({
  currentStep,
  totalSteps,
}: {
  currentStep: number;
  totalSteps: number;
}) {
  return (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-2">
        {STEPS.map((label, index) => {
          const stepNum = index + 1;
          const isCompleted = stepNum < currentStep;
          const isCurrent = stepNum === currentStep;
          return (
            <div key={index} className="flex flex-col items-center flex-1">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium mb-1
                  ${isCompleted ? "bg-blue-600 text-white" : isCurrent ? "bg-blue-100 text-blue-700 border-2 border-blue-600" : "bg-slate-100 text-slate-400"}`}
              >
                {isCompleted ? (
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2.5}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                ) : (
                  stepNum
                )}
              </div>
              <span
                className={`text-xs hidden sm:block text-center ${isCurrent ? "text-blue-700 font-medium" : "text-slate-400"}`}
              >
                {label}
              </span>
            </div>
          );
        })}
      </div>
      <div className="w-full bg-slate-200 rounded-full h-1.5 mt-1">
        <div
          className="bg-blue-600 h-1.5 rounded-full transition-all duration-300"
          style={{
            width: `${Math.min(((currentStep - 1) / (totalSteps - 1)) * 100, 100)}%`,
          }}
        />
      </div>
    </div>
  );
}

function FieldError({ msg }: { msg?: string }) {
  if (!msg) return null;
  return <p className="mt-1 text-sm text-red-600">{msg}</p>;
}

export default function OnboardingForm({
  eegId,
  eegName,
  contractText: contractTextProp,
  documents = [],
  verifiedEmail,
  verifiedName1,
  verifiedName2,
}: Props) {
  // If verifiedEmail is set, skip straight to step 2
  const initialStep = verifiedEmail ? 2 : 1;

  const [step, setStep] = useState(initialStep);
  // "form" | "awaiting_verification"
  const [phase, setPhase] = useState<"form" | "awaiting_verification">("form");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submittedEmail, setSubmittedEmail] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [contractChecked, setContractChecked] = useState(false);
  const [resendLoading, setResendLoading] = useState(false);
  const [resendSuccess, setResendSuccess] = useState(false);

  const tomorrowISO = (() => {
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  })();
  const maxBeitrittISO = (() => {
    const d = new Date();
    d.setDate(d.getDate() + 31);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  })();

  const [formData, setFormData] = useState<FormData>({
    vorname: "",
    nachname: "",
    name1: verifiedName1 || "",
    name2: verifiedName2 || "",
    email: verifiedEmail || "",
    phone: "",
    strasse: "",
    plz: "",
    ort: "",
    iban: "",
    bic: "",
    memberType: "CONSUMER",
    businessRole: "privat",
    useVat: false,
    uidNummer: "",
    beitrittsDatum: tomorrowISO,
    meterPoints: [{ zaehlpunkt: "", direction: "CONSUMPTION", generationType: "", participationFactor: 100 }],
  });

  function updateField(field: keyof FormData, value: string | boolean) {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (typeof value === "string" && fieldErrors[field as string]) {
      setFieldErrors((prev) => {
        const copy = { ...prev };
        delete copy[field as string];
        return copy;
      });
    }
  }

  function updateMeterPoint(
    index: number,
    field: keyof MeterPointEntry,
    value: string | number
  ) {
    setFormData((prev) => {
      const mps = [...prev.meterPoints];
      mps[index] = { ...mps[index], [field]: value };
      return { ...prev, meterPoints: mps };
    });
  }

  function addMeterPoint() {
    if (formData.meterPoints.length >= 5) return;
    const defaultDirection =
      formData.memberType === "PRODUCER" ? "GENERATION" : "CONSUMPTION";
    setFormData((prev) => ({
      ...prev,
      meterPoints: [
        ...prev.meterPoints,
        { zaehlpunkt: "", direction: defaultDirection, generationType: "", participationFactor: 100 },
      ],
    }));
  }

  function removeMeterPoint(index: number) {
    setFormData((prev) => ({
      ...prev,
      meterPoints: prev.meterPoints.filter((_, i) => i !== index),
    }));
  }

  // Returns the effective name1 for submission/email-verification
  function computeName1(): string {
    if (formData.businessRole === "privat") {
      const combined = `${formData.vorname.trim()} ${formData.nachname.trim()}`.trim();
      // If coming back from email verification, vorname is blank — fall back to stored name1
      return combined || formData.name1.trim();
    }
    return formData.name1.trim();
  }

  function validateStep1(): boolean {
    const errors: Record<string, string> = {};
    if (formData.businessRole === "privat") {
      if (!formData.vorname.trim()) errors.vorname = "Vorname ist erforderlich.";
      if (!formData.nachname.trim()) errors.nachname = "Nachname ist erforderlich.";
    } else {
      if (!formData.name1.trim()) errors.name1 = "Firmenname ist erforderlich.";
    }
    if (!formData.email.trim()) errors.email = "E-Mail ist erforderlich.";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email))
      errors.email = "Ungültige E-Mail-Adresse.";
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  function validateStep2(): boolean {
    const errors: Record<string, string> = {};
    if (!formData.strasse.trim()) errors.strasse = "Straße ist erforderlich.";
    if (!formData.plz.trim()) errors.plz = "PLZ ist erforderlich.";
    if (!formData.ort.trim()) errors.ort = "Ort ist erforderlich.";
    if (!formData.iban.trim()) {
      errors.iban = "IBAN ist erforderlich.";
    } else {
      const ibanErr = validateIBAN(formData.iban);
      if (ibanErr) errors.iban = ibanErr;
    }
    if (formData.bic.trim()) {
      const bicErr = validateBIC(formData.bic);
      if (bicErr) errors.bic = bicErr;
    }
    if (formData.useVat && !formData.uidNummer.trim()) {
      errors.uidNummer = "UID-Nummer ist erforderlich wenn USt-pflichtig.";
    } else if (formData.uidNummer.trim()) {
      const uidErr = validateUIDNummer(formData.uidNummer);
      if (uidErr) errors.uidNummer = uidErr;
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  function validateStep3(): boolean {
    const errors: Record<string, string> = {};
    const hasAny = formData.meterPoints.some((mp) => mp.zaehlpunkt.trim() !== "");
    if (!hasAny) {
      errors["zaehlpunkt_0"] = "Mindestens ein Zählpunkt ist erforderlich.";
    }
    formData.meterPoints.forEach((mp, i) => {
      if (mp.zaehlpunkt.trim()) {
        const err = validateZaehlpunkt(mp.zaehlpunkt);
        if (err) errors[`zaehlpunkt_${i}`] = err;
      }
    });
    const filledPoints = formData.meterPoints.filter((mp) => mp.zaehlpunkt.trim() !== "");
    const consumptionCount = filledPoints.filter((mp) => mp.direction === "CONSUMPTION").length;
    const generationCount = filledPoints.filter((mp) => mp.direction === "GENERATION").length;
    if (formData.memberType === "CONSUMER") {
      if (consumptionCount < 1)
        errors["meterPoints_type"] = "Verbraucher benötigt mindestens einen Bezugszählpunkt.";
      else if (generationCount > 0)
        errors["meterPoints_type"] = "Verbraucher darf keine Einspeisezählpunkte haben.";
    } else if (formData.memberType === "PRODUCER") {
      if (generationCount < 1)
        errors["meterPoints_type"] = "Erzeuger benötigt mindestens einen Einspeisezählpunkt.";
      else if (consumptionCount > 0)
        errors["meterPoints_type"] = "Erzeuger darf keine Bezugszählpunkte haben.";
    } else if (formData.memberType === "PROSUMER") {
      if (consumptionCount < 1 && generationCount < 1)
        errors["meterPoints_type"] = "Prosumer benötigt mindestens einen Bezugs- und einen Einspeisezählpunkt.";
      else if (consumptionCount < 1)
        errors["meterPoints_type"] = "Prosumer benötigt mindestens einen Bezugszählpunkt.";
      else if (generationCount < 1)
        errors["meterPoints_type"] = "Prosumer benötigt mindestens einen Einspeisezählpunkt.";
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  function validateStep4(): boolean {
    if (!contractChecked) {
      setError(
        "Bitte bestätigen Sie die Beitrittserklärung, um fortzufahren."
      );
      return false;
    }
    return true;
  }

  async function handleNext() {
    setError(null);

    if (step === 1) {
      if (!validateStep1()) return;

      // If email is already verified (came from ?ev= param), skip verification
      if (verifiedEmail && verifiedEmail === formData.email) {
        setStep(2);
        return;
      }

      // Otherwise send verification email
      setLoading(true);
      try {
        const res = await fetch(
          `/api/public/eegs/${eegId}/onboarding/verify-email`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              email: formData.email.trim(),
              name1: computeName1(),
              name2: formData.name2.trim(),
            }),
          }
        );
        if (!res.ok) {
          const d = await res.json().catch(() => ({}));
          setError(
            (d as { error?: string }).error ||
              "Die Bestätigungs-E-Mail konnte nicht gesendet werden. Bitte versuchen Sie es erneut."
          );
          return;
        }
        setSubmittedEmail(formData.email.trim());
        setPhase("awaiting_verification");
      } catch {
        setError("Netzwerkfehler. Bitte prüfen Sie Ihre Verbindung.");
      } finally {
        setLoading(false);
      }
      return;
    }

    if (step === 2 && !validateStep2()) return;
    if (step === 3 && !validateStep3()) return;
    if (step < 4) setStep((s) => s + 1);
  }

  function handleBack() {
    setError(null);
    if (step > 1) setStep((s) => s - 1);
  }

  async function handleResend() {
    setResendLoading(true);
    setResendSuccess(false);
    try {
      const res = await fetch(
        `/api/public/eegs/${eegId}/onboarding/verify-email`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            email: submittedEmail,
            name1: computeName1(),
            name2: formData.name2.trim(),
          }),
        }
      );
      if (res.ok) {
        setResendSuccess(true);
        setTimeout(() => setResendSuccess(false), 4000);
      }
    } finally {
      setResendLoading(false);
    }
  }

  async function handleSubmit() {
    setError(null);
    if (!validateStep4()) return;

    setLoading(true);
    try {
      const meterPointsToSend = formData.meterPoints
        .filter((mp) => mp.zaehlpunkt.trim() !== "")
        .map((mp) => ({
          zaehlpunkt: mp.zaehlpunkt.trim(),
          direction: mp.direction,
          ...(mp.direction === "GENERATION" && mp.generationType ? { generation_type: mp.generationType } : {}),
          participation_factor: mp.participationFactor > 0 && mp.participationFactor <= 100 ? mp.participationFactor : 100,
        }));

      const res = await fetch(`/api/public/eegs/${eegId}/onboarding`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name1: computeName1(),
          name2: formData.name2.trim(),
          email: formData.email.trim(),
          phone: formData.phone.trim(),
          strasse: formData.strasse.trim(),
          plz: formData.plz.trim(),
          ort: formData.ort.trim(),
          iban: formData.iban.replace(/\s/g, ""),
          bic: formData.bic.trim(),
          member_type: formData.memberType,
          business_role: formData.businessRole,
          use_vat: formData.useVat,
          uid_nummer: formData.uidNummer.trim(),
          beitritts_datum: formData.beitrittsDatum || undefined,
          meter_points: meterPointsToSend,
          contract_accepted: true,
        }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(
          (data as { error?: string }).error ||
            "Der Antrag konnte nicht eingereicht werden. Bitte versuchen Sie es erneut."
        );
        return;
      }

      setSubmittedEmail(formData.email);
      setStep(5);
    } catch {
      setError(
        "Netzwerkfehler. Bitte prüfen Sie Ihre Verbindung und versuchen Sie es erneut."
      );
    } finally {
      setLoading(false);
    }
  }

  const today = new Date().toLocaleDateString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });

  const ibanDisplay = formData.iban || "(nicht angegeben)";
  const defaultContractText = `BEITRITTSERKLÄRUNG ZUR ENERGIEGEMEINSCHAFT

Hiermit erkläre ich/wir den Beitritt zur Energiegemeinschaft ${eegName} gemäß den Bestimmungen des Erneuerbaren-Ausbau-Gesetzes (EAG).

Ich/Wir bestätige/n:
• Die angegebenen Stammdaten (Name, Adresse, Kontaktdaten) sind korrekt.
• Die angegebenen Zählpunktnummern gehören zu meinen/unseren Anschlussstellen.
• Ich/Wir bin/sind berechtigt, über die genannten Zählpunkte zu verfügen.
• Ich/Wir werde/n die für die Anmeldung bei der Energiegemeinschaft notwendigen Schritte beim zuständigen Netzbetreiber einleiten.

Kontoverbindung für die Abrechnung:
IBAN: ${ibanDisplay}

Mit der Unterzeichnung dieser Beitrittserklärung erkenne ich/wir die Satzung und die Allgemeinen Geschäftsbedingungen der Energiegemeinschaft an.

Datum der elektronischen Unterzeichnung: ${today}`;

  const contractText = contractTextProp
    ? contractTextProp
        .replace(/\\u([0-9a-fA-F]{4})/g, (_, code) => String.fromCharCode(parseInt(code, 16)))
        .replace(/\{iban\}/g, ibanDisplay)
        .replace(/\{datum\}/g, today)
    : defaultContractText;

  // ── Awaiting email verification ──────────────────────────────────────────
  if (phase === "awaiting_verification") {
    return (
      <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8 text-center">
        <div className="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <svg
            className="w-8 h-8 text-blue-600"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
            />
          </svg>
        </div>
        <h2 className="text-xl font-bold text-slate-900 mb-2">
          Bestätigungslink gesendet
        </h2>
        <p className="text-slate-600 mb-2">
          Wir haben einen Bestätigungslink an{" "}
          <span className="font-medium text-slate-900">{submittedEmail}</span>{" "}
          gesendet.
        </p>
        <p className="text-sm text-slate-500 mb-8">
          Bitte öffnen Sie den Link in der E-Mail, um fortzufahren. Der Link ist 30 Minuten gültig.
        </p>

        <div className="flex flex-col items-center gap-3">
          {resendSuccess && (
            <p className="text-sm text-green-600">
              Neuer Link wurde gesendet.
            </p>
          )}
          <button
            type="button"
            onClick={handleResend}
            disabled={resendLoading}
            className="px-4 py-2 border border-slate-300 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-50 transition-colors disabled:opacity-50"
          >
            {resendLoading ? "Wird gesendet…" : "Neuen Link anfordern"}
          </button>
          <button
            type="button"
            onClick={() => {
              setPhase("form");
              setError(null);
            }}
            className="text-sm text-slate-400 hover:text-slate-600 transition-colors"
          >
            Zurück zu Schritt 1
          </button>
        </div>
      </div>
    );
  }

  // ── Success screen ───────────────────────────────────────────────────────
  if (step === 5) {
    return (
      <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8 text-center">
        <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <svg
            className="w-8 h-8 text-green-600"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M5 13l4 4L19 7"
            />
          </svg>
        </div>
        <h2 className="text-2xl font-bold text-slate-900 mb-2">
          Ihr Antrag wurde eingereicht!
        </h2>
        <p className="text-slate-600 mb-6">
          Wir haben Ihnen einen Link an{" "}
          <span className="font-medium text-slate-900">{submittedEmail}</span>{" "}
          gesendet, über den Sie den Status Ihres Antrags verfolgen können.
        </p>

        <div className="bg-blue-50 rounded-xl p-6 text-left">
          <h3 className="font-semibold text-slate-900 mb-4">
            Was passiert als Nächstes?
          </h3>
          <ol className="space-y-3">
            {[
              "Wir prüfen Ihren Antrag und melden uns bei Ihnen",
              "Nach Genehmigung legen wir Ihr Mitgliedskonto an und stellen die EDA-Anmeldung beim Netzbetreiber",
              "Sie erhalten eine E-Mail mit dem Link zum Portal Ihres Netzbetreibers",
              "Dort aktivieren Sie den 15-Minuten-Takt und erteilen die Datenfreigabe für die Energiegemeinschaft",
              "Sobald der Netzbetreiber verarbeitet hat, beginnt die Energieverrechnung automatisch",
            ].map((step, i) => (
              <li key={i} className="flex gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-xs font-bold">
                  {i + 1}
                </span>
                <span className="text-slate-700 text-sm">{step}</span>
              </li>
            ))}
          </ol>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8">
      <StepIndicator currentStep={step} totalSteps={4} />

      {/* Step 1: Persönliche Daten / Firmendaten */}
      {step === 1 && (
        <div>
          <h2 className="text-lg font-semibold text-slate-900 mb-6">
            {formData.businessRole === "privat" ? "Persönliche Daten" : formData.businessRole === "verein" ? "Vereinsdaten" : "Firmendaten"}
          </h2>
          <div className="space-y-4">
            {/* Unternehmensart — drives the name fields below */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Ich bin…
              </label>
              <select
                value={formData.businessRole}
                onChange={(e) => {
                  updateField("businessRole", e.target.value);
                  // Clear name fields when switching type to avoid stale values
                  updateField("vorname", "");
                  updateField("nachname", "");
                  updateField("name1", "");
                  updateField("name2", "");
                }}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
              >
                <option value="privat">Privatperson</option>
                <option value="kleinunternehmer">Kleinunternehmer/in</option>
                <option value="verein">Verein</option>
                <option value="landwirt_pauschaliert">Landwirt/in (pauschaliert, § 22 UStG)</option>
                <option value="landwirt">Landwirt/in (buchführungspflichtig)</option>
                <option value="unternehmen">Unternehmen / GmbH / AG …</option>
                <option value="gemeinde_bga">Gemeinde (BgA)</option>
                <option value="gemeinde_hoheitlich">Gemeinde (hoheitlich)</option>
              </select>
            </div>

            {/* Name fields — conditional on businessRole */}
            {formData.businessRole === "privat" ? (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Vorname <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.vorname}
                    onChange={(e) => updateField("vorname", e.target.value)}
                    placeholder="Max"
                    autoComplete="given-name"
                    className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.vorname ? "border-red-400" : "border-slate-300"}`}
                  />
                  <FieldError msg={fieldErrors.vorname} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Nachname <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.nachname}
                    onChange={(e) => updateField("nachname", e.target.value)}
                    placeholder="Mustermann"
                    autoComplete="family-name"
                    className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.nachname ? "border-red-400" : "border-slate-300"}`}
                  />
                  <FieldError msg={fieldErrors.nachname} />
                </div>
              </div>
            ) : (
              <>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    {formData.businessRole === "verein" ? "Vereinsname" : "Firmenname"} <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.name1}
                    onChange={(e) => updateField("name1", e.target.value)}
                    placeholder={formData.businessRole === "verein" ? "Muster Verein" : "Muster GmbH"}
                    autoComplete="organization"
                    className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.name1 ? "border-red-400" : "border-slate-300"}`}
                  />
                  <FieldError msg={fieldErrors.name1} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Zusatz (optional)
                  </label>
                  <input
                    type="text"
                    value={formData.name2}
                    onChange={(e) => updateField("name2", e.target.value)}
                    placeholder="z.B. Abteilung oder c/o"
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </>
            )}

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                E-Mail-Adresse <span className="text-red-500">*</span>
              </label>
              <input
                type="email"
                value={formData.email}
                onChange={(e) => updateField("email", e.target.value)}
                placeholder="max@beispiel.at"
                autoComplete="email"
                className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.email ? "border-red-400" : "border-slate-300"}`}
              />
              <FieldError msg={fieldErrors.email} />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Telefonnummer (optional)
              </label>
              <input
                type="tel"
                value={formData.phone}
                onChange={(e) => updateField("phone", e.target.value)}
                placeholder="+43 123 456 789"
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>
        </div>
      )}

      {/* Step 2: Adresse & Bankdaten */}
      {step === 2 && (
        <div>
          <h2 className="text-lg font-semibold text-slate-900 mb-6">
            Adresse & Bankdaten
          </h2>
          {verifiedEmail && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700 flex items-center gap-2">
              <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              E-Mail-Adresse <strong>{verifiedEmail}</strong> bestätigt.
            </div>
          )}
          {/* Show businessRole here only when Step 1 was skipped (email-verified path) */}
          {verifiedEmail && (
            <div className="mb-4">
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Ich bin…
              </label>
              <select
                value={formData.businessRole}
                onChange={(e) => updateField("businessRole", e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
              >
                <option value="privat">Privatperson</option>
                <option value="kleinunternehmer">Kleinunternehmer/in</option>
                <option value="verein">Verein</option>
                <option value="landwirt_pauschaliert">Landwirt/in (pauschaliert, § 22 UStG)</option>
                <option value="landwirt">Landwirt/in (buchführungspflichtig)</option>
                <option value="unternehmen">Unternehmen / GmbH / AG …</option>
                <option value="gemeinde_bga">Gemeinde (BgA)</option>
                <option value="gemeinde_hoheitlich">Gemeinde (hoheitlich)</option>
              </select>
            </div>
          )}
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Straße und Hausnummer <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={formData.strasse}
                onChange={(e) => updateField("strasse", e.target.value)}
                placeholder="Musterstraße 1"
                className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.strasse ? "border-red-400" : "border-slate-300"}`}
              />
              <FieldError msg={fieldErrors.strasse} />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  PLZ <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.plz}
                  onChange={(e) => updateField("plz", e.target.value)}
                  placeholder="1234"
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.plz ? "border-red-400" : "border-slate-300"}`}
                />
                <FieldError msg={fieldErrors.plz} />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Ort <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.ort}
                  onChange={(e) => updateField("ort", e.target.value)}
                  placeholder="Musterort"
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${fieldErrors.ort ? "border-red-400" : "border-slate-300"}`}
                />
                <FieldError msg={fieldErrors.ort} />
              </div>
            </div>

            <hr className="border-slate-200 my-2" />

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                IBAN (für Abrechnung) <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={formData.iban}
                onChange={(e) => {
                  const stripped = e.target.value.replace(/[^A-Za-z0-9]/g, "").toUpperCase();
                  const formatted = stripped.match(/.{1,4}/g)?.join(" ") ?? stripped;
                  updateField("iban", formatted);
                }}
                onBlur={(e) => {
                  const err = validateIBAN(e.target.value.replace(/\s/g, ""));
                  setFieldErrors((prev) => err ? { ...prev, iban: err } : (({ iban: _, ...rest }) => rest)(prev));
                }}
                placeholder="AT12 3456 7890 1234 5678"
                className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 font-mono ${fieldErrors.iban ? "border-red-400 focus:ring-red-400" : "border-slate-300 focus:ring-blue-500"}`}
              />
              {fieldErrors.iban && <p className="text-xs text-red-600 mt-1">{fieldErrors.iban}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                BIC (optional)
              </label>
              <input
                type="text"
                value={formData.bic}
                onChange={(e) => updateField("bic", e.target.value.toUpperCase())}
                onBlur={(e) => {
                  const err = validateBIC(e.target.value);
                  setFieldErrors((prev) => err ? { ...prev, bic: err } : (({ bic: _, ...rest }) => rest)(prev));
                }}
                placeholder="RLNWATWWXXX"
                className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 font-mono ${fieldErrors.bic ? "border-red-400 focus:ring-red-400" : "border-slate-300 focus:ring-blue-500"}`}
              />
              {fieldErrors.bic && <p className="text-xs text-red-600 mt-1">{fieldErrors.bic}</p>}
            </div>

            <hr className="border-slate-200 my-2" />

            <div>
              <label className="flex items-start gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.useVat}
                  onChange={(e) => {
                    updateField("useVat", e.target.checked);
                    if (!e.target.checked) updateField("uidNummer", "");
                  }}
                  className="mt-0.5 h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-slate-700">
                  Ich bin umsatzsteuerpflichtig (USt-pflichtig)
                </span>
              </label>
              <p className="mt-1 text-xs text-slate-500 ml-7">
                Privatpersonen sind üblicherweise <strong>nicht</strong> umsatzsteuerpflichtig.
                Unternehmen mit einem Jahresumsatz über € 55.000 (netto) sind es in der Regel schon.
              </p>
            </div>

            {formData.useVat && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  UID-Nummer <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.uidNummer}
                  onChange={(e) => updateField("uidNummer", e.target.value.toUpperCase())}
                  placeholder="ATU12345678"
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono ${fieldErrors.uidNummer ? "border-red-400" : "border-slate-300"}`}
                />
                <FieldError msg={fieldErrors.uidNummer} />
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Mitgliedstyp
              </label>
              <select
                value={formData.memberType}
                onChange={(e) => updateField("memberType", e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
              >
                <option value="CONSUMER">Verbraucher (Bezug)</option>
                <option value="PRODUCER">Erzeuger (Einspeisung)</option>
                <option value="PROSUMER">Prosumer (Bezug & Einspeisung)</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Gewünschtes Beitrittsdatum (optional)
              </label>
              <input
                type="date"
                value={formData.beitrittsDatum}
                min={tomorrowISO}
                max={maxBeitrittISO}
                onChange={(e) => updateField("beitrittsDatum", e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-xs text-slate-500 mt-1">Frühestens morgen, höchstens 30 Tage in der Zukunft. Das genaue Datum wird vom Netzbetreiber bestätigt.</p>
            </div>
          </div>
        </div>
      )}

      {/* Step 3: Zählpunkte */}
      {step === 3 && (
        <div>
          <h2 className="text-lg font-semibold text-slate-900 mb-2">
            Zählpunkte
          </h2>
          <p className="text-sm text-slate-500 mb-3">
            Geben Sie Ihre Zählpunktnummern ein (beginnen mit AT0...). Mindestens ein Zählpunkt ist erforderlich. <span className="text-red-500">*</span>
          </p>
          {fieldErrors["meterPoints_type"] && (
            <div className="mb-3 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {fieldErrors["meterPoints_type"]}
            </div>
          )}
          <div className="mb-5 p-3 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-600">
            <span className="font-medium">Teilnahmefaktor:</span> Gibt an, welcher Anteil Ihrer Energie (in %) an der Energiegemeinschaft teilnimmt. Für die meisten Mitglieder ist der Standardwert <strong>100%</strong> korrekt. Ein anderer Wert ist nur relevant, wenn Sie gleichzeitig in mehreren Energiegemeinschaften Mitglied sind.
          </div>
          <div className="space-y-3">
            {formData.meterPoints.map((mp, index) => (
              <div
                key={index}
                className="flex gap-2 items-start p-3 bg-slate-50 rounded-lg border border-slate-200"
              >
                <div className="flex-1">
                  <label className="block text-xs font-medium text-slate-600 mb-1">
                    Zählpunktnummer {index + 1}
                  </label>
                  <input
                    type="text"
                    value={mp.zaehlpunkt}
                    onChange={(e) =>
                      updateMeterPoint(index, "zaehlpunkt", e.target.value.toUpperCase())
                    }
                    onBlur={(e) => {
                      const err = validateZaehlpunkt(e.target.value);
                      const key = `zaehlpunkt_${index}`;
                      setFieldErrors((prev) => err ? { ...prev, [key]: err } : Object.fromEntries(Object.entries(prev).filter(([k]) => k !== key)));
                    }}
                    placeholder="AT0000000000000000000000000000000"
                    className={`w-full px-3 py-2 border rounded-lg text-xs focus:outline-none focus:ring-2 font-mono bg-white ${fieldErrors[`zaehlpunkt_${index}`] ? "border-red-400 focus:ring-red-400" : "border-slate-300 focus:ring-blue-500"}`}
                  />
                  {fieldErrors[`zaehlpunkt_${index}`] && (
                    <p className="text-xs text-red-600 mt-1">{fieldErrors[`zaehlpunkt_${index}`]}</p>
                  )}
                </div>
                <div className="w-36">
                  <label className="block text-xs font-medium text-slate-600 mb-1">
                    Energierichtung
                  </label>
                  <select
                    value={mp.direction}
                    onChange={(e) =>
                      updateMeterPoint(index, "direction", e.target.value)
                    }
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                  >
                    {(formData.memberType === "PRODUCER"
                      ? ["GENERATION"]
                      : formData.memberType === "CONSUMER"
                        ? ["CONSUMPTION"]
                        : ["CONSUMPTION", "GENERATION"]
                    ).map((dir) => (
                      <option key={dir} value={dir}>
                        {dir === "CONSUMPTION" ? "Bezug" : "Einspeisung"}
                      </option>
                    ))}
                  </select>
                </div>
                {mp.direction === "GENERATION" && (
                  <div className="w-40">
                    <label className="block text-xs font-medium text-slate-600 mb-1">
                      Erzeugungsart
                    </label>
                    <select
                      value={mp.generationType}
                      onChange={(e) =>
                        updateMeterPoint(index, "generationType", e.target.value)
                      }
                      className="w-full px-3 py-2 border border-slate-300 rounded-lg text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                    >
                      <option value="">– bitte wählen –</option>
                      <option value="PV">Photovoltaik</option>
                      <option value="Windkraft">Windkraft</option>
                      <option value="Wasserkraft">Wasserkraft</option>
                      <option value="Biomasse">Biomasse</option>
                      <option value="Sonstige">Sonstige</option>
                    </select>
                  </div>
                )}
                <div className="w-32">
                  <label className="block text-xs font-medium text-slate-600 mb-1">
                    Teilnahmefaktor
                  </label>
                  <div className="flex items-center gap-1">
                    <input
                      type="number"
                      min="0.01"
                      max="100"
                      step="0.01"
                      value={mp.participationFactor}
                      onChange={(e) =>
                        updateMeterPoint(index, "participationFactor", parseFloat(e.target.value) || 100)
                      }
                      className="w-full px-2 py-2 border border-slate-300 rounded-lg text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                    />
                    <span className="text-xs text-slate-500">%</span>
                  </div>
                </div>
                {formData.meterPoints.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeMeterPoint(index)}
                    className="mt-5 p-1.5 text-slate-400 hover:text-red-500 transition-colors"
                    title="Zählpunkt entfernen"
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
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                )}
              </div>
            ))}
          </div>
          {formData.meterPoints.length < 5 && (
            <button
              type="button"
              onClick={addMeterPoint}
              className="mt-3 flex items-center gap-2 text-sm text-blue-600 hover:text-blue-800 font-medium"
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
                  d="M12 4v16m8-8H4"
                />
              </svg>
              Zählpunkt hinzufügen
            </button>
          )}
        </div>
      )}

      {/* Step 4: Vertrag & Unterzeichnung */}
      {step === 4 && (
        <div>
          <h2 className="text-lg font-semibold text-slate-900 mb-4">
            Vertrag & Unterzeichnung
          </h2>
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-4 mb-6 max-h-72 overflow-y-auto">
            <pre className="text-xs text-slate-700 whitespace-pre-wrap font-sans leading-relaxed">
              {contractText}
            </pre>
          </div>

          {documents.length > 0 && (
            <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
              <p className="text-sm font-medium text-blue-900 mb-2">Dokumente</p>
              <ul className="space-y-1">
                {documents.map((doc) => (
                  <li key={doc.id}>
                    <a
                      href={`/api/public/eegs/${eegId}/documents/${doc.id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 text-sm text-blue-700 hover:text-blue-900 hover:underline"
                    >
                      <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                      {doc.title}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {error}
            </div>
          )}

          <label className="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={contractChecked}
              onChange={(e) => {
                setContractChecked(e.target.checked);
                if (e.target.checked) setError(null);
              }}
              className="mt-0.5 h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-slate-700">
              Ich bestätige die Richtigkeit meiner Angaben und stimme dem
              Beitritt zur Energiegemeinschaft{" "}
              <strong>{eegName}</strong> zu. Ich habe die
              Beitrittserklärung gelesen und akzeptiere diese.{" "}
              <span className="text-red-500">*</span>
            </span>
          </label>
        </div>
      )}

      {/* Error (outside step 4) */}
      {error && step !== 4 && (
        <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Navigation */}
      <div className="flex gap-3 mt-8">
        {step > 1 && (
          <button
            type="button"
            onClick={handleBack}
            disabled={loading}
            className="flex-1 px-4 py-2.5 border border-slate-300 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-50 transition-colors disabled:opacity-50"
          >
            Zurück
          </button>
        )}
        {step < 4 ? (
          <button
            type="button"
            onClick={handleNext}
            disabled={loading}
            className="flex-1 px-4 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {loading && (
              <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
            )}
            {loading ? "Bitte warten…" : "Weiter"}
          </button>
        ) : (
          <button
            type="button"
            onClick={handleSubmit}
            disabled={loading || !contractChecked}
            className="flex-1 px-4 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "Wird eingereicht…" : "Verbindlich beitreten"}
          </button>
        )}
      </div>
    </div>
  );
}
