import { Search } from "lucide-react";
import { Input } from "./FormControls";
import Button from "./Button";
export default function SearchField({
  value,
  label,
  placeholder,
  disabled,
  onChange,
  onSearch,
}: {
  value: string;
  label: string;
  placeholder?: string;
  disabled?: boolean;
  onChange: (value: string) => void;
  onSearch: () => void;
}) {
  return (
    <div
      role="search"
      className="flex min-w-0 max-w-[420px] items-center gap-2"
    >
      <div className="relative min-w-0 flex-1">
        <Search
          size={16}
          aria-hidden="true"
          className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted"
        />
        <Input
          disabled={disabled}
          aria-label={label}
          placeholder={placeholder}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              onSearch();
            }
          }}
          className="pl-9"
        />
      </div>
      <Button
        disabled={disabled}
        type="button"
        onClick={onSearch}
        variant="secondary"
      >
        搜索
      </Button>
    </div>
  );
}
