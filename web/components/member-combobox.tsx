"use client";

import { useState, useRef, useEffect } from "react";
import type { Member } from "@/lib/api";

interface Props {
  members: Member[];
  value: string;
  onChange: (memberId: string) => void;
  placeholder?: string;
  className?: string;
}

function memberLabel(m: Member): string {
  return [m.name1, m.name2].filter(Boolean).join(" ") || m.name || m.id.slice(0, 8);
}

export default function MemberCombobox({ members, value, onChange, placeholder = "Alle Mitglieder", className = "" }: Props) {
  const selected = members.find((m) => m.id === value);
  const [query, setQuery] = useState(selected ? memberLabel(selected) : "");
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const m = members.find((mm) => mm.id === value);
    setQuery(m ? memberLabel(m) : "");
  }, [value, members]);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        const m = members.find((mm) => mm.id === value);
        setQuery(m ? memberLabel(m) : "");
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [members, value]);

  const filtered = query.trim().length < 1
    ? members
    : members.filter((m) => memberLabel(m).toLowerCase().includes(query.toLowerCase()));

  function select(m: Member) {
    onChange(m.id);
    setQuery(memberLabel(m));
    setOpen(false);
  }

  function clear() {
    onChange("");
    setQuery("");
    setOpen(false);
  }

  return (
    <div ref={ref} className={`relative ${className}`}>
      <input
        type="text"
        value={query}
        onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
        onFocus={() => setOpen(true)}
        placeholder={placeholder}
        autoComplete="off"
        className="w-full px-3 py-1.5 text-sm border border-slate-200 rounded-lg bg-white text-slate-700 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
      {open && (
        <ul className="absolute z-50 mt-1 w-full max-h-64 overflow-y-auto bg-white border border-slate-200 rounded-lg shadow-lg text-sm">
          <li
            onMouseDown={() => clear()}
            className="px-3 py-2 hover:bg-blue-50 cursor-pointer text-slate-500"
          >
            Alle Mitglieder
          </li>
          {filtered.length === 0 && (
            <li className="px-3 py-2 text-slate-400">Keine Treffer</li>
          )}
          {filtered.map((m) => (
            <li
              key={m.id}
              onMouseDown={() => select(m)}
              className={`px-3 py-2 hover:bg-blue-50 cursor-pointer truncate ${
                m.id === value ? "bg-blue-50 text-blue-700 font-medium" : "text-slate-800"
              }`}
            >
              {memberLabel(m)}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
