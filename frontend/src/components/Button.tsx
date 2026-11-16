import type { ComponentPropsWithoutRef } from "react";
import { LoaderCircle } from "lucide-react";
import { Link, type LinkProps } from "react-router-dom";

type CommonProps = {
  variant?: "primary" | "secondary" | "danger" | "text" | "dangerText";
  loading?: boolean;
};
type Props = CommonProps &
  ((ComponentPropsWithoutRef<"button"> & { to?: never }) | LinkProps);
const variants = {
  primary: "border border-primary bg-primary text-white hover:bg-primary-hover",
  secondary: "border border-line bg-surface text-ink hover:bg-hover",
  danger: "border border-danger bg-danger text-white hover:brightness-110",
  text: "text-accent hover:underline",
  dangerText: "text-error-ink hover:underline",
};
export default function Button(props: Props) {
  const {
    variant = "primary",
    loading = false,
    className = "",
    children,
  } = props;
  const text = variant === "text" || variant === "dangerText";
  const classes = `inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 font-medium whitespace-nowrap transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:cursor-not-allowed disabled:opacity-45 ${text ? "gap-[5px] text-[12px]" : "rounded-[7px] px-[17px] py-[10px]"} ${variants[variant]} ${className}`;
  if (props.to !== undefined) {
    const {
      variant: _variant,
      loading: _loading,
      className: _class,
      children: _children,
      ...rest
    } = props;
    return (
      <Link {...rest} className={classes}>
        {children}
      </Link>
    );
  }
  const {
    variant: _variant,
    loading: _loading,
    className: _class,
    children: _children,
    to: _to,
    type = "button",
    disabled,
    ...rest
  } = props;
  return (
    <button
      {...rest}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={classes}
    >
      {loading && (
        <LoaderCircle size={16} className="animate-spin" aria-hidden="true" />
      )}
      {children}
    </button>
  );
}
