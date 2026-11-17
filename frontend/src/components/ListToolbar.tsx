import type { ReactNode } from "react";
import SearchField from "./SearchField";

export default function ListToolbar({
  children,
  ...search
}: {
  label: string;
  placeholder?: string;
  value: string;
  disabled?: boolean;
  onChange: (value: string) => void;
  onSearch: () => void;
  children?: ReactNode;
}) {
  return (
    <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
      <SearchField {...search} />
      {children && (
        <div className="ml-auto flex flex-wrap items-center gap-3">
          {children}
        </div>
      )}
    </div>
  );
}
