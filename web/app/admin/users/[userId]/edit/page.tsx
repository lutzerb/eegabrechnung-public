"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getUser, updateUser, getUserEEGs, setUserEEGs, listEEGs, type AdminUser, type EEG } from "@/lib/api";
import Link from "next/link";

interface Props {
  params: Promise<{ userId: string }>;
  searchParams: Promise<{ success?: string; error?: string }>;
}

export default async function EditUserPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");
  if (session.role !== "admin") redirect("/eegs");

  const { userId } = await params;
  const { success, error: errorParam } = await searchParams;

  let user: AdminUser | null = null;
  let allEEGs: EEG[] = [];
  let assignedEEGIds: string[] = [];
  let error: string | null = null;

  try {
    [user, allEEGs, assignedEEGIds] = await Promise.all([
      getUser(session.accessToken!, userId),
      listEEGs(session.accessToken!),
      getUserEEGs(session.accessToken!, userId),
    ]);
  } catch (err: unknown) {
    error = (err as { message?: string }).message || "Fehler beim Laden.";
  }

  async function updateUserAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session || session.role !== "admin") return;

    const name = formData.get("name") as string;
    const email = formData.get("email") as string;
    const role = formData.get("role") as string;
    const password = (formData.get("password") as string) || undefined;
    const eegIds = formData.getAll("eeg_ids") as string[];

    let saveError: string | null = null;
    try {
      await updateUser(session.accessToken!, userId, { name, email, role, password: password || undefined });
      await setUserEEGs(session.accessToken!, userId, eegIds);
    } catch (err: unknown) {
      saveError = (err as { message?: string }).message || "Speichern fehlgeschlagen.";
    }
    if (saveError) {
      redirect(`/admin/users/${userId}/edit?error=${encodeURIComponent(saveError)}`);
    }
    redirect(`/admin/users/${userId}/edit?success=${encodeURIComponent("Änderungen gespeichert.")}`);
  }

  if (error || !user) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {error || "Benutzer nicht gefunden."}
        </div>
      </div>
    );
  }

  return (
    <div className="p-8 max-w-xl">
      <div className="mb-6">
        <Link href="/admin/users" className="text-sm text-slate-500 hover:text-slate-700">
          ← Benutzerverwaltung
        </Link>
      </div>
      <h1 className="text-2xl font-bold text-slate-900 mb-6">Benutzer bearbeiten</h1>

      {success && (
        <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
          {decodeURIComponent(success)}
        </div>
      )}
      {errorParam && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {decodeURIComponent(errorParam)}
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <form action={updateUserAction} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Name</label>
            <input
              name="name"
              type="text"
              required
              defaultValue={user.name}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">E-Mail</label>
            <input
              name="email"
              type="email"
              required
              defaultValue={user.email}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Neues Passwort <span className="text-slate-400 font-normal">(leer lassen = unverändert)</span>
            </label>
            <input
              name="password"
              type="password"
              minLength={6}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Rolle</label>
            <select
              name="role"
              defaultValue={user.role}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="user">Benutzer</option>
              <option value="admin">Administrator</option>
            </select>
          </div>

          {/* EEG assignments — only relevant for non-admin users */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Energiegemeinschaften
              <span className="ml-1 text-xs text-slate-400 font-normal">
                (Administratoren sehen alle — Auswahl gilt nur für Benutzer-Rolle)
              </span>
            </label>
            <div className="space-y-2 max-h-48 overflow-y-auto border border-slate-200 rounded-lg p-3">
              {allEEGs.length === 0 ? (
                <p className="text-sm text-slate-400">Keine Energiegemeinschaften vorhanden.</p>
              ) : (
                allEEGs.map((eeg) => (
                  <label key={eeg.id} className="flex items-center gap-2.5 cursor-pointer">
                    <input
                      type="checkbox"
                      name="eeg_ids"
                      value={eeg.id}
                      defaultChecked={assignedEEGIds.includes(eeg.id)}
                      className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-sm text-slate-700">{eeg.name}</span>
                    <span className="text-xs text-slate-400">{eeg.gemeinschaft_id}</span>
                  </label>
                ))
              )}
            </div>
          </div>

          <div className="pt-2 flex gap-3">
            <button
              type="submit"
              className="px-5 py-2 text-sm font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 transition-colors"
            >
              Speichern
            </button>
            <Link
              href="/admin/users"
              className="px-5 py-2 text-sm font-medium text-slate-700 bg-slate-100 rounded-lg hover:bg-slate-200 transition-colors"
            >
              Abbrechen
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
