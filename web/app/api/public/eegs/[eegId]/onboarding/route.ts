const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(
  request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const body = await request.json();
  const ip =
    request.headers.get("x-forwarded-for") ||
    request.headers.get("x-real-ip") ||
    "unknown";

  const res = await fetch(
    `${API}/api/v1/public/eegs/${params.eegId}/onboarding`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Real-IP": ip,
      },
      body: JSON.stringify(body),
    }
  );
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
