import type { ComponentPropsWithoutRef, ReactNode } from "react";
import Card from "./Card";
export default function FormSection({
  title,
  description,
  children,
  className = "",
  ...props
}: Omit<ComponentPropsWithoutRef<"section">, "title"> & {
  title: ReactNode;
  description?: ReactNode;
}) {
  return (
    <Card as="section" {...props} className={className}>
      <h2 className="mb-2 flex items-center gap-2 text-[15px] font-semibold">
        {title}
      </h2>
      {description && (
        <p className="text-[12px] leading-[1.8] text-muted">{description}</p>
      )}
      {children}
    </Card>
  );
}
