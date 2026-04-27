export default async function PortalActivatePage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-md p-8 max-w-md w-full text-center">
        <div className="mb-6">
          <div className="mx-auto w-14 h-14 bg-green-100 rounded-full flex items-center justify-center mb-4">
            <svg
              className="w-7 h-7 text-green-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
              />
            </svg>
          </div>
          <h1 className="text-xl font-semibold text-gray-900">
            Mitglieder-Portal
          </h1>
          <p className="text-gray-500 mt-2 text-sm">
            Klicken Sie auf den Button, um sich anzumelden.
          </p>
        </div>

        <form action={`/portal/${token}/activate`} method="POST">
          <button
            type="submit"
            className="w-full bg-green-600 text-white py-3 px-6 rounded-lg font-medium hover:bg-green-700 active:bg-green-800 transition-colors"
          >
            Jetzt anmelden
          </button>
        </form>

        <p className="text-xs text-gray-400 mt-5">
          Dieser Link ist 30 Minuten gültig und kann nur einmal verwendet
          werden. Falls er abgelaufen ist, fordern Sie bitte einen neuen Link
          an.
        </p>
      </div>
    </div>
  );
}
