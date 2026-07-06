"use client";

interface Props {
  action: (formData: FormData) => Promise<void>;
  hiddenName: string;
  hiddenValue: string;
  confirmMessage: string;
  className: string;
  label?: string;
}

export function ConfirmDeleteButton({
  action,
  hiddenName,
  hiddenValue,
  confirmMessage,
  className,
  label = "Löschen",
}: Props) {
  return (
    <form action={action} className="inline">
      <input type="hidden" name={hiddenName} value={hiddenValue} />
      <button
        type="submit"
        className={className}
        onClick={(e) => {
          if (!confirm(confirmMessage)) {
            e.preventDefault();
          }
        }}
      >
        {label}
      </button>
    </form>
  );
}
