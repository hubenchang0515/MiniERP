import { ChevronLeft, ChevronRight } from "lucide-react";
import IconButton from "./IconButton";
export default function Pagination({
  page,
  total,
  pageSize = 10,
  disabled,
  onChange,
}: {
  page: number;
  total: number;
  pageSize?: number;
  disabled?: boolean;
  onChange: (page: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="flex items-center justify-between gap-3 border-t border-line px-4 py-3 text-[12px] text-muted">
      <span>
        共 {total} 条 · 第 {page} / {pages} 页
      </span>
      <div className="flex gap-2">
        <IconButton
          icon={ChevronLeft}
          label="上一页"
          disabled={disabled || page <= 1}
          onClick={() => onChange(page - 1)}
        />
        <IconButton
          icon={ChevronRight}
          label="下一页"
          disabled={disabled || page >= pages}
          onClick={() => onChange(page + 1)}
        />
      </div>
    </div>
  );
}
