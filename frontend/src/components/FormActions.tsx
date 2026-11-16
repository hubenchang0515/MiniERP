import type { ComponentPropsWithoutRef } from "react";
export default function FormActions({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"div">) {
  return (
    <div
      {...props}
      className={`col-span-full flex flex-wrap items-center justify-end gap-[10px] ${className}`}
    />
  );
}
