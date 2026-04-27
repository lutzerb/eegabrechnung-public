const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _req: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const res = await fetch(`${API}/api/v1/public/eegs/${params.eegId}/info`, {
    cache: "no-store",
  });
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
