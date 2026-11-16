import type { ComponentPropsWithoutRef } from "react";

const paddingClasses = {
  none: "",
  default: "p-6",
  auth: "p-7 max-[760px]:p-6",
};

type CardProps = ComponentPropsWithoutRef<"section"> & {
  as?: "div" | "section";
  padding?: keyof typeof paddingClasses;
};

export default function Card({
  as: Tag = "div",
  padding = "default",
  className = "",
  ...props
}: CardProps) {
  return (
    <Tag
      {...props}
      className={`rounded-[7px] border border-line bg-surface ${paddingClasses[padding]} ${className}`}
    />
  );
}
