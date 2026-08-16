import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API_INTERNAL_URL =
  process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

// Streams a sample invoice PDF rendered with the given (not necessarily saved)
// design parameters — used by the live preview in the "Rechnungsdesign" settings
// tab. Nothing is persisted; forwards straight through to the Go endpoint.
export async function POST(
  req: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const { eegId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) {
    return new NextResponse("Unauthorized", { status: 401 });
  }

  const body = await req.text();
  const apiUrl = `${API_INTERNAL_URL}/api/v1/eegs/${eegId}/invoice-design/preview`;

  const res = await fetch(apiUrl, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      "Content-Type": "application/json",
    },
    body,
  });

  if (!res.ok) {
    const text = await res.text();
    return new NextResponse(text || "Vorschau fehlgeschlagen", { status: res.status });
  }

  const blob = await res.blob();
  return new NextResponse(blob, {
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": `inline; filename="Rechnungsdesign_Vorschau.pdf"`,
    },
  });
}
