const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(request: Request) {
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/public/portal/request-link`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return Response.json(await res.json(), { status: res.status });
}
