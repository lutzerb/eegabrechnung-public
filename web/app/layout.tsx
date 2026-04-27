import type { Metadata } from "next";
import "./globals.css";
import { SessionProvider } from "next-auth/react";
import { auth } from "@/lib/auth";
import Nav from "@/components/nav";

export const metadata: Metadata = {
  title: "EEG Abrechnung",
  description: "Österreichische Energiegemeinschaft Abrechnungsplattform",
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await auth();

  return (
    <html lang="de">
      <body className="min-h-screen bg-slate-50 text-slate-900 antialiased">
        <SessionProvider session={session}>
          <div className="flex min-h-screen">
            {session && <Nav session={session} />}
            <main
              className={`flex-1 ${session ? "sm:ml-64 ml-0 pt-14 sm:pt-0" : ""} flex flex-col`}
            >
              {children}
            </main>
          </div>
        </SessionProvider>
      </body>
    </html>
  );
}
