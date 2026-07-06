import { cookies } from "next/headers";

export async function POST() {
  const cookieStore = await cookies();
  cookieStore.delete("portal_session");
  return Response.json({ ok: true });
}
