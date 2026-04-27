"use client";

import { useState, useRef } from "react";
import { validateIBAN, validateBIC, validateZaehlpunkt, validateSepaCreditorId, validateUIDNummer, formatIBAN } from "@/lib/validation";

const VALIDATORS: Record<string, (value: string) => string | null> = {
  iban: validateIBAN,
  bic: validateBIC,
  zaehlpunkt: validateZaehlpunkt,
  sepa_creditor_id: validateSepaCreditorId,
  uid_nummer: validateUIDNummer,
};

interface ValidatedInputProps {
  name: string;
  defaultValue?: string;
  placeholder?: string;
  /** Pass a validator function (from client components) OR a validatorName string (from server components). */
  validate?: (value: string) => string | null;
  /** Name of a built-in validator — use this from server components instead of the `validate` function prop. */
  validatorName?: keyof typeof VALIDATORS;
  /** Tailwind classes forwarded to the <input> element. */
  inputClassName?: string;
}

/**
 * Drop-in replacement for <input> that shows an inline error on blur
 * and sets the browser's native custom validity (blocks form submission
 * when the field is invalid, even in server-action forms).
 *
 * For IBAN fields: auto-formats with spaces every 4 characters while typing.
 * A hidden input carries the clean (no-spaces) value for form submission.
 */
export function ValidatedInput({
  name,
  defaultValue = "",
  placeholder,
  validate,
  validatorName,
  inputClassName = "",
}: ValidatedInputProps) {
  const isIban = validatorName === "iban";
  const validateFn = validate ?? (validatorName ? VALIDATORS[validatorName] : null);

  // For IBAN: store the formatted (with spaces) value; hidden input holds the clean value.
  const [value, setValue] = useState(isIban ? formatIBAN(defaultValue) : defaultValue);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  function clean(val: string) {
    return isIban ? val.replace(/\s/g, "").toUpperCase() : val;
  }

  function check(val: string) {
    const err = validateFn ? validateFn(clean(val)) : null;
    setError(err);
    inputRef.current?.setCustomValidity(err ?? "");
    return err;
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    let v = e.target.value;
    if (isIban) {
      // Strip all spaces/non-alphanumeric, uppercase, then reformat in groups of 4
      const stripped = v.replace(/[^A-Za-z0-9]/g, "").toUpperCase();
      v = stripped.match(/.{1,4}/g)?.join(" ") ?? stripped;
    }
    setValue(v);
    if (error !== null) check(v);
  }

  const cleanValue = clean(value);

  return (
    <>
      {/* Hidden input carries the clean value (no spaces) for form submission */}
      {isIban && <input type="hidden" name={name} value={cleanValue} />}
      <input
        ref={inputRef}
        name={isIban ? undefined : name}
        value={value}
        placeholder={placeholder}
        className={`${inputClassName} ${error ? "!border-red-400 focus:!ring-red-400" : ""}`}
        onChange={handleChange}
        onBlur={(e) => check(e.target.value)}
        onInvalid={(e) => e.preventDefault()}
      />
      {error && <p className="text-xs text-red-600 mt-1">{error}</p>}
    </>
  );
}
