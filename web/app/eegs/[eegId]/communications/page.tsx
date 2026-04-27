"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG } from "@/lib/api";
import Link from "next/link";
import EmailComposer from "./EmailComposer";
import CampaignHistory from "./CampaignHistory";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

interface Campaign {
  id: string;
  subject: string;
  created_at: string;
  recipient_count: number;
  attachment_count: number;
}

interface Props {
  params: Promise<{ eegId: string }>;
}

export default async function CommunicationsPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;

  let eeg = null;
  try {
    eeg = await getEEG(session.accessToken!, eegId);
  } catch {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler beim Laden der EEG.</p>
        </div>
      </div>
    );
  }

  let campaigns: Campaign[] = [];
  try {
    const res = await fetch(`${API}/api/v1/eegs/${eegId}/communications`, {
      headers: { Authorization: `Bearer ${session.accessToken!}` },
      cache: "no-store",
    });
    if (res.ok) {
      campaigns = await res.json();
    }
  } catch {
    // show empty list on error
  }

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link
          href={`/eegs/${eegId}`}
          className="text-sm text-slate-500 hover:text-slate-700"
        >
          {eeg.name}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Kommunikation</span>
      </div>

      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Mitglieder-E-Mail</h1>
        <p className="text-slate-500 mt-1">
          E-Mails an alle aktiven Mitglieder von {eeg.name} senden.
        </p>
      </div>

      {/* Composer (client component) */}
      <div className="max-w-3xl mb-8">
        <EmailComposer eegId={eegId} />
      </div>

      {/* Campaign history */}
      <div className="max-w-3xl">
        <h2 className="text-base font-semibold text-slate-900 mb-4">Verlauf</h2>
        <CampaignHistory campaigns={campaigns} eegId={eegId} />
      </div>
    </div>
  );
}
