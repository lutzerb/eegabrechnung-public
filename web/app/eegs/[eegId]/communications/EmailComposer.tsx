"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Link from "@tiptap/extension-link";
import Underline from "@tiptap/extension-underline";
import TextAlign from "@tiptap/extension-text-align";
import Image from "@tiptap/extension-image";

interface Props {
  eegId: string;
}

interface Member {
  id: string;
  name1: string;
  name2: string;
  email: string;
  status: string;
  mitglieds_nr: string;
}

const PLACEHOLDERS = [
  { key: "{{vorname}}",      label: "Vorname" },
  { key: "{{nachname}}",     label: "Nachname" },
  { key: "{{name}}",         label: "Vollst. Name" },
  { key: "{{mitglieds_nr}}", label: "Mitgl.-Nr." },
  { key: "{{eeg_name}}",     label: "EEG-Name" },
  { key: "{{email}}",        label: "E-Mail" },
];

const PLACEHOLDER_EXAMPLES: Record<string, string> = {
  "{{vorname}}":      "Maria",
  "{{nachname}}":     "Mustermann",
  "{{name}}":         "Maria Mustermann",
  "{{mitglieds_nr}}": "0042",
  "{{eeg_name}}":     "Sonnenstrom Mustertal",
  "{{email}}":        "maria@beispiel.at",
};

function applyExamples(s: string): string {
  return Object.entries(PLACEHOLDER_EXAMPLES).reduce(
    (acc, [key, val]) => acc.replaceAll(key, val),
    s
  );
}

function ToolbarButton({ onClick, active, title, children }: {
  onClick: () => void; active?: boolean; title: string; children: React.ReactNode;
}) {
  return (
    <button type="button" onClick={onClick} title={title}
      className={`p-1.5 rounded text-sm transition-colors ${active ? "bg-blue-100 text-blue-700" : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"}`}>
      {children}
    </button>
  );
}

function ToolbarSep() {
  return <div className="w-px h-5 bg-slate-300 mx-0.5 self-center" />;
}

export default function EmailComposer({ eegId }: Props) {
  const router = useRouter();
  const [subject, setSubject] = useState("");
  const [attachments, setAttachments] = useState<File[]>([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const subjectInputRef = useRef<HTMLInputElement>(null);
  const lastFocusedRef = useRef<"subject" | "body">("body");

  // Recipient selection
  const [recipientMode, setRecipientMode] = useState<"all" | "selected">("all");
  const [members, setMembers] = useState<Member[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [memberSearch, setMemberSearch] = useState("");

  useEffect(() => {
    if (recipientMode !== "selected" || members.length > 0) return;
    setMembersLoading(true);
    fetch(`/api/eegs/${eegId}/members`)
      .then(r => r.json())
      .then((data: Member[]) => {
        const active = (Array.isArray(data) ? data : []).filter(m => m.status !== "INACTIVE" && m.email);
        setMembers(active);
        setSelectedIds(new Set());
      })
      .catch(() => {})
      .finally(() => setMembersLoading(false));
  }, [recipientMode, eegId, members.length]);

  const filteredMembers = members.filter(m => {
    if (!memberSearch) return true;
    const q = memberSearch.toLowerCase();
    return `${m.name1} ${m.name2}`.toLowerCase().includes(q) || m.email.toLowerCase().includes(q);
  });

  function toggleMember(id: string) {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }

  function toggleAll() {
    if (selectedIds.size === members.length) setSelectedIds(new Set());
    else setSelectedIds(new Set(members.map(m => m.id)));
  }

  const editor = useEditor({
    extensions: [StarterKit, Underline, Link.configure({ openOnClick: false }),
      TextAlign.configure({ types: ["heading", "paragraph"] }), Image],
    content: "<p></p>",
    editorProps: {
      attributes: { class: "prose prose-sm max-w-none focus:outline-none min-h-[200px] px-4 py-3 text-slate-800" },
      handleDOMEvents: { focus: () => { lastFocusedRef.current = "body"; return false; } },
    },
  });

  function insertPlaceholder(key: string) {
    if (lastFocusedRef.current === "subject") {
      const el = subjectInputRef.current;
      if (!el) return;
      const start = el.selectionStart ?? subject.length;
      const end = el.selectionEnd ?? subject.length;
      const newVal = subject.slice(0, start) + key + subject.slice(end);
      setSubject(newVal);
      requestAnimationFrame(() => { el.focus(); el.setSelectionRange(start + key.length, start + key.length); });
    } else {
      editor?.chain().focus().insertContent(key).run();
    }
  }

  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files) return;
    setAttachments(prev => [...prev, ...Array.from(e.target.files as FileList)]);
    e.target.value = "";
  }, []);

  const removeAttachment = useCallback((index: number) => {
    setAttachments(prev => prev.filter((_, i) => i !== index));
  }, []);

  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  const setLink = useCallback(() => {
    if (!editor) return;
    const previousUrl = editor.getAttributes("link").href as string | undefined;
    const url = window.prompt("URL eingeben:", previousUrl ?? "https://");
    if (url === null) return;
    if (url === "") editor.chain().focus().extendMarkRange("link").unsetLink().run();
    else editor.chain().focus().extendMarkRange("link").setLink({ href: url }).run();
  }, [editor]);

  async function doSend() {
    setShowConfirm(false);
    setError(null);
    setSending(true);
    try {
      const fd = new FormData();
      fd.append("subject", subject);
      fd.append("html_body", editor?.getHTML() ?? "");
      if (recipientMode === "selected") {
        fd.append("member_ids", JSON.stringify(Array.from(selectedIds)));
      }
      for (const file of attachments) fd.append("files", file);
      const res = await fetch(`/api/eegs/${eegId}/communications`, { method: "POST", body: fd });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        setError((d as { error?: string }).error || "Senden fehlgeschlagen. Bitte versuchen Sie es erneut.");
        return;
      }
      setSubject("");
      setAttachments([]);
      editor?.commands.setContent("<p></p>");
      router.refresh();
    } catch {
      setError("Netzwerkfehler. Bitte prüfen Sie Ihre Verbindung.");
    } finally {
      setSending(false);
    }
  }

  if (!editor) return null;

  const previewSubject = applyExamples(subject);
  const previewBody = applyExamples(editor.getHTML());
  const recipientCount = recipientMode === "all" ? null : selectedIds.size;

  return (
    <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
      <div className="p-6 border-b border-slate-100">
        <h2 className="text-base font-semibold text-slate-900 mb-4">Neue Mitglieder-E-Mail verfassen</h2>

        {/* Subject */}
        <div className="mb-3">
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Betreff <span className="text-red-500">*</span>
          </label>
          <input ref={subjectInputRef} type="text" value={subject}
            onChange={e => setSubject(e.target.value)}
            onFocus={() => { lastFocusedRef.current = "subject"; }}
            placeholder="Betreff der E-Mail"
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
        </div>

        {/* Placeholder chips */}
        <div className="mb-4">
          <p className="text-xs text-slate-500 mb-1.5">
            Platzhalter einfügen — Klick fügt an der Cursorposition in Betreff oder Nachricht ein:
          </p>
          <div className="flex flex-wrap gap-1.5">
            {PLACEHOLDERS.map(p => (
              <button key={p.key} type="button" onClick={() => insertPlaceholder(p.key)} title={p.key}
                className="px-2 py-0.5 rounded border border-slate-300 bg-slate-50 text-xs font-mono text-slate-700 hover:bg-blue-50 hover:border-blue-400 hover:text-blue-700 transition-colors">
                {p.label}
              </button>
            ))}
          </div>
        </div>

        {/* Editor */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Nachricht</label>
          <div className="flex flex-wrap items-center gap-0.5 px-2 py-1.5 border border-slate-300 rounded-t-lg bg-slate-50">
            <ToolbarButton onClick={() => editor.chain().focus().toggleBold().run()} active={editor.isActive("bold")} title="Fett"><strong>B</strong></ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().toggleItalic().run()} active={editor.isActive("italic")} title="Kursiv"><em>I</em></ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().toggleUnderline().run()} active={editor.isActive("underline")} title="Unterstrichen"><span className="underline">U</span></ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().toggleStrike().run()} active={editor.isActive("strike")} title="Durchgestrichen"><span className="line-through">S</span></ToolbarButton>
            <ToolbarSep />
            <ToolbarButton onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()} active={editor.isActive("heading", { level: 1 })} title="Überschrift 1"><span className="font-bold text-xs">H1</span></ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} active={editor.isActive("heading", { level: 2 })} title="Überschrift 2"><span className="font-bold text-xs">H2</span></ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()} active={editor.isActive("heading", { level: 3 })} title="Überschrift 3"><span className="font-bold text-xs">H3</span></ToolbarButton>
            <ToolbarSep />
            <ToolbarButton onClick={() => editor.chain().focus().toggleBulletList().run()} active={editor.isActive("bulletList")} title="Aufzählungsliste">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" /></svg>
            </ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().toggleOrderedList().run()} active={editor.isActive("orderedList")} title="Nummerierte Liste">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 6h13M7 12h13M7 18h13M3 6h.01M3 12h.01M3 18h.01" /></svg>
            </ToolbarButton>
            <ToolbarSep />
            <ToolbarButton onClick={setLink} active={editor.isActive("link")} title="Link einfügen">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>
            </ToolbarButton>
            <ToolbarSep />
            <ToolbarButton onClick={() => editor.chain().focus().setTextAlign("left").run()} active={editor.isActive({ textAlign: "left" })} title="Linksbündig">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h14" /></svg>
            </ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().setTextAlign("center").run()} active={editor.isActive({ textAlign: "center" })} title="Zentriert">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M7 12h10M5 18h14" /></svg>
            </ToolbarButton>
            <ToolbarButton onClick={() => editor.chain().focus().setTextAlign("right").run()} active={editor.isActive({ textAlign: "right" })} title="Rechtsbündig">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M10 12h10M6 18h14" /></svg>
            </ToolbarButton>
            <ToolbarSep />
            <ToolbarButton onClick={() => editor.chain().focus().unsetAllMarks().clearNodes().run()} title="Formatierung entfernen">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
            </ToolbarButton>
          </div>
          <div className="border border-t-0 border-slate-300 rounded-b-lg bg-white">
            <EditorContent editor={editor} />
          </div>
        </div>

        {/* Attachments */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Anhänge (optional)</label>
          <input ref={fileInputRef} type="file" multiple onChange={handleFileChange} className="hidden" />
          <button type="button" onClick={() => fileInputRef.current?.click()}
            className="flex items-center gap-2 px-3 py-2 border border-dashed border-slate-300 rounded-lg text-sm text-slate-600 hover:border-blue-400 hover:text-blue-600 transition-colors">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
            </svg>
            Dateien hinzufügen
          </button>
          {attachments.length > 0 && (
            <ul className="mt-2 space-y-1">
              {attachments.map((file, i) => (
                <li key={i} className="flex items-center justify-between gap-2 px-3 py-1.5 bg-slate-50 rounded-lg text-sm">
                  <span className="text-slate-700 truncate">{file.name}</span>
                  <span className="text-slate-400 text-xs flex-shrink-0">{formatFileSize(file.size)}</span>
                  <button type="button" onClick={() => removeAttachment(i)} className="text-slate-400 hover:text-red-500 transition-colors flex-shrink-0">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Recipient selection */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">Empfänger</label>
          <div className="flex gap-2 mb-3">
            <button type="button" onClick={() => setRecipientMode("all")}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors border ${recipientMode === "all" ? "bg-blue-700 text-white border-blue-700" : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"}`}>
              Alle aktiven Mitglieder
            </button>
            <button type="button" onClick={() => setRecipientMode("selected")}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors border ${recipientMode === "selected" ? "bg-blue-700 text-white border-blue-700" : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"}`}>
              Auswahl
              {recipientMode === "selected" && selectedIds.size > 0 && (
                <span className="ml-1.5 px-1.5 py-0.5 bg-blue-500 rounded text-xs">{selectedIds.size}</span>
              )}
            </button>
          </div>

          {recipientMode === "selected" && (
            <div className="border border-slate-200 rounded-lg overflow-hidden">
              {/* Search + select all */}
              <div className="flex items-center gap-2 px-3 py-2 bg-slate-50 border-b border-slate-200">
                <input type="text" value={memberSearch} onChange={e => setMemberSearch(e.target.value)}
                  placeholder="Suchen…"
                  className="flex-1 text-sm px-2 py-1 border border-slate-300 rounded focus:outline-none focus:ring-1 focus:ring-blue-500" />
                {!membersLoading && members.length > 0 && (
                  <button type="button" onClick={toggleAll}
                    className="text-xs text-slate-500 hover:text-blue-600 whitespace-nowrap transition-colors">
                    {selectedIds.size === members.length ? "Alle abwählen" : "Alle auswählen"}
                  </button>
                )}
              </div>

              {membersLoading ? (
                <div className="px-4 py-6 text-sm text-slate-400 text-center">Wird geladen…</div>
              ) : members.length === 0 ? (
                <div className="px-4 py-6 text-sm text-slate-400 text-center">Keine aktiven Mitglieder mit E-Mail-Adresse.</div>
              ) : (
                <div className="max-h-52 overflow-y-auto divide-y divide-slate-100">
                  {filteredMembers.map(m => (
                    <label key={m.id} className="flex items-center gap-3 px-3 py-2 hover:bg-slate-50 cursor-pointer">
                      <input type="checkbox" checked={selectedIds.has(m.id)} onChange={() => toggleMember(m.id)}
                        className="rounded border-slate-300 text-blue-600 focus:ring-blue-500" />
                      <div className="min-w-0">
                        <p className="text-sm text-slate-900 truncate">{[m.name1, m.name2].filter(Boolean).join(" ")}</p>
                        <p className="text-xs text-slate-400 truncate">{m.email}</p>
                      </div>
                    </label>
                  ))}
                  {filteredMembers.length === 0 && (
                    <div className="px-4 py-4 text-sm text-slate-400 text-center">Keine Treffer.</div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        {error && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{error}</div>
        )}
      </div>

      {/* Actions */}
      <div className="px-6 py-4 bg-slate-50 flex items-center gap-3">
        <button type="button" onClick={() => setShowPreview(true)} disabled={!subject.trim()}
          className="px-4 py-2 border border-slate-300 text-slate-700 font-medium rounded-lg text-sm hover:bg-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          Vorschau
        </button>
        <button type="button"
          onClick={() => {
            if (!subject.trim()) { setError("Bitte geben Sie einen Betreff ein."); return; }
            if (recipientMode === "selected" && selectedIds.size === 0) { setError("Bitte wählen Sie mindestens ein Mitglied aus."); return; }
            setError(null); setShowConfirm(true);
          }}
          disabled={sending}
          className="px-5 py-2 bg-blue-700 text-white font-medium rounded-lg text-sm hover:bg-blue-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2">
          {sending && (
            <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
          )}
          {sending ? "Wird gesendet…" : "Senden"}
        </button>
      </div>

      {/* Preview Modal */}
      {showPreview && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-xl shadow-xl max-w-2xl w-full max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
              <div>
                <p className="text-xs text-slate-400 mb-0.5">Vorschau mit Beispiel-Mitglied</p>
                <p className="text-xs text-slate-500 mb-0.5">Betreff</p>
                <p className="font-semibold text-slate-900">{previewSubject || "(kein Betreff)"}</p>
              </div>
              <button type="button" onClick={() => setShowPreview(false)} className="text-slate-400 hover:text-slate-700 transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-6">
              <div className="prose prose-sm max-w-none text-slate-800" dangerouslySetInnerHTML={{ __html: previewBody }} />
            </div>
            {attachments.length > 0 && (
              <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
                <p className="text-xs text-slate-500">{attachments.length} Anhang/Anhänge: {attachments.map(f => f.name).join(", ")}</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Confirm Modal */}
      {showConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-xl shadow-xl max-w-sm w-full p-6">
            <h3 className="font-semibold text-slate-900 mb-2">E-Mail senden?</h3>
            <p className="text-sm text-slate-600 mb-6">
              {recipientCount !== null
                ? `Diese E-Mail wird an ${recipientCount} ausgewählte${recipientCount === 1 ? "s" : ""} Mitglied${recipientCount === 1 ? "" : "er"} gesendet.`
                : "Diese E-Mail wird an alle aktiven Mitglieder gesendet."}
            </p>
            <div className="flex gap-3">
              <button type="button" onClick={() => setShowConfirm(false)}
                className="flex-1 px-4 py-2 border border-slate-300 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-50 transition-colors">
                Abbrechen
              </button>
              <button type="button" onClick={doSend}
                className="flex-1 px-4 py-2 bg-blue-700 text-white rounded-lg text-sm font-medium hover:bg-blue-800 transition-colors">
                Jetzt senden
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
