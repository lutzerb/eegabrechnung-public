"use client";

interface Props {
  action: (formData: FormData) => Promise<void>;
  hiddenName: string;
  hiddenValue: string;
  confirmMessage: string;
  className: string;
  label?: string;
  title?: string;
}

export function ConfirmDeleteButton({
  action,
  hiddenName,
  hiddenValue,
  confirmMessage,
  className,
  label = "Löschen",
  title,
}: Props) {
  return (
    <form action={action} className="inline">
      <input type="hidden" name={hiddenName} value={hiddenValue} />
      <button
        type="submit"
        title={title}
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
