"use client";

import { useState, useRef, useCallback } from "react";

export interface Document {
  id: string;
  title: string;
  description: string;
  filename: string;
  mime_type: string;
  file_size_bytes: number;
  show_in_onboarding: boolean;
  created_at: string;
}

interface Props {
  eegId: string;
  initialDocuments: Document[];
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

export default function DocumentManager({ eegId, initialDocuments }: Props) {
  const [documents, setDocuments] = useState<Document[]>(initialDocuments);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploadSuccess, setUploadSuccess] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [showInOnboarding, setShowInOnboarding] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null;
    setFile(f);
    if (f && !title) {
      setTitle(f.name.replace(/\.[^.]+$/, ""));
    }
  }, [title]);

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault();
    if (!file || !title.trim()) return;

    setUploading(true);
    setUploadError(null);
    setUploadSuccess(false);

    try {
      const fd = new FormData();
      fd.append("title", title.trim());
      fd.append("description", description.trim());
      fd.append("show_in_onboarding", showInOnboarding ? "true" : "false");
      fd.append("file", file);

      const res = await fetch(`/api/eegs/${eegId}/documents`, {
        method: "POST",
        body: fd,
      });

      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        setUploadError(
          (d as { error?: string }).error ||
            "Upload fehlgeschlagen. Bitte versuchen Sie es erneut."
        );
        return;
      }

      const newDoc: Document = await res.json();
      setDocuments((prev) => [newDoc, ...prev]);
      setTitle("");
      setDescription("");
      setShowInOnboarding(false);
      setFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      setUploadSuccess(true);
      setTimeout(() => setUploadSuccess(false), 3000);
    } catch {
      setUploadError("Netzwerkfehler. Bitte prüfen Sie Ihre Verbindung.");
    } finally {
      setUploading(false);
    }
  }

  async function handleDelete(docId: string) {
    if (!confirm("Dokument wirklich löschen?")) return;
    setDeletingId(docId);
    try {
      const res = await fetch(`/api/eegs/${eegId}/documents/${docId}`, {
        method: "DELETE",
      });
      if (res.ok) {
        setDocuments((prev) => prev.filter((d) => d.id !== docId));
      }
    } finally {
      setDeletingId(null);
    }
  }

  async function handleToggleOnboarding(docId: string, current: boolean) {
    setTogglingId(docId);
    try {
      const res = await fetch(`/api/eegs/${eegId}/documents/${docId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ show_in_onboarding: !current }),
      });
      if (res.ok) {
        setDocuments((prev) =>
          prev.map((d) =>
            d.id === docId ? { ...d, show_in_onboarding: !current } : d
          )
        );
      }
    } finally {
      setTogglingId(null);
    }
  }

  async function handleDownload(docId: string, filename: string) {
    const res = await fetch(`/api/eegs/${eegId}/documents/${docId}`);
    if (!res.ok) return;
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  const inputClass =
    "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm";

  return (
    <div className="space-y-6">
      {/* Upload form */}
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <h2 className="text-base font-semibold text-slate-900 mb-4">
          Dokument hochladen
        </h2>
        <form onSubmit={handleUpload} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">
              Titel <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="z.B. Allgemeine Geschäftsbedingungen"
              className={inputClass}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">
              Beschreibung (optional)
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Kurze Beschreibung des Dokuments"
              className={inputClass}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">
              Datei <span className="text-red-500">*</span>
            </label>
            <input
              ref={fileInputRef}
              type="file"
              onChange={handleFileChange}
              className="w-full text-sm text-slate-600 file:mr-4 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100 transition-colors cursor-pointer"
              required
            />
            {file && (
              <p className="text-xs text-slate-400 mt-1">
                {formatFileSize(file.size)}
              </p>
            )}
          </div>
          <div>
            <label className="flex items-center gap-2.5 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={showInOnboarding}
                onChange={(e) => setShowInOnboarding(e.target.checked)}
                className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="text-sm text-slate-700">Im Onboarding anzeigen</span>
            </label>
            <p className="text-xs text-slate-400 mt-1 ml-6.5">
              Dokument wird im Anmeldeformular als Download angeboten und in der Bestätigungsmail verlinkt.
            </p>
          </div>

          {uploadError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {uploadError}
            </div>
          )}
          {uploadSuccess && (
            <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700">
              Dokument erfolgreich hochgeladen.
            </div>
          )}

          <button
            type="submit"
            disabled={uploading || !file || !title.trim()}
            className="px-5 py-2 bg-blue-700 text-white font-medium rounded-lg text-sm hover:bg-blue-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {uploading && (
              <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
            )}
            {uploading ? "Wird hochgeladen…" : "Hochladen"}
          </button>
        </form>
      </div>

      {/* Document list */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-100">
          <h2 className="text-base font-semibold text-slate-900">
            Vorhandene Dokumente
            {documents.length > 0 && (
              <span className="ml-2 text-sm font-normal text-slate-400">
                ({documents.length})
              </span>
            )}
          </h2>
        </div>

        {documents.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <svg className="w-10 h-10 text-slate-300 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <p className="text-slate-400 text-sm">Noch keine Dokumente hochgeladen.</p>
          </div>
        ) : (
          <ul className="divide-y divide-slate-100">
            {documents.map((doc) => (
              <li key={doc.id} className="px-6 py-4 flex items-center gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="font-medium text-slate-900 text-sm truncate">{doc.title}</p>
                    {doc.show_in_onboarding && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
                        Onboarding
                      </span>
                    )}
                  </div>
                  {doc.description && (
                    <p className="text-xs text-slate-500 mt-0.5 truncate">{doc.description}</p>
                  )}
                  <p className="text-xs text-slate-400 mt-0.5 font-mono">
                    {doc.filename} &middot; {formatFileSize(doc.file_size_bytes)} &middot; {fmtDate(doc.created_at)}
                  </p>
                </div>
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    type="button"
                    onClick={() => handleToggleOnboarding(doc.id, doc.show_in_onboarding)}
                    disabled={togglingId === doc.id}
                    title={doc.show_in_onboarding ? "Aus Onboarding entfernen" : "Im Onboarding anzeigen"}
                    className={`px-2 py-1 rounded text-xs font-medium transition-colors disabled:opacity-50 ${
                      doc.show_in_onboarding
                        ? "bg-blue-50 text-blue-700 hover:bg-blue-100"
                        : "bg-slate-100 text-slate-500 hover:bg-slate-200"
                    }`}
                  >
                    {doc.show_in_onboarding ? "Im Onboarding" : "Onboarding"}
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDownload(doc.id, doc.filename)}
                    title="Herunterladen"
                    className="p-1.5 text-slate-400 hover:text-blue-600 transition-colors"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDelete(doc.id)}
                    disabled={deletingId === doc.id}
                    title="Löschen"
                    className="p-1.5 text-slate-400 hover:text-red-500 transition-colors disabled:opacity-50"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
