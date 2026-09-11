import { auth } from "@/lib/auth";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: Request,
  context: { params: Promise<{ eegId: string; meterPointId: string }> }
) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken)
    return Response.json({ error: "Unauthorized" }, { status: 401 });

  const url = new URL(request.url);
  const qs = new URLSearchParams();
  for (const key of ["from", "to", "limit", "offset"]) {
    const value = url.searchParams.get(key);
    if (value) qs.set(key, value);
  }

  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/meter-points/${params.meterPointId}/readings?${qs.toString()}`,
    { headers: { Authorization: `Bearer ${session.accessToken}` } }
  );
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
