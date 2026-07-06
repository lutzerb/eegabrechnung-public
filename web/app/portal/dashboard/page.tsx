import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import PortalDashboardClient from "./PortalDashboardClient";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

async function portalFetch(path: string, sessionToken: string) {
  const res = await fetch(`${API}${path}`, {
    headers: { "X-Portal-Session": sessionToken },
    cache: "no-store",
  });
  if (!res.ok) return null;
  return res.json();
}

export default async function PortalDashboardPage() {
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get("portal_session")?.value;

  if (!sessionToken) {
    redirect("/portal");
  }

  const [meData, invoicesData, documentsData, meterPointsData] = await Promise.all([
    portalFetch("/api/v1/public/portal/me", sessionToken),
    portalFetch("/api/v1/public/portal/invoices", sessionToken),
    portalFetch("/api/v1/public/portal/documents", sessionToken),
    portalFetch("/api/v1/public/portal/meter-points", sessionToken),
  ]);

  if (!meData) {
    redirect("/portal");
  }

  return (
    <PortalDashboardClient
      member={meData.member}
      eeg={meData.eeg}
      invoices={invoicesData || []}
      documents={documentsData || []}
      meterPoints={meterPointsData || []}
      showFullEnergy={meData.eeg?.portal_show_full_energy !== false}
    />
  );
}
