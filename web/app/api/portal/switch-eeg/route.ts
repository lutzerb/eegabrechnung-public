import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(request: Request) {
  const sessionToken = (await cookies()).get("portal_session")?.value;
  if (!sessionToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const body = await request.text();
  const res = await fetch(`${API}/api/v1/public/portal/switch-eeg`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Portal-Session": sessionToken,
      "X-Tenant-Host": request.headers.get("host") || "",
    },
    body,
    cache: "no-store",
  });

  const data = await res.json().catch(() => ({}));
  const response = NextResponse.json(data, { status: res.status });
  if (res.ok && data.session_token) {
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
