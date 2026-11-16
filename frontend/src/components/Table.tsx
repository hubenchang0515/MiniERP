import type { ComponentPropsWithoutRef } from "react";
export function Table({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"table">) {
  return (
    <div className="overflow-auto">
      <table
        {...props}
        className={`w-full border-collapse text-left text-[13px] whitespace-nowrap ${className}`}
      />
    </div>
  );
}
export function TableHead({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"thead">) {
  return (
    <thead
      {...props}
      className={`border-y border-line bg-table-head text-[10px] text-muted ${className}`}
    />
  );
}
export function TableBody({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"tbody">) {
  return (
    <tbody
      {...props}
      className={`[&>tr:last-child>td]:border-b-0 [&>tr:hover]:bg-row-hover ${className}`}
    />
  );
}
export function TableHeader({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"th">) {
  return (
    <th
      {...props}
      className={`px-5 py-3 font-medium max-[760px]:px-[14px] ${className}`}
    />
  );
}
export function TableCell({
  className = "",
  ...props
}: ComponentPropsWithoutRef<"td">) {
  return (
    <td
      {...props}
      className={`border-b border-line px-5 py-[14px] max-[760px]:px-[14px] ${className}`}
    />
  );
}
