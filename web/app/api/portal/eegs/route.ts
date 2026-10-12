import { cookies } from "next/headers";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(request: Request) {
  const sessionToken = (await cookies()).get("portal_session")?.value;
  if (!sessionToken) return Response.json({ error: "Unauthorized" }, { status: 401 });

  const res = await fetch(`${API}/api/v1/public/portal/eegs`, {
    headers: {
      "X-Portal-Session": sessionToken,
      "X-Tenant-Host": request.headers.get("host") || "",
    },
    cache: "no-store",
  });
  return Response.json(await res.json(), { status: res.status });
}
