// Resolves an EEG's alias, falling back to its legal name. Use for anything
// shown to users (navigation, lists, emails, portal) — never for legally
// binding documents (invoice Rechnungssteller block, SEPA XML/mandate,
// onboarding declaration), which must keep using `name` directly.
export function eegDisplayName(eeg: { name: string; display_name?: string | null }): string {
  return eeg.display_name?.trim() ? eeg.display_name : eeg.name;
}
