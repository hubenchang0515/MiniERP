import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
export default function ResultState({
  icon: Icon,
  title,
  description,
  actions,
}: {
  icon?: LucideIcon;
  title?: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="flex min-h-[70vh] flex-col items-center justify-center gap-[17px] p-[30px] text-center">
      {Icon && <Icon size={48} className="text-muted" aria-hidden="true" />}
      {title && (
        <h1 className="text-[24px] leading-[1.5] font-semibold tracking-[-0.8px]">
          {title}
        </h1>
      )}
      {description && <div className="text-muted">{description}</div>}
      {actions && (
        <div className="flex flex-wrap justify-center gap-[10px]">
          {actions}
        </div>
      )}
    </div>
  );
}
