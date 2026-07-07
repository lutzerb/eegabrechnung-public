const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(req: Request) {
  const { token } = await req.json();
  if (!token) return Response.json({ error: "invalid request" }, { status: 400 });

  const res = await fetch(`${API}/api/v1/public/portal/email-change/confirm/${encodeURIComponent(token)}`, {
    method: "POST",
  });
  const data = await res.json().catch(() => ({}));
  return Response.json(data, { status: res.status });
}
