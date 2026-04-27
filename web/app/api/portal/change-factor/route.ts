import { cookies } from "next/headers";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(req: Request) {
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get("portal_session")?.value;
  if (!sessionToken) return Response.json({ error: "Unauthorized" }, { status: 401 });

  const body = await req.json();
  const res = await fetch(`${API}/api/v1/public/portal/change-factor`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Portal-Session": sessionToken,
    },
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  return Response.json(data, { status: res.status });
}
