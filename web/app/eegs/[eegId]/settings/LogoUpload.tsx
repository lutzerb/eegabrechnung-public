"use client";

import { useRef, useState } from "react";
import { useRouter } from "next/navigation";

interface Props {
  eegId: string;
  currentLogoPath?: string | null;
}

export default function LogoUpload({ eegId, currentLogoPath }: Props) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  // Store the selected file in state so it survives the input being unmounted
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError(null);
    setSelectedFile(file);
    setPreview(URL.createObjectURL(file));
  }

  function handleCancel() {
    setSelectedFile(null);
    setPreview(null);
    setError(null);
    if (inputRef.current) inputRef.current.value = "";
  }

  async function handleConfirm() {
    if (!selectedFile) return;

    setUploading(true);
    setError(null);

    const formData = new FormData();
    formData.append("logo", selectedFile);

    try {
      const res = await fetch(`/api/eegs/${eegId}/logo`, {
        method: "POST",
        body: formData,
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Hochladen fehlgeschlagen.");
        return;
      }
      setSelectedFile(null);
      setPreview(null);
      router.refresh();
    } catch {
      setError("Netzwerkfehler beim Hochladen.");
    } finally {
      setUploading(false);
    }
  }

  return (
    <div>
      {/* Current logo — shown when no preview pending */}
      {currentLogoPath && !preview && (
        <div className="mb-4">
          <p className="text-xs text-slate-500 mb-2">Aktuelles Logo:</p>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={`/api/eegs/${eegId}/logo`}
            alt="Aktuelles Logo"
            className="max-h-16 max-w-48 object-contain border border-slate-200 rounded p-1 bg-white"
          />
        </div>
      )}

      {/* Preview + confirm */}
      {preview && (
        <div className="mb-4 p-4 bg-slate-50 border border-slate-200 rounded-lg">
          <p className="text-xs font-medium text-slate-700 mb-2">
            Vorschau — {selectedFile?.name}
          </p>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={preview}
            alt="Vorschau"
            className="max-h-24 max-w-64 object-contain border border-slate-200 rounded p-1 bg-white mb-3"
          />
          {error && <p className="text-xs text-red-600 mb-3">{error}</p>}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={handleConfirm}
              disabled={uploading}
              className="px-3 py-1.5 text-xs bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-60"
            >
              {uploading ? "Hochladen…" : "Bestätigen & hochladen"}
            </button>
            <button
              type="button"
              onClick={handleCancel}
              disabled={uploading}
              className="px-3 py-1.5 text-xs border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-100 transition-colors disabled:opacity-60"
            >
              Abbrechen
            </button>
          </div>
        </div>
      )}

      {/* File picker */}
      {!preview && (
        <input
          ref={inputRef}
          type="file"
          accept="image/jpeg,image/png"
          onChange={handleFileChange}
          className="block text-sm text-slate-700 file:mr-4 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-xs file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
        />
      )}
    </div>
  );
}
