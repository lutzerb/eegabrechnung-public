import { cookies } from "next/headers";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(request: Request) {
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get("portal_session")?.value;
  if (!sessionToken) return Response.json({ error: "Unauthorized" }, { status: 401 });

  const { searchParams } = new URL(request.url);
  const qs = searchParams.toString();
  const res = await fetch(
    `${API}/api/v1/public/portal/energy${qs ? `?${qs}` : ""}`,
    { headers: { "X-Portal-Session": sessionToken }, cache: "no-store" }
  );
  if (!res.ok) return Response.json({ error: "upstream error" }, { status: res.status });
  return Response.json(await res.json(), { status: res.status });
}
