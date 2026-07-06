const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(
  _req: Request,
  context: { params: Promise<{ token: string }> }
) {
  const params = await context.params;
  const res = await fetch(
    `${API}/api/v1/public/onboarding/confirm/${params.token}`,
    { method: "POST", cache: "no-store" }
  );
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
