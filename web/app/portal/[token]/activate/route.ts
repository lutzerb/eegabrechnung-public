import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ token: string }> }
) {
  const params = await context.params;
  const { token } = params;

  const proto = request.headers.get("x-forwarded-proto") ?? "http";
  const host =
    request.headers.get("x-forwarded-host") ??
    request.headers.get("host") ??
    "localhost:3001";
  const base = `${proto}://${host}`;

  const res = await fetch(`${API}/api/v1/public/portal/exchange`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
    cache: "no-store",
  });

  if (!res.ok) {
    return NextResponse.redirect(`${base}/portal?error=link-invalid`);
  }

  const data = await res.json();
  const sessionToken: string = data.session_token;

  const response = NextResponse.redirect(`${base}/portal/dashboard`);
  response.cookies.set("portal_session", sessionToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: 24 * 60 * 60,
    path: "/",
  });

  return response;
}
