import type { ComponentPropsWithoutRef } from "react";
import type { LucideIcon } from "lucide-react";
import { NavLink, type NavLinkProps } from "react-router-dom";

type CommonProps = {
  icon: LucideIcon;
  label: string;
};
type ButtonProps = CommonProps &
  Omit<
    ComponentPropsWithoutRef<"button">,
    "children" | "className" | "aria-label" | "title"
  > & {
    to?: never;
  };
type LinkProps = CommonProps &
  Omit<NavLinkProps, "children" | "className" | "aria-label" | "title">;

const buttonClass =
  "inline-flex size-[34px] shrink-0 cursor-pointer items-center justify-center rounded-[6px] border border-line bg-surface p-0 text-muted transition-colors hover:bg-hover hover:text-ink active:bg-nav-active focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:cursor-not-allowed disabled:opacity-45";

export default function IconButton(props: ButtonProps | LinkProps) {
  if (props.to !== undefined) {
    const { icon: Icon, label, ...linkProps } = props;
    return (
      <NavLink
        {...linkProps}
        className={buttonClass}
        aria-label={label}
        title={label}
      >
        <Icon size={16} strokeWidth={2} aria-hidden="true" />
      </NavLink>
    );
  }
  const { icon: Icon, label, to: _to, type = "button", ...buttonProps } = props;
  return (
    <button
      {...buttonProps}
      type={type}
      className={buttonClass}
      aria-label={label}
      title={label}
    >
      <Icon size={16} strokeWidth={2} aria-hidden="true" />
    </button>
  );
}
