const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(request: Request) {
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/public/portal/request-link`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      // The Go API's own inbound Host header always reflects the internal Docker
      // service name (e.g. eegabrechnung-api:8080), not the domain the browser
      // used — pass it through explicitly so the API can resolve the correct
      // organization for cross-tenant-safe member lookups.
      "X-Tenant-Host": request.headers.get("host") || "",
    },
    body: JSON.stringify(body),
  });
  return Response.json(await res.json(), { status: res.status });
}
