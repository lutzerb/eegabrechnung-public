"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { createUser } from "@/lib/api";
import Link from "next/link";

export default async function NewUserPage() {
  const session = await auth();
  if (!session) redirect("/auth/signin");
  if (session.role !== "admin") redirect("/eegs");

  async function createUserAction(formData: FormData) {
    "use server";
    const session = await auth();
    if (!session || session.role !== "admin") return;

    const name = formData.get("name") as string;
    const email = formData.get("email") as string;
    const password = formData.get("password") as string;
    const role = formData.get("role") as string;

    let saveError: string | null = null;
    let newUserId: string | null = null;
    try {
      const user = await createUser(session.accessToken!, { name, email, password, role });
      newUserId = user.id;
    } catch (err: unknown) {
      saveError = (err as { message?: string }).message || "Anlegen fehlgeschlagen.";
    }
    if (saveError) {
      redirect(`/admin/users/new?error=${encodeURIComponent(saveError)}`);
    }
    redirect(`/admin/users/${newUserId}/edit?success=${encodeURIComponent("Benutzer erfolgreich angelegt.")}`);
  }

  return (
    <div className="p-8 max-w-xl">
      <div className="mb-6">
        <Link href="/admin/users" className="text-sm text-slate-500 hover:text-slate-700">
          ← Benutzerverwaltung
        </Link>
      </div>
      <h1 className="text-2xl font-bold text-slate-900 mb-6">Neuer Benutzer</h1>

      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <form action={createUserAction} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Name</label>
            <input
              name="name"
              type="text"
              required
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">E-Mail</label>
            <input
              name="email"
              type="email"
              required
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Passwort</label>
            <input
              name="password"
              type="password"
              required
              minLength={6}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Rolle</label>
            <select
              name="role"
              defaultValue="user"
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="user">Benutzer</option>
              <option value="admin">Administrator</option>
            </select>
          </div>
          <div className="pt-2 flex gap-3">
            <button
              type="submit"
              className="px-5 py-2 text-sm font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 transition-colors"
            >
              Anlegen
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
