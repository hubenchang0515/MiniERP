import type { ReactNode } from "react";
const tones = {
  error: "border-error-border bg-error-bg text-error-ink",
  success: "border-success-border bg-success-bg text-success-ink",
  info: "border-line bg-subtle text-muted",
};
export default function Alert({
  tone = "info",
  children,
}: {
  tone?: keyof typeof tones;
  children: ReactNode;
}) {
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={`my-[14px] rounded-[7px] border px-4 py-3 text-[13px] leading-[1.7] ${tones[tone]}`}
    >
      {children}
    </div>
  );
}
