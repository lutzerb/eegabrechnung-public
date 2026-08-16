import { NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(request: Request) {
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/public/portal/login-password`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      // See request-link/route.ts — the API needs the original public domain to
      // resolve the correct organization for cross-tenant-safe member lookups.
      "X-Tenant-Host": request.headers.get("host") || "",
    },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  const data = await res.json().catch(() => ({}));

  const response = NextResponse.json(data, { status: res.status });

  if (res.ok && typeof data.session_token === "string") {
    response.cookies.set("portal_session", data.session_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 24 * 60 * 60,
      path: "/",
    });
  }

  return response;
}
