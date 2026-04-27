import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getMember, listInvoices } from "@/lib/api";
import { formatIBAN } from "@/lib/validation";
import Link from "next/link";
import AustrittDialog from "./AustrittDialog";
import DeleteMeterPointButton from "./DeleteMeterPointButton";

interface Props {
  params: Promise<{ eegId: string; memberId: string }>;
}

const BUSINESS_ROLE_LABELS: Record<string, string> = {
  privat:               "Privatperson",
  kleinunternehmer:     "Kleinunternehmer",
  verein:               "Verein",
  landwirt_pauschaliert:"Landwirt (pauschaliert, § 22 UStG)",
  landwirt:             "Landwirt (buchführungspflichtig)",
  unternehmen:          "Unternehmen",
  gemeinde_bga:         "Gemeinde (BgA)",
  gemeinde_hoheitlich:  "Gemeinde (hoheitlich)",
};

const STATUS_LABELS: Record<string, { label: string; color: string }> = {
  ACTIVE:     { label: "Aktiv",        color: "bg-green-100 text-green-700" },
  REGISTERED: { label: "Angemeldet",   color: "bg-blue-100 text-blue-700" },
  NEW:        { label: "Neu",          color: "bg-yellow-100 text-yellow-700" },
  INACTIVE:   { label: "Inaktiv",      color: "bg-slate-100 text-slate-500" },
};

const INVOICE_STATUS: Record<string, { label: string; cls: string }> = {
  draft:     { label: "Entwurf",    cls: "bg-slate-100 text-slate-600" },
  finalized: { label: "Finalisiert", cls: "bg-green-50 text-green-700" },
  pending:   { label: "Ausstehend", cls: "bg-yellow-50 text-yellow-700" },
  sent:      { label: "Versendet",  cls: "bg-blue-50 text-blue-700" },
  paid:      { label: "Bezahlt",    cls: "bg-green-50 text-green-700" },
  cancelled: { label: "Storniert",  cls: "bg-red-50 text-red-700" },
};

function formatDate(dateStr: string | undefined | null): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(amount);
}

export default async function MemberDetailPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId, memberId } = await params;

  let eeg = null;
  let member = null;
  let loadError: string | null = null;

  try {
    [eeg, member] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      getMember(session.accessToken!, eegId, memberId),
    ]);
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    loadError = apiError.message || "Fehler beim Laden des Mitglieds.";
  }

  // Fetch invoices filtered by member
  let invoices: Awaited<ReturnType<typeof listInvoices>> = [];
  try {
    const allInvoices = await listInvoices(session.accessToken!, eegId);
    invoices = allInvoices.filter((inv) => inv.member_id === memberId);
  } catch {
    // non-fatal
  }


  if (loadError || !member) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{loadError}</p>
        </div>
      </div>
    );
  }

  const st = member.status || "ACTIVE";
  const statusMeta = STATUS_LABELS[st] || { label: st, color: "bg-slate-100 text-slate-600" };
  const memberName = member.name || [member.name1, member.name2].filter(Boolean).join(" ");

  return (
    <div className="p-8 space-y-6">
      {/* Breadcrumb */}
      <div>
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">Energiegemeinschaften</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">{eeg?.name || eegId}</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/members`} className="text-sm text-slate-500 hover:text-slate-700">Mitglieder</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">{memberName}</span>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-2xl font-bold text-slate-900">{memberName}</h1>
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusMeta.color}`}>
              {statusMeta.label}
            </span>
          </div>
          {member.mitglieds_nr && (
            <p className="text-sm text-slate-500 mt-1 font-mono">Nr. {member.mitglieds_nr}</p>
          )}
        </div>
        <div className="flex gap-2 flex-wrap">
          <Link
            href={`/eegs/${eegId}/members`}
            className="px-4 py-2 text-sm font-medium border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors"
          >
            ← Zurück
          </Link>
          <Link
            href={`/eegs/${eegId}/members/${memberId}/edit`}
            className="px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
          >
            Bearbeiten
          </Link>
          <a
            href={`/api/eegs/${eegId}/members/${memberId}/sepa-mandat`}
            target="_blank"
            rel="noopener noreferrer"
            className="px-4 py-2 text-sm font-medium bg-slate-100 text-slate-700 rounded-lg hover:bg-slate-200 transition-colors"
          >
            SEPA-Mandat
          </a>
          {(member.status === "ACTIVE" || member.status === "REGISTERED") && (
            <AustrittDialog
              eegId={eegId}
              memberId={memberId}
              memberName={memberName}
            />
          )}
        </div>
      </div>

      {/* Member details card */}
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <h2 className="text-base font-semibold text-slate-900 mb-4">Stammdaten</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-8 gap-y-4">
          <div>
            <p className="text-xs font-medium text-slate-500 mb-0.5">E-Mail</p>
            <p className="text-sm text-slate-900">{member.email || "—"}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-slate-500 mb-0.5">IBAN</p>
            <p className="text-sm font-mono text-slate-900">{member.iban ? formatIBAN(member.iban) : "—"}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-slate-500 mb-0.5">Unternehmensrolle</p>
            <p className="text-sm text-slate-900">{(member.business_role && BUSINESS_ROLE_LABELS[member.business_role]) || member.business_role || "—"}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-slate-500 mb-0.5">Adresse</p>
            <p className="text-sm text-slate-900">
              {[member.strasse, [member.plz, member.ort].filter(Boolean).join(" ")].filter(Boolean).join(", ") || "—"}
            </p>
          </div>
          <div>
            <p className="text-xs font-medium text-slate-500 mb-0.5">Beitrittsdatum</p>
            <p className="text-sm text-slate-900">{formatDate(member.beitritts_datum)}</p>
          </div>
          {member.austritts_datum && (
            <div>
              <p className="text-xs font-medium text-slate-500 mb-0.5">Austrittsdatum</p>
              <p className="text-sm text-slate-900">{formatDate(member.austritts_datum)}</p>
            </div>
          )}
          {member.uid_nummer && (
            <div>
              <p className="text-xs font-medium text-slate-500 mb-0.5">UID-Nummer</p>
              <p className="text-sm font-mono text-slate-900">{member.uid_nummer}</p>
            </div>
          )}
        </div>
      </div>

      {/* Meter points */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
          <h2 className="text-base font-semibold text-slate-900">Zählpunkte</h2>
          <Link
            href={`/eegs/${eegId}/members/${memberId}/meter-points/new`}
            className="px-3 py-1.5 text-xs font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 transition-colors"
          >
            + Zählpunkt
          </Link>
        </div>
        {!member.meter_points || member.meter_points.length === 0 ? (
          <div className="px-6 py-8 text-center text-sm text-slate-400">Keine Zählpunkte vorhanden.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50">
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Zählpunkt-ID</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Richtung</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Typ</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Teilnahmefaktor</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">EDA-Status</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Aktiv seit</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Abgemeldet am</th>
                  <th className="text-right px-6 py-3 font-medium text-slate-600">Aktionen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {member.meter_points.map((mp) => (
                  <tr key={mp.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-6 py-3 font-mono text-xs text-slate-700">
                      <span className="flex items-center gap-1.5">
                        {mp.meter_id}
                        {mp.gap_alert_sent_at && (
                          <span
                            title={`Datenlücke: kein Reading seit ${new Date(mp.gap_alert_sent_at).toLocaleDateString("de-AT")}`}
                            className="inline-flex items-center justify-center w-4 h-4 rounded-full bg-orange-100 text-orange-600 flex-shrink-0"
                            aria-label="Datenlücke"
                          >
                            <svg viewBox="0 0 16 16" fill="currentColor" className="w-3 h-3">
                              <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zm0 3a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 8 4zm0 7.5a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5z"/>
                            </svg>
                          </span>
                        )}
                      </span>
                    </td>
                    <td className="px-6 py-3">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                        mp.direction === "CONSUMPTION" ? "bg-blue-50 text-blue-700" : "bg-green-50 text-green-700"
                      }`}>
                        {mp.direction === "CONSUMPTION" ? "Bezug" : mp.direction === "GENERATION" ? "Einspeisung" : mp.direction}
                      </span>
                    </td>
                    <td className="px-6 py-3 text-slate-600 text-xs">
                      {mp.generation_type || "—"}
                    </td>
                    <td className="px-6 py-3 text-xs text-slate-700">
                      {mp.participation_factor != null
                        ? `${mp.participation_factor}%`
                        : <span className="text-slate-300">—</span>}
                    </td>
                    <td className="px-6 py-3 text-xs">
                      <EdaStatusBadge anmeldung={mp.anmeldung_status} abmeldung={mp.abmeldung_status} />
                    </td>
                    <td className="px-6 py-3 text-slate-600 text-xs">
                      {mp.registriert_seit
                        ? new Date(mp.registriert_seit).toLocaleDateString("de-AT")
                        : <span className="text-slate-300">—</span>}
                    </td>
                    <td className="px-6 py-3 text-xs">
                      {mp.abgemeldet_am
                        ? <span className="text-orange-600">{new Date(mp.abgemeldet_am).toLocaleDateString("de-AT")}</span>
                        : <span className="text-slate-300">—</span>}
                    </td>
                    <td className="px-6 py-3 text-right flex items-center justify-end gap-1">
                      <Link
                        href={`/eegs/${eegId}/members/${memberId}/meter-points/${mp.id}/edit`}
                        className="px-2 py-1 text-xs text-blue-700 hover:bg-blue-50 rounded transition-colors"
                      >
                        Bearbeiten
                      </Link>
                      <DeleteMeterPointButton
                        eegId={eegId}
                        meterPointId={mp.id}
                        meterId={mp.meter_id}
                        anmeldungStatus={mp.anmeldung_status}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Invoices */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200">
          <h2 className="text-base font-semibold text-slate-900">Rechnungen</h2>
          <p className="text-xs text-slate-500 mt-0.5">{invoices.length} Rechnungen gesamt</p>
        </div>
        {invoices.length === 0 ? (
          <div className="px-6 py-8 text-center text-sm text-slate-400">Keine Rechnungen vorhanden.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50">
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Nr.</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Zeitraum</th>
                  <th className="text-right px-6 py-3 font-medium text-slate-600">Betrag</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Typ</th>
                  <th className="text-left px-6 py-3 font-medium text-slate-600">Status</th>
                  <th className="text-right px-6 py-3 font-medium text-slate-600">Aktionen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {invoices
                  .sort((a, b) => new Date(b.period_start).getTime() - new Date(a.period_start).getTime())
                  .map((invoice) => {
                    const invStatus = INVOICE_STATUS[invoice.status?.toLowerCase()] || { label: invoice.status, cls: "bg-slate-100 text-slate-600" };
                    return (
                      <tr key={invoice.id} className="hover:bg-slate-50 transition-colors">
                        <td className="px-6 py-3 text-slate-400 font-mono text-xs">
                          {invoice.invoice_number ? `#${invoice.invoice_number}` : invoice.id.slice(0, 8)}
                        </td>
                        <td className="px-6 py-3 text-slate-600 text-xs whitespace-nowrap">
                          {formatDate(invoice.period_start)} – {formatDate(invoice.period_end)}
                        </td>
                        <td className="px-6 py-3 text-right font-medium text-slate-900">
                          {formatCurrency(invoice.total_amount)}
                        </td>
                        <td className="px-6 py-3">
                          {invoice.document_type === "credit_note" ? (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-50 text-purple-700">Gutschrift</span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">Rechnung</span>
                          )}
                        </td>
                        <td className="px-6 py-3">
                          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${invStatus.cls}`}>
                            {invStatus.label}
                          </span>
                        </td>
                        <td className="px-6 py-3 text-right">
                          {invoice.pdf_path && (
                            <a
                              href={`/api/eegs/${eegId}/invoices/${invoice.id}/pdf`}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="px-2 py-1 text-xs font-medium text-slate-700 bg-slate-100 rounded hover:bg-slate-200 transition-colors"
                            >
                              PDF
                            </a>
                          )}
                        </td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

// EDA-Status-Badge: zeigt Anmeldungs- und Abmeldungs-Status kompakt an.
function EdaStatusBadge({ anmeldung, abmeldung }: { anmeldung?: string; abmeldung?: string }) {
  if (!anmeldung && !abmeldung) return <span className="text-slate-300">—</span>;

  const anmeldungBadge = anmeldung ? (() => {
    const cfg: Record<string, { label: string; cls: string }> = {
      confirmed:       { label: "ANM bestätigt", cls: "bg-green-100 text-green-700" },
      first_confirmed: { label: "ANM teilbestätigt", cls: "bg-emerald-50 text-emerald-600" },
      sent:            { label: "ANM gesendet", cls: "bg-yellow-50 text-yellow-700" },
      pending:         { label: "ANM ausstehend", cls: "bg-slate-100 text-slate-500" },
      rejected:        { label: "ANM abgelehnt", cls: "bg-red-100 text-red-600" },
      error:           { label: "ANM Fehler", cls: "bg-red-100 text-red-600" },
    };
    const c = cfg[anmeldung] ?? { label: anmeldung, cls: "bg-slate-100 text-slate-500" };
    return <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${c.cls}`}>{c.label}</span>;
  })() : null;

  const abmeldungBadge = abmeldung ? (() => {
    const cfg: Record<string, { label: string; cls: string }> = {
      confirmed:       { label: "ABM bestätigt", cls: "bg-orange-100 text-orange-700" },
      first_confirmed: { label: "ABM teilbestätigt", cls: "bg-orange-50 text-orange-600" },
      sent:            { label: "ABM gesendet", cls: "bg-yellow-50 text-yellow-700" },
      pending:         { label: "ABM ausstehend", cls: "bg-slate-100 text-slate-500" },
      rejected:        { label: "ABM abgelehnt", cls: "bg-red-100 text-red-600" },
      error:           { label: "ABM Fehler", cls: "bg-red-100 text-red-600" },
    };
    const c = cfg[abmeldung] ?? { label: abmeldung, cls: "bg-slate-100 text-slate-500" };
    return <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${c.cls}`}>{c.label}</span>;
  })() : null;

  return (
    <div className="flex flex-col gap-0.5">
      {anmeldungBadge}
      {abmeldungBadge}
    </div>
  );
}
