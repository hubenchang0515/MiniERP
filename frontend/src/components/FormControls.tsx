import {
  useId,
  useState,
  cloneElement,
  type ComponentPropsWithoutRef,
  type ReactNode,
  type ReactElement,
} from "react";
import { Eye, EyeOff } from "lucide-react";

const fieldClass =
  "block w-full rounded-[6px] border border-field-border bg-field px-3 py-[11px] text-ink outline-none transition-colors placeholder:text-[12px] placeholder:text-muted focus:border-accent focus:ring-[3px] focus:ring-accent/10 disabled:bg-subtle disabled:text-muted read-only:bg-subtle";
export function Input({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"input">) {
  return <input {...props} className={`${fieldClass} ${className}`} />;
}
export function Textarea({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"textarea">) {
  return <textarea {...props} className={`${fieldClass} ${className}`} />;
}
export function FormField({
  label,
  required,
  hint,
  error,
  children,
}: {
  label: ReactNode;
  required?: boolean;
  hint?: ReactNode;
  error?: string;
  children: ReactElement<
    ComponentPropsWithoutRef<"input"> | ComponentPropsWithoutRef<"textarea">
  >;
}) {
  const generatedId = useId();
  const id = children.props.id ?? generatedId;
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const describedBy =
    [children.props["aria-describedby"], hint && hintId, error && errorId]
      .filter(Boolean)
      .join(" ") || undefined;
  return (
    <div>
      <label htmlFor={id} className="block text-[13px] font-medium text-ink">
        {label}
        {required && (
          <span className="ml-1 text-error-ink" aria-hidden="true">
            *
          </span>
        )}
      </label>
      <div className="mt-2">
        {cloneElement(children, {
          id,
          "aria-describedby": describedBy,
          "aria-invalid": error ? true : children.props["aria-invalid"],
        })}
      </div>
      {hint && (
        <p id={hintId} className="mt-1 text-[12px] leading-[1.8] text-muted">
          {hint}
        </p>
      )}
      {error && (
        <p
          id={errorId}
          role="alert"
          className="mt-1 text-[12px] text-error-ink"
        >
          {error}
        </p>
      )}
    </div>
  );
}
export function PasswordInput({
  reveal = false,
  className = "",
  disabled,
  ...props
}: Omit<ComponentPropsWithoutRef<"input">, "type"> & { reveal?: boolean }) {
  const [visible, setVisible] = useState(false);
  if (!reveal)
    return (
      <Input
        {...props}
        disabled={disabled}
        type="password"
        className={className}
      />
    );
  return (
    <span className="relative block">
      <Input
        {...props}
        disabled={disabled}
        type={visible ? "text" : "password"}
        className={`pr-[45px] ${className}`}
      />
      <button
        type="button"
        disabled={disabled}
        className="absolute top-1/2 right-[10px] -translate-y-1/2 cursor-pointer rounded p-1 text-muted focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-45"
        aria-label={visible ? "隐藏密码" : "显示密码"}
        onClick={() => setVisible(!visible)}
      >
        {visible ? (
          <EyeOff size={18} aria-hidden="true" />
        ) : (
          <Eye size={18} aria-hidden="true" />
        )}
      </button>
    </span>
  );
}
export function CheckboxItem({
  title,
  description,
  ...props
}: Omit<ComponentPropsWithoutRef<"input">, "type" | "title"> & {
  title: ReactNode;
  description?: ReactNode;
}) {
  return (
    <label className="flex cursor-pointer items-center gap-3 border-b border-line py-[17px] text-[13px] text-ink last:border-0 has-disabled:cursor-not-allowed has-disabled:opacity-60">
      <input
        {...props}
        type="checkbox"
        className="size-4 shrink-0 accent-accent"
      />
      <span className="flex-1">
        <strong className="block text-[12px] font-medium">{title}</strong>
        {description && (
          <span className="mt-1 block text-[12px] leading-[1.8] text-muted">
            {description}
          </span>
        )}
      </span>
    </label>
  );
}
