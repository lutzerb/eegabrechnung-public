import { auth } from "@/lib/auth";
import { NextResponse } from "next/server";

export default auth((req) => {
  // Redirect to sign-in if the Go JWT has expired mid-session
  if (req.auth?.error === "AccessTokenExpired") {
    const url = req.nextUrl.clone();
    url.pathname = "/auth/signin";
    url.searchParams.set("callbackUrl", req.nextUrl.pathname);
    return NextResponse.redirect(url);
  }
  // Default auth behavior: unauthenticated -> redirect to /auth/signin
});

export const config = {
  matcher: [
    "/((?!api/auth|api/public|api/portal|onboarding|auth|portal|_next/static|_next/image|favicon.ico).*)",
  ],
};
