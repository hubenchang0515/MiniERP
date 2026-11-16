import type { ReactNode } from "react";
export default function AccountActionRow({
  title,
  description,
  action,
}: {
  title: ReactNode;
  description: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="mt-5 flex items-center justify-between gap-[15px] border-t border-line pt-5 max-[760px]:items-start">
      <div className="min-w-0">
        <strong className="text-[12px] font-medium">{title}</strong>
        <p className="mt-[6px] text-[12px] text-muted">{description}</p>
      </div>
      {action}
    </div>
  );
}
