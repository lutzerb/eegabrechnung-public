"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { signOut, useSession } from "next-auth/react";
import type { Session } from "next-auth";
import clsx from "clsx";
import { useState, useEffect, useRef } from "react";
import { SearchOverlay } from "@/components/search-overlay";

interface NavProps {
  session: Session;
}

interface EEG {
  id: string;
  name: string;
}

interface NavChild {
  segment: string;
  label: string;
}

interface NavItem {
  segment: string;
  label: string;
  exact?: boolean;
  icon: React.ReactNode;
  children?: NavChild[];
}

const EEG_NAV: NavItem[] = [
  {
    segment: "",
    label: "Übersicht",
    exact: true,
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
      </svg>
    ),
  },
  {
    segment: "billing",
    label: "Abrechnung",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M9 14l6-6m-5.5.5h.01m4.99 5h.01M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16l3.5-2 3.5 2 3.5-2 3.5 2z" />
      </svg>
    ),
  },
  {
    segment: "members",
    label: "Mitglieder",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
  {
    segment: "import",
    label: "Daten importieren",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
      </svg>
    ),
  },
  {
    segment: "tariffs",
    label: "Tarifpläne",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
      </svg>
    ),
  },
  {
    segment: "reports",
    label: "Auswertungen",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    ),
  },
  {
    segment: "eda",
    label: "EDA",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
      </svg>
    ),
  },
  {
    segment: "onboarding",
    label: "Onboarding",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
      </svg>
    ),
  },
  {
    segment: "communications",
    label: "Aussendungen",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
      </svg>
    ),
  },
  {
    segment: "documents",
    label: "Dokumente",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
      </svg>
    ),
  },
  {
    segment: "accounting",
    label: "Buchhaltungsexport",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
      </svg>
    ),
  },
  {
    segment: "ea",
    label: "E/A-Buchhaltung",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
      </svg>
    ),
    children: [
      { segment: "ea/buchungen", label: "Journal" },
      { segment: "ea/buchungen/neu", label: "Buchung erfassen" },
      { segment: "ea/konten", label: "Kontenplan" },
      { segment: "ea/saldenliste", label: "Saldenliste" },
      { segment: "ea/jahresabschluss", label: "Jahresabschluss" },
      { segment: "ea/uva", label: "UVA" },
      { segment: "ea/erklaerungen", label: "Jahreserklärungen" },
      { segment: "ea/import", label: "Rechnungsimport" },
      { segment: "ea/bank", label: "Bankimport" },
      { segment: "ea/changelog", label: "Audit-Trail" },
      { segment: "ea/settings", label: "E/A-Einstellungen" },
    ],
  },
  {
    segment: "settings",
    label: "Einstellungen",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
];

function EEGSwitcher({ currentId, currentName, eegs, loading }: {
  currentId: string;
  currentName: string;
  eegs: EEG[];
  loading: boolean;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const router = useRouter();

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  return (
    <div ref={ref} className="relative px-3 mb-2">
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center gap-2 px-3 py-2 rounded-lg bg-blue-50 border border-blue-200 hover:bg-blue-100 transition-colors group"
      >
        <div className="w-6 h-6 rounded bg-blue-700 flex items-center justify-center flex-shrink-0">
          <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <span className="flex-1 text-left text-sm font-semibold text-blue-900 truncate min-w-0">
          {loading ? "..." : currentName}
        </span>
        <svg className={clsx("w-4 h-4 text-blue-500 flex-shrink-0 transition-transform", open && "rotate-180")}
          fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        <div className="absolute left-3 right-3 top-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg z-50 overflow-hidden">
          <div className="py-1 max-h-56 overflow-y-auto">
            {eegs.map((eeg) => (
              <button
                key={eeg.id}
                onClick={() => { router.push(`/eegs/${eeg.id}`); setOpen(false); }}
                className={clsx(
                  "w-full text-left px-3 py-2 text-sm transition-colors flex items-center gap-2",
                  eeg.id === currentId
                    ? "bg-blue-50 text-blue-700 font-medium"
                    : "text-slate-700 hover:bg-slate-50"
                )}
              >
                {eeg.id === currentId && (
                  <svg className="w-3.5 h-3.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                  </svg>
                )}
                <span className={clsx("truncate", eeg.id !== currentId && "ml-5")}>{eeg.name}</span>
              </button>
            ))}
            <div className="border-t border-slate-100 mt-1 pt-1">
              <Link
                href="/eegs"
                onClick={() => setOpen(false)}
                className="w-full block px-3 py-2 text-xs text-slate-500 hover:bg-slate-50 transition-colors"
              >
                Alle Energiegemeinschaften →
              </Link>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function Nav({ session }: NavProps) {
  const pathname = usePathname();
  const { data: clientSession } = useSession();
  const isAdmin = session.role === "admin";
  const [searchOpen, setSearchOpen] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

  const userName = session.user?.name || session.user?.email || "Benutzer";
  const userInitials = userName
    .split(" ")
    .map((n: string) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  // Detect current EEG from path
  const eegMatch = pathname.match(/^\/eegs\/([^/]+)/);
  const currentEegId = eegMatch ? eegMatch[1] : null;

  // Auto-expand groups whose children match the current path
  useEffect(() => {
    if (!currentEegId) return;
    const toExpand = new Set<string>();
    for (const item of EEG_NAV) {
      if (item.children) {
        const anyChildActive = item.children.some((child) =>
          pathname.startsWith(`/eegs/${currentEegId}/${child.segment}`)
        );
        if (anyChildActive) toExpand.add(item.segment);
      }
    }
    if (toExpand.size > 0) {
      setExpandedGroups((prev) => {
        const next = new Set(prev);
        toExpand.forEach((s) => next.add(s));
        return next;
      });
    }
  }, [pathname, currentEegId]);

  // Close mobile nav on route change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  // Open search with '/' key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "/" && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const tag = (e.target as HTMLElement)?.tagName?.toLowerCase();
        if (tag === "input" || tag === "textarea" || tag === "select") return;
        if (currentEegId) {
          e.preventDefault();
          setSearchOpen(true);
        }
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [currentEegId]);

  // Load EEG list for switcher
  const [eegs, setEegs] = useState<EEG[]>([]);
  const [loadingEegs, setLoadingEegs] = useState(false);

  useEffect(() => {
    if (!currentEegId || !clientSession?.accessToken) return;
    setLoadingEegs(true);
    fetch("/api/eegs", { headers: { Authorization: `Bearer ${clientSession.accessToken}` } })
      .then((r) => r.json())
      .then((data) => setEegs(Array.isArray(data) ? data : []))
      .catch(() => {})
      .finally(() => setLoadingEegs(false));
  }, [currentEegId, clientSession?.accessToken]);

  const currentEeg = eegs.find((e) => e.id === currentEegId);

  return (
    <>
    {searchOpen && currentEegId && (
      <SearchOverlay eegId={currentEegId} onClose={() => setSearchOpen(false)} />
    )}

    {/* Mobile top bar */}
    <div className="sm:hidden fixed top-0 left-0 right-0 z-20 h-14 bg-white border-b border-slate-200 flex items-center px-4 gap-3">
      <button
        onClick={() => setMobileOpen((o) => !o)}
        className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-slate-100 transition-colors"
        aria-label="Menü öffnen"
      >
        <svg className="w-5 h-5 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
      <Link href="/dashboard" className="flex items-center gap-2">
        <div className="w-7 h-7 rounded-md bg-blue-700 flex items-center justify-center flex-shrink-0">
          <svg className="w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <span className="font-bold text-slate-900 text-sm">EEG Abrechnung</span>
      </Link>
    </div>

    {/* Mobile overlay backdrop */}
    {mobileOpen && (
      <div
        className="sm:hidden fixed inset-0 z-20 bg-black/40"
        onClick={() => setMobileOpen(false)}
      />
    )}

    <aside className={`fixed top-0 left-0 h-full w-64 bg-white border-r border-slate-200 flex flex-col z-30 transition-transform duration-200 ${mobileOpen ? "translate-x-0" : "-translate-x-full sm:translate-x-0"}`}>
      {/* Brand */}
      <div className="px-5 py-4 border-b border-slate-200 flex-shrink-0">
        <Link href="/dashboard" className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-blue-700 flex items-center justify-center flex-shrink-0">
            <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <div>
            <p className="font-bold text-slate-900 text-sm leading-tight">EEG Abrechnung</p>
            <p className="text-xs text-slate-400 leading-tight">Österreich</p>
          </div>
        </Link>
      </div>

      <nav className="flex-1 py-3 overflow-y-auto flex flex-col gap-0">
        {currentEegId ? (
          <>
            {/* EEG switcher */}
            <EEGSwitcher
              currentId={currentEegId}
              currentName={currentEeg?.name ?? (loadingEegs ? "..." : currentEegId.slice(0, 8) + "…")}
              eegs={eegs}
              loading={loadingEegs}
            />

            {/* Search button */}
            <div className="px-3 mb-1">
              <button
                onClick={() => setSearchOpen(true)}
                className="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm text-slate-500 hover:bg-slate-50 hover:text-slate-700 transition-colors border border-slate-200"
              >
                <svg className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <span className="flex-1 text-left text-xs">Suchen…</span>
                <kbd className="text-xs text-slate-300 font-mono">/</kbd>
              </button>
            </div>

            {/* EEG sub-nav */}
            <div className="px-3 space-y-0.5">
              {EEG_NAV.map(({ segment, label, exact, icon, children }) => {
                const href = `/eegs/${currentEegId}${segment ? `/${segment}` : ""}`;
                const isActive = exact
                  ? pathname === `/eegs/${currentEegId}`
                  : pathname.startsWith(`/eegs/${currentEegId}/${segment}`);
                const isExpanded = expandedGroups.has(segment);

                if (children) {
                  const anyChildActive = children.some((child) =>
                    pathname.startsWith(`/eegs/${currentEegId}/${child.segment}`)
                  );
                  void anyChildActive; // used for auto-expand logic via useEffect
                  return (
                    <div key={segment}>
                      <div className="flex items-center gap-0.5">
                        <Link
                          href={href}
                          className={clsx(
                            "flex-1 flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                            isActive
                              ? "bg-blue-50 text-blue-700"
                              : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                          )}
                        >
                          <span className={clsx("flex-shrink-0", isActive ? "text-blue-600" : "text-slate-400")}>
                            {icon}
                          </span>
                          {label}
                        </Link>
                        <button
                          onClick={() => setExpandedGroups((prev) => {
                            const next = new Set(prev);
                            if (next.has(segment)) next.delete(segment); else next.add(segment);
                            return next;
                          })}
                          className={clsx(
                            "flex-shrink-0 w-7 h-7 flex items-center justify-center rounded-lg transition-colors",
                            isActive ? "text-blue-500 hover:bg-blue-100" : "text-slate-400 hover:bg-slate-100"
                          )}
                          aria-label={isExpanded ? "Einklappen" : "Ausklappen"}
                        >
                          <svg className={clsx("w-3.5 h-3.5 transition-transform", isExpanded && "rotate-180")} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                          </svg>
                        </button>
                      </div>
                      {isExpanded && (
                        <div className="ml-4 pl-3 border-l border-slate-200 mt-0.5 space-y-0.5">
                          {children.map((child) => {
                            const childHref = `/eegs/${currentEegId}/${child.segment}`;
                            const childActive = pathname === childHref || pathname.startsWith(childHref + "/");
                            return (
                              <Link
                                key={child.segment}
                                href={childHref}
                                className={clsx(
                                  "block px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
                                  childActive
                                    ? "bg-blue-50 text-blue-700"
                                    : "text-slate-500 hover:bg-slate-50 hover:text-slate-800"
                                )}
                              >
                                {child.label}
                              </Link>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                }

                return (
                  <Link
                    key={segment}
                    href={href}
                    className={clsx(
                      "flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                      isActive
                        ? "bg-blue-50 text-blue-700"
                        : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                    )}
                  >
                    <span className={clsx("flex-shrink-0", isActive ? "text-blue-600" : "text-slate-400")}>
                      {icon}
                    </span>
                    {label}
                  </Link>
                );
              })}
            </div>

            {/* Divider + global links */}
            <div className="mx-3 my-3 border-t border-slate-100" />
            <div className="px-3 space-y-0.5">
              <Link
                href="/eegs"
                className={clsx(
                  "flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                  pathname === "/eegs"
                    ? "bg-blue-50 text-blue-700"
                    : "text-slate-500 hover:bg-slate-50 hover:text-slate-900"
                )}
              >
                <svg className="w-4 h-4 flex-shrink-0 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                    d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                </svg>
                Alle Gemeinschaften
              </Link>
              {isAdmin && (
                <Link
                  href="/admin/users"
                  className={clsx(
                    "flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    pathname.startsWith("/admin")
                      ? "bg-blue-50 text-blue-700"
                      : "text-slate-500 hover:bg-slate-50 hover:text-slate-900"
                  )}
                >
                  <svg className="w-4 h-4 flex-shrink-0 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                      d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                  </svg>
                  Benutzerverwaltung
                </Link>
              )}
            </div>
          </>
        ) : (
          /* Global nav (not inside an EEG) */
          <div className="px-3 space-y-0.5">
            {[
              {
                href: "/dashboard",
                label: "Dashboard",
                icon: (
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                      d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                  </svg>
                ),
                show: true,
              },
              {
                href: "/eegs",
                label: "Energiegemeinschaften",
                icon: (
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                ),
                show: true,
              },
              {
                href: "/admin/users",
                label: "Benutzerverwaltung",
                icon: (
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                      d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                  </svg>
                ),
                show: isAdmin,
              },
            ]
              .filter((l) => l.show)
              .map((link) => {
                const isActive =
                  link.href === "/dashboard"
                    ? pathname === "/dashboard"
                    : pathname.startsWith(link.href);
                return (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={clsx(
                      "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors",
                      isActive
                        ? "bg-blue-50 text-blue-700"
                        : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                    )}
                  >
                    <span className={clsx(isActive ? "text-blue-700" : "text-slate-400")}>{link.icon}</span>
                    {link.label}
                  </Link>
                );
              })}
          </div>
        )}
      </nav>

      {/* User section */}
      <div className="px-3 py-4 border-t border-slate-200 flex-shrink-0">
        <div className="flex items-center gap-3 px-3 py-2">
          <div className="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
            <span className="text-xs font-semibold text-blue-700">{userInitials}</span>
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-slate-900 truncate">{userName}</p>
            <p className="text-xs text-slate-400">{isAdmin ? "Administrator" : "Benutzer"}</p>
          </div>
        </div>
        <button
          onClick={() => signOut({ callbackUrl: "/auth/signin" })}
          className="mt-1 w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition-colors"
        >
          <svg className="w-5 h-5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          Abmelden
        </button>
      </div>
    </aside>
    </>
  );
}
