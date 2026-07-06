import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

// Deprecated: anmeldung-online is merged into anmeldung (EC_REQ_ONL).
// This route proxies to /eda/anmeldung for backward compatibility.
export async function POST(
  request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const body = await request.text();
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/eda/anmeldung`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      "Content-Type": "application/json",
    },
    body,
  });
  return Response.json(await res.json(), { status: res.status });
}
