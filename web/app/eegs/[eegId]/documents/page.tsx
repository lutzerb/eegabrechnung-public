"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG } from "@/lib/api";
import Link from "next/link";
import DocumentManager, { Document } from "./DocumentManager";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

interface Props {
  params: Promise<{ eegId: string }>;
}

export default async function DocumentsPage({ params }: Props) {
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

  let documents: Document[] = [];
  try {
    const res = await fetch(`${API}/api/v1/eegs/${eegId}/documents`, {
      headers: { Authorization: `Bearer ${session.accessToken!}` },
      cache: "no-store",
    });
    if (res.ok) {
      documents = await res.json();
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
        <span className="text-sm text-slate-900 font-medium">Dokumente</span>
      </div>

      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Dokumente für Mitglieder</h1>
        <p className="text-slate-500 mt-1">
          Diese Dokumente sind im Mitglieder-Portal unter Downloads verfügbar.
        </p>
      </div>

      <div className="max-w-2xl">
        <DocumentManager eegId={eegId} initialDocuments={documents} />
      </div>
    </div>
  );
}
