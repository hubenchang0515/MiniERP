import type { ReactNode } from "react";
import { X } from "lucide-react";
export default function Badge({
  children,
  onRemove,
  removeLabel = "移除标签",
  disabled = false,
}: {
  children: ReactNode;
  onRemove?: () => void;
  removeLabel?: string;
  disabled?: boolean;
}) {
  return (
    <span className="inline-flex items-center gap-[5px] rounded-[5px] border border-tag-border bg-tag-bg px-2 py-1 text-[12px] leading-[1.5] text-tag-ink">
      {children}
      {onRemove && (
        <button
          type="button"
          disabled={disabled}
          onClick={onRemove}
          aria-label={removeLabel}
          title={removeLabel}
          className="inline-flex size-4 shrink-0 cursor-pointer items-center justify-center rounded text-inherit hover:bg-hover focus-visible:outline-2 focus-visible:outline-accent disabled:cursor-not-allowed disabled:opacity-45"
        >
          <X size={12} aria-hidden="true" />
        </button>
      )}
    </span>
  );
}
