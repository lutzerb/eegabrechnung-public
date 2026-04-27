"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { listUsers, deleteUser, type AdminUser } from "@/lib/api";
import Link from "next/link";

interface Props {
  searchParams: Promise<{ success?: string; error?: string }>;
}

export default async function AdminUsersPage({ searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");
  if (session.role !== "admin") redirect("/eegs");
  const { success: spSuccess, error: spError } = await searchParams;

  let users: AdminUser[] = [];
  let error: string | null = null;

  try {
    users = await listUsers(session.accessToken!);
  } catch (err: unknown) {
    error = (err as { message?: string }).message || "Fehler beim Laden der Benutzer.";
  }

  async function deleteUserAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session || session.role !== "admin") return;
    const userId = formData.get("userId") as string;
    try {
      await deleteUser(session.accessToken!, userId);
      revalidatePath("/admin/users");
    } catch (err: unknown) {
      const msg = (err as { message?: string }).message || "Löschen fehlgeschlagen.";
      redirect(`/admin/users?error=${encodeURIComponent(msg)}`);
    }
  }

  return (
    <div className="p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Benutzerverwaltung</h1>
          <p className="text-slate-500 mt-1">Benutzer anlegen, bearbeiten und Zugänge verwalten.</p>
        </div>
        <Link
          href="/admin/users/new"
          className="px-4 py-2 text-sm font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 transition-colors"
        >
          + Neuer Benutzer
        </Link>
      </div>

      {spSuccess && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
          {decodeURIComponent(spSuccess)}
        </div>
      )}
      {(error || spError) && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {error || decodeURIComponent(spError!)}
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        {users.length === 0 ? (
          <div className="px-6 py-16 text-center text-slate-400 text-sm">
            Keine Benutzer vorhanden.
          </div>
        ) : (
          <div className="overflow-x-auto">
          <table className="w-full text-sm min-w-[500px]">
            <thead>
              <tr className="bg-slate-50 border-b border-slate-100">
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">Name</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">E-Mail</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">Rolle</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">Erstellt</th>
                <th className="text-right px-5 py-3 text-xs font-medium text-slate-500">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-slate-50 transition-colors">
                  <td className="px-5 py-3 font-medium text-slate-900">{user.name}</td>
                  <td className="px-5 py-3 text-slate-600">{user.email}</td>
                  <td className="px-5 py-3">
                    <span className={`text-xs px-2 py-0.5 rounded font-medium ${
                      user.role === "admin"
                        ? "bg-violet-50 text-violet-700"
                        : "bg-slate-100 text-slate-600"
                    }`}>
                      {user.role === "admin" ? "Administrator" : "Benutzer"}
                    </span>
                  </td>
                  <td className="px-5 py-3 text-slate-500">
                    {new Date(user.created_at).toLocaleDateString("de-AT")}
                  </td>
                  <td className="px-5 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <Link
                        href={`/admin/users/${user.id}/edit`}
                        className="px-2.5 py-1 text-xs font-medium text-slate-700 bg-slate-100 rounded hover:bg-slate-200 transition-colors"
                      >
                        Bearbeiten
                      </Link>
                      <form action={deleteUserAction}>
                        <input type="hidden" name="userId" value={user.id} />
                        <button
                          type="submit"
                          className="px-2.5 py-1 text-xs font-medium text-red-700 bg-red-50 rounded hover:bg-red-100 transition-colors"
                        >
                          Löschen
                        </button>
                      </form>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          </div>
        )}
      </div>
    </div>
  );
}
