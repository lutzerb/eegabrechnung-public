const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

// Public endpoint — no auth required
export async function POST(
  request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const body = await request.json();

  const res = await fetch(
    `${API}/api/v1/public/eegs/${params.eegId}/onboarding/verify-email`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }
  );

  const data = await res.json().catch(() => ({}));
  return Response.json(data, { status: res.status });
}
