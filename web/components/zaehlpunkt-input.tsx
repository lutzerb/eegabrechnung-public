"use client";

import { ValidatedInput } from "@/components/validated-input";
import { validateZaehlpunkt, NetzbetreiberContext } from "@/lib/validation";

interface Props {
  name: string;
  defaultValue?: string;
  placeholder?: string;
  inputClassName?: string;
  /** Pass null when the Netzbetreiber reference data couldn't be loaded — falls back to format-only validation instead of over-blocking. */
  netzbetreiberContext: NetzbetreiberContext | null;
}

/**
 * Zählpunkt input with full client-side validation (format + Netzbetreiber
 * plausibility), for use from server components — a plain closure can't be
 * passed as ValidatedInput's `validate` prop across the server/client
 * boundary, so this wrapper builds it from a serializable NB context prop.
 */
export function ZaehlpunktInput({
  name,
  defaultValue,
  placeholder,
  inputClassName,
  netzbetreiberContext,
}: Props) {
  return (
    <ValidatedInput
      name={name}
      defaultValue={defaultValue}
      placeholder={placeholder}
      inputClassName={inputClassName}
      validate={(v) => validateZaehlpunkt(v, netzbetreiberContext ?? undefined)}
    />
  );
}
