import { auth } from "@/lib/auth";
import { getEEG } from "@/lib/api";

interface Props {
  children: React.ReactNode;
  params: Promise<{ eegId: string }>;
}

export default async function EEGLayout({ children, params }: Props) {
  const { eegId } = await params;
  const session = await auth();

  let isDemo = false;
  if (session?.accessToken) {
    try {
      const eeg = await getEEG(session.accessToken, eegId);
      isDemo = eeg.is_demo ?? false;
    } catch {
      // ignore — page will handle its own error
    }
  }

  return (
    <>
      {isDemo && (
        <div className="bg-amber-400 text-amber-900 text-center text-sm font-semibold py-2 px-4 sticky top-0 z-50 shadow">
          Demo-Modus — Keine E-Mails oder EDA-Nachrichten werden versendet. Daten werden nachts zurückgesetzt.
        </div>
      )}
      {children}
    </>
  );
}
