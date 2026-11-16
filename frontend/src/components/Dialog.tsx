import FormActions from "./FormActions";
import { useId, useLayoutEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import Button from "./Button";
import Alert from "./Alert";

export function Dialog({
  title,
  description,
  children,
  busy = false,
  onClose,
}: {
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  busy?: boolean;
  onClose: () => void;
}) {
  const titleId = useId();
  const descriptionId = useId();
  const panel = useRef<HTMLDivElement>(null);
  const state = useRef({ busy, onClose });
  useLayoutEffect(() => {
    state.current = { busy, onClose };
  });
  useLayoutEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const focusable = () =>
      Array.from(
        panel.current?.querySelectorAll<HTMLElement>(
          'button:not(:disabled), a[href], input:not(:disabled), textarea:not(:disabled), select:not(:disabled), [tabindex="0"]',
        ) ?? [],
      );
    const focusFirst = () => (focusable()[0] ?? panel.current)?.focus();
    if (!panel.current?.contains(document.activeElement)) focusFirst();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        if (!state.current.busy) state.current.onClose();
      }
      if (event.key === "Tab") {
        const elements = focusable();
        const first = elements[0],
          last = elements.at(-1);
        if (!first) {
          event.preventDefault();
          panel.current?.focus();
        } else if (
          event.shiftKey &&
          (document.activeElement === first ||
            document.activeElement === panel.current)
        ) {
          event.preventDefault();
          last?.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first.focus();
        }
      }
    };
    const onFocus = (event: FocusEvent) => {
      if (!panel.current?.contains(event.target as Node)) focusFirst();
    };
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("focusin", onFocus);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("focusin", onFocus);
      document.body.style.overflow = overflow;
      if (previous?.isConnected) previous.focus();
    };
  }, []);
  return createPortal(
    <div
      className="fixed inset-0 z-50 grid place-items-center overflow-y-auto bg-black/40 p-[22px] backdrop-blur-[2px] dark:bg-black/65"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !busy) onClose();
      }}
    >
      <div
        ref={panel}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        aria-busy={busy || undefined}
        tabIndex={-1}
        className="max-h-[calc(100dvh-44px)] w-full max-w-[460px] overflow-y-auto rounded-xl border border-line bg-surface p-7 shadow-2xl outline-none"
      >
        <h2 id={titleId} className="mb-[13px] text-[19px] font-semibold">
          {title}
        </h2>
        {description && (
          <p
            id={descriptionId}
            className="mb-5 text-[13px] leading-[1.9] text-muted"
          >
            {description}
          </p>
        )}
        {children}
      </div>
    </div>,
    document.body,
  );
}
export function ConfirmDialog({
  title,
  description,
  busy,
  error,
  confirmLabel = "确认操作",
  onConfirm,
  onClose,
}: {
  title: ReactNode;
  description?: ReactNode;
  busy?: boolean;
  error?: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={title}
      description={description}
      busy={busy}
      onClose={onClose}
    >
      {error && <Alert tone="error">{error}</Alert>}
      <FormActions className="mt-6">
        <Button variant="secondary" disabled={busy} onClick={onClose}>
          取消
        </Button>
        <Button variant="danger" loading={busy} onClick={onConfirm}>
          {busy ? "正在处理…" : confirmLabel}
        </Button>
      </FormActions>
    </Dialog>
  );
}
