import { cookies } from "next/headers";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET() {
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get("portal_session")?.value;
  if (!sessionToken) return Response.json({ error: "Unauthorized" }, { status: 401 });

  const res = await fetch(`${API}/api/v1/public/portal/meter-points`, {
    headers: { "X-Portal-Session": sessionToken },
    cache: "no-store",
  });
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
